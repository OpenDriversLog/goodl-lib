// This package is responsible for processing raw GPSData and generating TrackRecords & Trips from it.
package datapolish

import (
	"database/sql"
	"errors"
	"fmt"
	geo "github.com/kellydunn/golang-geo"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/addressManager"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/deviceManager"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/tripMan"
	. "github.com/OpenDriversLog/goodl-lib/models/SQLite"
	. "github.com/OpenDriversLog/goodl-lib/models/geo"
	"github.com/OpenDriversLog/goodl-lib/tools"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/carManager"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/notificationManager"
	"github.com/OpenDriversLog/goodl-lib/translate"
)

const pdTag = "glib/dp/processData.go"

var TrackRecordData []Location

func init() {
	// if not using goodl-lib from goodl, we have to register SQLITE
	tools.RegisterSqlite("SQLITE")
}

var ErrGpsDataAlreadyImported = errors.New("ProcessGPSData: KeyPoints & Tracks already imported!")
// ProcessGPSData does all the processing from trackRecords to Tracks, TrackPoints and KeyPoints
// where maxTime is optional
func ProcessGPSData(startTime int64, endTime int64, deviceId int, recalculate bool, uId int64,activeNotifications *[]*notificationManager.Notification,T *translate.Translater, dbCon *sql.DB) (err error) {

	var device *deviceManager.Device
	devices, err := deviceManager.GetDevices(dbCon)
	if err != nil {
		dbg.E(pdTag, "Error getting devices : ", err)
	}
	for _, v := range devices {
		if int(v.Id) == deviceId {
			device = v
		}
	}
	if device == nil {
		dbg.E(pdTag, "Could not find device with id : %d", deviceId)
		err = errors.New("Unknown device!")
		return
	}
	carId := device.CarId
	config := GetDefaultLocationConfig()
	car, err := carManager.GetCarById(dbCon,int64(carId))
	if err != nil {
		dbg.E(pdTag,"Error getting car for device %d with carId %d: ",device.Id,carId,err)
		return
	}
	if car.Owner.Id<0 {
		dbg.E(pdTag,"No carOwner defined for car %d ", car.Id)
		return errors.New("Empty car owner")
	}
	driverId := int64(car.Owner.Id)
	// var startKeyPointId int64
	var kpEnd *KeyPoint
	var countNewTracks int64
	var countNewKPs int
	var countNewTPs int
	createFirst := false

	if endTime == 0 || endTime <= startTime {
		_, max, err2 := GetDeviceTimeRange(deviceId, dbCon)
		if err2 != nil {
			dbg.W(pdTag, "Failed to get min/max time for device...setting to maxInt64")
			endTime = math.MaxInt64
		} else {
			endTime = max
		}
	}

	// 1.read a devices data from the database starting from startTime
	trackrecords, err := GetTrackRecordsForDevice(startTime, endTime, deviceId, dbCon)
	if len(trackrecords) <= 0 {
		dbg.W(pdTag, "no trackRecords for this device : %v (%s), aborting ProcessGPSData ", deviceId, device.Description, startTime, endTime, (endTime-startTime)/1000, err)
		trips, err := tripMan.GetTripsInTimeRange(startTime, endTime, []interface{}{deviceId}, false, false, false,uId,activeNotifications,T,true, dbCon)
		if err != nil {
			dbg.E(pdTag, "Error getting trips", err)
			return err
		}
		if len(trips) != 0 {
			dbg.WTF(pdTag, "How can we have trips without trackrecords for device %d in timerange from %d to %d??? Will delete them", deviceId, startTime, endTime)
			for _, t := range trips {
				_, err = dbCon.Exec("DELETE FROM Tracks_Trips WHERE tripId=?;DELETE FROM Trips WHERE _tripId=?", t.Id, t.Id)
				if err != nil {
					dbg.E(pdTag, "Error deleting trip %d : ", t.Id, err)
				}
			}
		}
		return nil
	}

	tx, err := dbCon.Begin()
	if err != nil {
		dbg.E(pdTag, "Error starting transaction : ", err)
		return
	}
	// 2.find all the keypoints
	// check if there are already KeyPoints from this device in the given time
	prevKP := KeyPoint{}

	// look up DB if a very similar point already exists, could not be true if we're going to create the first track for this device
	// TODO: CS use crossplattform DB stuff
	row := dbCon.QueryRow("SELECT _keyPointId, latitude, longitude, starttime, previoustrackid, nexttrackid FROM `keyPoints` WHERE (deviceId=? AND startTime > ? AND endTime < ?) ORDER BY starttime DESC LIMIT 1", deviceId, startTime, endTime)
	err = row.Scan(&prevKP.KeyPointId, &prevKP.Latitude, &prevKP.Longitude, &prevKP.StartTime, &prevKP.PreviousTrackId, &prevKP.NextTrackId)

	if err == sql.ErrNoRows {
		dbg.W(pdTag, "no valid previous keypoint found... going to createFirst... ", err)
		createFirst = true
	} else if err != nil {
		dbg.E(pdTag, "Failed to get kp id count in time range...", err)
		return err
	} else {
		dbg.I(pdTag, "got previous keypoint...", prevKP)
		createFirst = false
	}
	if prevKP.StartTime.Int64 >= startTime {
		// Ok, we got a keypoint in the imported time range
		if recalculate {
			dbg.I(pdTag, "Recalculate active & there are alreadyKPs in time range since  %d (%d) (until %d) for deviceId %d...", startTime, prevKP.StartTime.Int64, endTime, deviceId)
			trackIds, err := tripMan.GetTrackIdsInTimeRange(startTime, endTime, []interface{}{int64(deviceId)}, dbCon)

			for _, tId := range trackIds {
				_, err := dbCon.Exec(`
					DELETE FROM KeyPoints WHERE _keyPointId IN (
SELECT endKeyPointId FROM Tracks WHERE _trackId=? AND
(
SELECT COUNT(_trackId) FROM Tracks B WHERE Tracks.endKeyPointId=B.startKeyPointId
) = 0
);
DELETE FROM trackPoints WHERE trackId=?;
DELETE FROM Tracks_Trips WHERE trackId=?;
DELETE FROM Tracks WHERE _trackId=?;
				`, tId, tId, tId, tId)
				if err != nil {
					dbg.E(pdTag, " Error while deleting trackId %d", tId)
					return err
				}

			}

			// Determine new start- and endTime by first keyPoint before deleted tracks and first keyPoint after deleted Tracks
			var newStartTime sql.NullInt64
			err = dbCon.QueryRow(`SELECT endTime FROM KeyPoints LEFT JOIN Tracks ON startKeyPointId=_keyPointId
 WHERE endTime<=? AND startKeyPointId IS NOT NULL ORDER BY endTime DESC LIMIT 1`, startTime).Scan(&newStartTime)
			if err != nil && err != sql.ErrNoRows {
				dbg.E(pdTag, "Error querying new start time : ", err)
				return err
			}
			if newStartTime.Valid {
				startTime = newStartTime.Int64
			} else {
				dbg.I(pdTag, "No keypoint found before ", startTime)
			}

			var newEndTime sql.NullInt64
			err = dbCon.QueryRow(`SELECT startTime FROM KeyPoints LEFT JOIN Tracks ON endKeyPointId=_keyPointId
 WHERE startTime>=? AND endKeyPointId IS NOT NULL ORDER BY startTime ASC LIMIT 1`, endTime).Scan(&newEndTime)
			if err != nil && err != sql.ErrNoRows {
				dbg.E(pdTag, "Error querying new end time : ", err)
				return err
			}
			if newEndTime.Valid {
				endTime = newEndTime.Int64
			} else {
				dbg.I(pdTag, "No keypoint found after ", endTime)
			}
		} else {
			dbg.W(pdTag, "Recalculate NOT active AND there are alreadyKPs in time range since  %d (%d) for deviceId %d...", startTime, prevKP.StartTime.Int64, deviceId)
			return ErrGpsDataAlreadyImported

		}

	}
	dbg.I(pdTag, "Reget trackrecords for device")
	trackrecords, err = GetTrackRecordsForDevice(startTime, endTime, deviceId, dbCon)
	dbg.I(pdTag, "Call FindKeyPoints")
	kps, err := FindKeyPoints(dbCon, trackrecords, nil,uId)

	dbg.I(pdTag, "ProcessGPSData: found %d keypoints in %d trackrecords...", len(kps), len(trackrecords))
	p := geo.NewPoint((kps)[0].Latitude.Float64, (kps)[0].Longitude.Float64)
	lastPoint := geo.NewPoint(prevKP.Latitude.Float64, prevKP.Longitude.Float64)

	var newTrackId sql.NullInt64
	var startKpId int64
	addedKps := make([]int64, 0)
	defer func() { // Last step : For all added KeyPoints insert GeoZone-information
		if len(addedKps) != 0 {
			err = addressManager.UpdateKeyPointsForAllGeozones(addedKps, dbCon)
			if err != nil {
				dbg.E(pdTag, "Error updating KeyPoints for all GeoZones : ", err)
			}
		}
	}()
	if p.GreatCircleDistance(lastPoint)*1000 > float64(config.MinMoveDist) || createFirst {
		newTrackId.Int64 = -1
		res, errKp := dbCon.Exec("INSERT INTO `keyPoints` (latitude, longitude, startTime, endTime, previousTrackId, nextTrackId, deviceId, addressId,carId) VALUES(?,?,?,?,?,?,?,?,?)",
			(kps)[0].Latitude, (kps)[0].Longitude, (kps)[0].StartTime, (kps)[0].EndTime, (kps)[0].PreviousTrackId, newTrackId, deviceId, (kps)[0].AddressId, carId)

		if errKp != nil {
			dbg.E(pdTag, "Failed to insert KeyPoint into DB", errKp)
			return errKp
		}

		startKpId, _ = res.LastInsertId()
		addedKps = append(addedKps, startKpId)
		// If we experience performance issues we could first bundle all KeyPoints and then do this.
		errKp = addressManager.UpdateKeyPointsForAllGeozones([]int64{startKpId}, dbCon)
		if err != nil {
			dbg.E(pdTag, "Failed to calculate GeoZones for KeyPoint", errKp)
			return errKp
		}
	} else { // last point we got seems not to be starting KeyPoint of this track
		(kps)[0].PreviousTrackId = prevKP.PreviousTrackId
		(kps)[0].StartTime = prevKP.StartTime
		startKpId = prevKP.KeyPointId.Int64
		_, errKp := dbCon.Exec("UPDATE `keyPoints` SET endTime=? WHERE _keyPointId=?", (kps)[0].EndTime, startKpId)
		if errKp != nil {
			dbg.E(pdTag, "Failed to insert KeyPoint into DB", errKp)
			return errKp
		}
	}

	// 3. creates a new track for each new KeyPoint
	for idx := 1; idx < len(kps); idx++ { // skip 1st, we already handled it
		kpEnd = (kps)[idx]
		// TODO: CS use crossplattform DB stuff
		resEndKP, errKp2 := dbCon.Exec("INSERT INTO `keyPoints` (latitude, longitude, startTime, endTime, previousTrackId, nextTrackId, deviceId, addressId) VALUES(?,?,?,?,?,?,?,?)",
			kpEnd.Latitude, kpEnd.Longitude, kpEnd.StartTime, kpEnd.EndTime, kpEnd.PreviousTrackId, sql.NullInt64{Int64: 0, Valid: false}, deviceId, kpEnd.AddressId)
		if errKp2 != nil {
			dbg.E(pdTag, "Failed to insert KeyPoint into DB", errKp2)
			return errKp2
		}

		endKpId, _ := resEndKP.LastInsertId()
		addedKps = append(addedKps, endKpId)
		// dbg.V(pdTag, "inserted KeyPoints %d + %d into DB...doing Tracks now", startKpId, endKpId, idx, idx-1)
		countNewKPs += 1

		// TODO: CS use crossplattform DB stuff
		resTrack, errTrack := dbCon.Exec("INSERT INTO `tracks` (deviceId, startKeyPointId, endKeyPointId, distance,carId) VALUES(?, ?, ?, -1,?)",
			deviceId, startKpId, endKpId, carId)
		if errTrack != nil {
			dbg.E(pdTag, "Failed to insert track into DB", errTrack)
			return errTrack
		}

		newTrackId, _ := resTrack.LastInsertId()
		countNewTracks += 1

		// TODO: CS use crossplattform DB stuff
		_, errUpNextTId := dbCon.Exec("UPDATE `keyPoints` SET nextTrackId=? WHERE _keyPointId=?", newTrackId, startKpId)
		if errUpNextTId != nil {
			dbg.E(pdTag, "Failed to update NextTrackId of StartKeyPoint", errUpNextTId)
			return errUpNextTId
		}

		// TODO: CS use crossplattform DB stuff
		_, errUpPrevKpId := dbCon.Exec("UPDATE `keyPoints` SET previousTrackId=? WHERE _keyPointId=?", newTrackId, endKpId)
		if errUpPrevKpId != nil {
			dbg.E(pdTag, "Failed to update PreviousTrackId of EndKeyPoint", errUpPrevKpId)
			return errUpPrevKpId
		}

		// 4. create filtered Trackpoints for each track
		newTPs, err := CreateFilteredTrackPoints(newTrackId, nil, dbCon)
		if err != nil {
			dbg.E(pdTag, "Failed to createFilteredTrackPoint for Track %d", newTrackId)
			return err
		}
		countNewTPs += len(newTPs)

		// 5. create default Trip for this Track
		// TODO: implement
		// tracks = []int
		// tracks = append(tracks, newTrackId)
		tripTracks := []int64{newTrackId}
		_, _, _, err = tripMan.CreateOrReviveTripByTracks(tripTracks, 0, "", "",driverId, -1, false, false, false,activeNotifications,T, dbCon)

		startKpId = endKpId
	} // for range kps ~ create track for each KP

	dbg.I(pdTag, "ProcessGPSData: inserted %d KeyPoints, %d Tracks and %d TrackPoints...", countNewKPs, countNewTracks, countNewTPs)

	err = tx.Commit()
	if err != nil {
		dbg.E(pdTag, "Error commiting transaction : ", err)
	}
	// TODO: maybe refactor: move DB index inserts/updates to extra function
	return nil
}

type AddressError struct {
	Error    error
	KeyPoint *KeyPoint
}

// FindKeyPoints analyzes a list of Locations to find KeyPoints to insert into KeyPointsTable
// expects array of Location for a specific device
// locations should be sorted by time and are from the same device
func FindKeyPoints(dbCon *sql.DB, raw []Location, config *LocationConfig, uId int64) ([]*KeyPoint, error) {
	cKpForAddresses := make(chan *KeyPoint)
	client := &http.Client{
		Timeout: time.Duration(10 * time.Second),
	}
	done := make(chan struct{})
	errs := make([]AddressError, 0)
	var wg sync.WaitGroup
	defer close(done)
	go func() {
		defer close(cKpForAddresses)
		defer func() {
			if err := recover(); err != nil {
				dbg.WTF(pdTag, "Fatal error finding keypoint addresses:", err)
				dbg.E(pdTag, "Fatal error finding keypoint addresses:", err)
			}
		}()
		var err error

		reqCount := 0 // TODO: REMOVE!
		for {
			select {
			case kp := <-cKpForAddresses:
				{
					var addrId int64 = -1
					reqCount++
					/*	if reqCount>1 { // TODO: Remove
						wg.Done()
						continue
					}*/
					addrId, err = addressManager.GetAddressIdForLatLng(kp.Latitude.Float64, kp.Longitude.Float64, client,uId, dbCon)
					if err != nil {

						dbg.W(pdTag, "Error getting addressId for [%d, %d]", kp.Latitude.Float64, kp.Longitude.Float64, err)
						errs = append(errs, AddressError{Error: err, KeyPoint: kp})
						if err != nil {
							dbg.E(pdTag, "Error creating unknown address - this is not good!!", err)
						}

					}
					kp.AddressId = sql.NullInt64{Int64: addrId, Valid: true}

					wg.Done()
				}
			case <-done:
				{
					return
				}
			}
		}
	}()
	keyPoints := make([]*KeyPoint, 0)
	candidates := make([]Location, 0)
	var p *geo.Point
	var lastLocation Location
	var currentLocation Location
	isFirst := true
	inaccuratePoints := 0 // not sure what we could use this for...

	if config == nil {
		config = GetDefaultLocationConfig()
	}

	if raw == nil || len(raw) < 1 {
		dbg.E(pdTag, "FindKeyPoints: got no correct locations-array", raw)
		return nil, errors.New("FindKeyPoints: got no correct locations-array")
	}

	// dbg.I(pdTag, "fkp..running through %d locations...", len(*raw))

	// set up first Points to compare
	currentLocation = (raw)[0]
	lastLocation = currentLocation
	var nextLocation *Location
	p = geo.NewPoint(0, 0)
	lastPoint := p

	// assume 1st point is candidate for keyPoint (because we started here)...
	candidates = append(candidates, raw[0])
	pointCount := len(raw)

	for idx := 1; idx < pointCount; idx++ { // start going through the raw points
		currentLocation = (raw)[idx]

		if pointCount == idx+1 {
			nextLocation = nil
		} else {
			nextLocation = &(raw[idx+1])
		}
		if currentLocation.Accuracy.Float64 > float64(config.AccuracyThreshold) {
			inaccuratePoints++
			continue // dont worry about inaccurate points at all
		}

		// check distance
		p = geo.NewPoint(currentLocation.Latitude.Float64, currentLocation.Longitude.Float64)
		// avgPointFromCandidates := interpolateLocations(&candidates)
		// lastPoint = geo.NewPoint(candidates[0].Latitude.Float64, candidates[0].Longitude.Float64)
		lastPoint = geo.NewPoint(lastLocation.Latitude.Float64, lastLocation.Longitude.Float64)

		// dbg.E(pdTag, "dist was %f at %d", lastPoint.GreatCircleDistance(p)*1000, idx)

		// dist := p.GreatCircleDistance(avgPointFromCandidates) * 1000
		dist := p.GreatCircleDistance(lastPoint) * 1000

		// if float64(config.MinMoveDist) < dist && currentLocation.Accuracy.Float64 < dist && lastLocation.Accuracy.Float64 < dist { // we moved
		if float64(config.MinMoveDist) < dist &&
			currentLocation.Accuracy.Float64 < dist && lastLocation.Accuracy.Float64 < dist &&
			(nextLocation == nil ||
				(lastPoint.GreatCircleDistance(geo.NewPoint(nextLocation.Latitude.Float64, nextLocation.Longitude.Float64))*1000 > float64(config.MinMoveDist) &&
					currentLocation.TimeMillis.Int64-nextLocation.TimeMillis.Int64 < 1000*60)) { // we moved and did not move back again in the next point (if during one minute)
			if len(candidates) > 0 {

				// create a keypoint from the candidates

				// if int64(config.MinMoveTime) < (currentLocation.TimeMillis.Int64-lastLocation.TimeMillis.Int64)
				if int64(config.MinMoveTime) < (currentLocation.TimeMillis.Int64-candidates[0].TimeMillis.Int64) || isFirst { // firstKeyPoint of track does not need to have minMoveTime

					//dbg.WTF(pdTag, "KeyPoint created because of distance moved : %d", dist)
					// check distance to last KeyPoint
					// if len(keyPoints) > 0 {
					//	curKPcand := geo.NewPoint(currentLocation.Latitude.Float64, currrentLocation.Longitude.Float64)
					//	lastKP := geo.NewPoint(keyPoints[len(keyPoints)-1].Latitude.Float64, keyPoints[len(keyPoints)-1].Longitude.Float64)
					// distlastKP := curKPcand.GreatCircleDistance(lastKP) * 1000

					// dbg.I(pdTag, "fkp.. in step %d added KP #%d (dur=%d s) after %d cands %f m away (acc=%f) from lastKP...", idx, len(keyPoints), (currentLocation.TimeMillis.Int64-candidates[0].TimeMillis.Int64)/1000, len(candidates), dist, currentLocation.Accuracy.Float64, currentLocation)

					newKP := interpolateLocationsToKeyPoint(candidates)
					// set the endtime to the time of the point that was distant - this is not 100% accurate, but we can't get it 100% accurate :/
					newKP.EndTime = currentLocation.TimeMillis
					wg.Add(1)
					cKpForAddresses <- newKP
					keyPoints = append(keyPoints, newKP)

					isFirst = false

				} // stay time > minMoveTime
			} //  we have candidates
			candidates = make([]Location, 0)
			candidates = append(candidates, currentLocation)
			lastLocation = currentLocation
		} else { // we did NOT move since last time
			candidates = append(candidates, currentLocation)
			// dbg.D(pdTag, "fkp..appended to candidate...", currentLocation)
		}
	} // we're done iterating

	// check if there are candidates left, which weren't made a keypoint yet
	// the last fix should be (part of) a KeyPoint too
	if len(candidates) > 0 { // create a keypoint from the candidates
		// TODO: make interpolation of average point
		newKP := interpolateLocationsToKeyPoint(candidates)
		wg.Add(1)
		cKpForAddresses <- newKP

		keyPoints = append(keyPoints, newKP)
		// dbg.I(pdTag, "fkp.. at the end added as KP (dur=%d sec) after %d cands...", (newKP.EndTime.Int64-newKP.StartTime.Int64)/1000, len(candidates), newKP)
	} else {
		keyPoints[len(keyPoints)-1].EndTime = (raw)[len(raw)-1].TimeMillis
		// dbg.I(pdTag, "fkp.. at the end updated endTime for last KP in list")
	}

	dbg.I(pdTag, "FindKeyPoints: found %d inaccureate and %d keypoints...", inaccuratePoints, len(keyPoints)) //, keyPoints)

	// make sure we have at least 2 keypoints in this timeframe,
	// TODO: if the last one is no real keypoint(moving) mark him as "notFinal"
	if len(keyPoints) < 2 {
		dbg.W(pdTag, "we got only 1 keypoint...AAAALERT")
	}

	wg.Wait()
	if len(errs) != 0 { // TODO: Save failed stuff into database for later reprocessing???
		dbg.E(pdTag, "Errors occured getting keypoints addresses")
		return keyPoints, errors.New("Errors occured getting keypoint addresses")
	}
	return keyPoints, nil
}

// CreateFilteredTrackPoints links a filtered list of trackpoints to a track using data from trackRecords
func CreateFilteredTrackPoints(trackId int64, config *LocationConfig, dbCon *sql.DB) ([]TrackPoint, error) {

	var inaccuratePoints int
	var startTime sql.NullInt64
	var endTime sql.NullInt64
	var deviceId sql.NullInt64
	var startLat sql.NullFloat64
	var endLat sql.NullFloat64
	var startLng sql.NullFloat64
	var endLng sql.NullFloat64

	trackPoints := make([]TrackPoint, 0)

	if config == nil {
		config = GetDefaultLocationConfig()
	}

	// TODO: CS use crossplattform DB stuff
	err := dbCon.QueryRow("SELECT "+
		"skp.endTime AS start, "+
		"ekp.startTime AS end, "+
		"t.deviceId, "+`
		skp.Latitude,
		ekp.Latitude,
		skp.Longitude,
		ekp.Longitude
		`+
		"FROM tracks AS t "+
		"LEFT JOIN keyPoints AS skp ON skp._keyPointId = t.startKeyPointId "+
		"LEFT JOIN keyPoints AS ekp ON ekp._keyPointId = t.endKeyPointId "+
		"WHERE _trackId=?", trackId).Scan(&startTime, &endTime, &deviceId, &startLat, &endLat, &startLng, &endLng)

	if err != nil {
		dbg.E(pdTag, "CreateFilteredTrackPoints: Failed to get trackData for %d from DB...", trackId, err)
		return nil, err
	}

	// check if trackPoints for track already there
	oldTPs, err := tripMan.GetTrackPointsForTrack(dbCon, trackId)
	oldTPsCount := len(*oldTPs)
	if oldTPsCount > 0 {
		dbg.I(pdTag, "there are already %d TrackPoints for Track %d...abort CreateFilteredTrackPoints", oldTPsCount, trackId)
		errStr := fmt.Sprintf("CreateFilteredTrackPoints:already %d TrackPoints imported for Track %d", oldTPsCount, trackId)
		return nil, errors.New(errStr)
	}

	trackRecs, err := GetTrackRecordsForDevice(startTime.Int64, endTime.Int64, int(deviceId.Int64), dbCon)
	if err != nil {
		dbg.E(pdTag, "CreateFilteredTrackPoints: Failed to get trackrecords...", err)
		return nil, err
	}

	startPoint := TrackPoint{
		TrackId:      sql.NullInt64{Int64: trackId, Valid: true},
		TimeMillis:   (trackRecs)[0].TimeMillis,
		Latitude:     startLat,
		Longitude:    startLng,
		Accuracy:     sql.NullFloat64{Float64: 0, Valid: true},
		Speed:        sql.NullFloat64{Float64: 0, Valid: true},
		MinZoomLevel: sql.NullInt64{Int64: 0, Valid: true},
		MaxZoomLevel: sql.NullInt64{Int64: 18, Valid: true},
	}
	prevPoint := &startPoint
	trackPoints = append(trackPoints, startPoint)
	var distance float64 // its in km
	distance = 0

	for _, currentLocation := range trackRecs {
		if currentLocation.Accuracy.Float64 > float64(config.AccuracyThreshold) {
			inaccuratePoints++
			continue // dont worry about inaccurate points at all
		}

		// check distance
		p := geo.NewPoint(currentLocation.Latitude.Float64, currentLocation.Longitude.Float64)
		lastPoint := geo.NewPoint(trackPoints[len(trackPoints)-1].Latitude.Float64, trackPoints[len(trackPoints)-1].Longitude.Float64)
		curDistance := p.GreatCircleDistance(lastPoint) * 1000
		if curDistance > ((float64)(currentLocation.TimeMillis.Int64-prevPoint.TimeMillis.Int64)*0.250) || (curDistance > ((float64)(currentLocation.TimeMillis.Int64-prevPoint.TimeMillis.Int64)*0.138) && currentLocation.TimeMillis.Int64-prevPoint.TimeMillis.Int64 < 1000*30*60) {
			dbg.I(pdTag, "Skipping because of curDistance %d", curDistance)
			// moving faster than 500 km/h (138 m/s) in a timespan of less than 30 minutes (probably not flying...) or more than 900km/h  (250 m/s) in any timespan
			inaccuratePoints++
			continue // dont worry about inaccurate points at all
		}
		if curDistance < float64(config.MinMoveDist/2) {
			continue // less than half of minMoveDist
		}

		// TODO: figure out the zoom levels http://wiki.openstreetmap.org/wiki/DE:Zoom_levels
		// set low max/min zoom for far apart points
		//   @see LocatoinConfig
		// keep track of how far the last zommed filtered point was
		// set high zoom level number for some points inbetween
		// check distance and number of lower level zoomed points
		minZ := sql.NullInt64{Int64: 3, Valid: true}
		maxZ := sql.NullInt64{Int64: 18, Valid: true}

		currentPoint := TrackPoint{
			TrackId:      sql.NullInt64{Int64: trackId, Valid: true},
			TimeMillis:   currentLocation.TimeMillis,
			Latitude:     currentLocation.Latitude,
			Longitude:    currentLocation.Longitude,
			Accuracy:     currentLocation.Accuracy,
			Speed:        currentLocation.Speed,
			MinZoomLevel: minZ,
			MaxZoomLevel: maxZ,
		}

		trackPoints = append(trackPoints, currentPoint)
		prevPoint = &currentPoint
		distance += curDistance
	}

	endPoint := TrackPoint{
		TrackId:      sql.NullInt64{Int64: trackId, Valid: true},
		TimeMillis:   endTime,
		Latitude:     endLat,
		Longitude:    endLng,
		Accuracy:     sql.NullFloat64{Float64: 0, Valid: true},
		Speed:        sql.NullFloat64{Float64: 0, Valid: true},
		MinZoomLevel: sql.NullInt64{Int64: 0, Valid: true},
		MaxZoomLevel: sql.NullInt64{Int64: 18, Valid: true},
	}
	trackPoints = append(trackPoints, endPoint)

	// dbg.I(pdTag, "CreateFilteredTrackRecords: found %d (orig=%d) trackPoints for Track %d... inserting now...", len(trackPoints), len(*trackRecs), trackId)

	// as seen in dbMan InsertCSVToDb: http://stackoverflow.com/a/25192138/3085985
	valueStrings := make([]string, 0, len(trackPoints))
	valueArgs := make([]interface{}, 0, len(trackPoints)*8)

	for idx, tp := range trackPoints {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs, trackId)
		valueArgs = append(valueArgs, tp.TimeMillis)
		valueArgs = append(valueArgs, tp.Latitude)
		valueArgs = append(valueArgs, tp.Longitude)
		valueArgs = append(valueArgs, tp.Accuracy)
		valueArgs = append(valueArgs, tp.Speed)
		valueArgs = append(valueArgs, tp.MinZoomLevel)
		valueArgs = append(valueArgs, tp.MaxZoomLevel)

		if len(valueArgs) > 990 {
			// TODO: remove duplication
			// dbg.D(pdTag, "idx: %d adding %d of %d trackPoints to DB for track %d", idx, len(valueArgs)/8, len(trackPoints), trackId)
			stmt := fmt.Sprintf("INSERT INTO `trackPoints` (trackId, timeMillis, latitude, longitude, accuracy, speed, minZoomLevel, maxZoomLevel) VALUES %s", strings.Join(valueStrings, ","))
			// TODO: CS use crossplattform DB stuff
			_, err := dbCon.Exec(stmt, valueArgs...)
			if err != nil {
				dbg.E(pdTag, "idx: %d unable to insert %d TrackPoints for Track %d to DB", idx, len(valueArgs), trackId)
				return nil, err
			}
			valueArgs = make([]interface{}, 0)
			valueStrings = make([]string, 0)
		}
	}

	// add the last ones too
	if len(valueArgs) > 0 {
		// dbg.D(pdTag, "adding LAST  %d of %d trackPoints to DB for track %d", len(valueArgs)/8, len(trackPoints), trackId)
		stmt := fmt.Sprintf("INSERT INTO `trackPoints` (trackId, timeMillis, latitude, longitude, accuracy, speed, minZoomLevel, maxZoomLevel) VALUES %s", strings.Join(valueStrings, ","))
		// TODO: CS use crossplattform DB stuff
		_, err := dbCon.Exec(stmt, valueArgs...)
		if err != nil {
			dbg.E(pdTag, "unable to insert LAST %d TrackPoints for Track %d to DB", len(valueArgs), trackId)
			return nil, err
		}
	}

	// and update distance for the track
	_, err = dbCon.Exec("UPDATE `Tracks` SET distance=? WHERE _trackId=?", distance, trackId)
	if err != nil {
		dbg.E(pdTag, "unable to update distance %d for Track %d to DB", distance, trackId)
		return nil, err
	}
	// TODO: after successful inserts, we could remove all this shit from trackRecords

	return trackPoints, nil
}

// interpolateLocationsToKeyPoint takes a bunch of Locations, returns a KeyPoint for them
func interpolateLocationsToKeyPoint(locs []Location) *KeyPoint {

	// Interpolate the locs to get the most center point (weighted by number of points) and display it
	// as KeyPoint

	var avgLat float64
	var avgLng float64
	lowestAcc := 1000000.0

	var newestTs int64
	var oldestTs int64
	// Try to find the most accurate point in the points to interpolate
	latSum := 0.0
	lngSum := 0.0
	pCnt := 0.0
	lastDistantPoint := locs[len(locs)-1]

	avgLat = lastDistantPoint.Latitude.Float64
	avgLng = lastDistantPoint.Longitude.Float64

	newestTs = lastDistantPoint.TimeMillis.Int64
	oldestTs = locs[0].TimeMillis.Int64

	lowestAcc = lastDistantPoint.Accuracy.Float64

	lastDpsToAnalyze := locs
	if len(locs) > 2 { // remove last point from the points we stayed at one point
		// as it probably is the one where we already started moving :)
		lastDpsToAnalyze = locs[:len(locs)-2]
	}

	// dbg.WTF(pdTag, "Starting to analyze %d points", len(lastDpsToAnalyze))
	for _, lp := range lastDpsToAnalyze {
		// dbg.I(pdTag, "Adding point %+v", lp)
		if newestTs < lp.TimeMillis.Int64 {
			newestTs = lp.TimeMillis.Int64
		}
		if oldestTs > lp.TimeMillis.Int64 {
			oldestTs = lp.TimeMillis.Int64
		}
		if lowestAcc > lp.Accuracy.Float64 {
			lowestAcc = lp.Accuracy.Float64
			avgLat = lp.Latitude.Float64
			avgLng = lp.Longitude.Float64

		}
		latSum += lp.Latitude.Float64
		lngSum += lp.Longitude.Float64
		pCnt += 1
	}

	if pCnt > 3 { // with at least 4 points, use the average point. Otherwise most accurate point will be used.
		avgLat = latSum / pCnt
		avgLng = lngSum / pCnt
	}
	var refPoint = geo.NewPoint(avgLat, avgLng)
	var avgDistance float64
	avgDistance = 10000

	for _, lp := range locs {
		d := refPoint.GreatCircleDistance(geo.NewPoint(lp.Latitude.Float64, lp.Longitude.Float64))
		if d < avgDistance {
			avgLat = lp.Latitude.Float64
			avgLng = lp.Longitude.Float64
			avgDistance = d
		}
	}

	newKP := KeyPoint{
		KeyPointId:        sql.NullInt64{Int64: 0, Valid: false},
		Latitude:          sql.NullFloat64{Float64: avgLat, Valid: true},
		Longitude:         sql.NullFloat64{Float64: avgLng, Valid: true},
		StartTime:         sql.NullInt64{Int64: oldestTs, Valid: true},
		EndTime:           sql.NullInt64{Int64: newestTs, Valid: true},
		PreviousTrackId:   sql.NullInt64{Int64: -4, Valid: false},
		NextTrackId:       sql.NullInt64{Int64: -3, Valid: false},
		DeviceId:          sql.NullInt64{Int64: -2, Valid: false},
		PointOfInterestId: sql.NullInt64{Int64: -1, Valid: false},
	}

	return &newKP
}

// ReprocessDataForDeviceInTimeRange deletes all tracks, KeyPoints and trackPoints for device in TimeRange
func ReprocessDataForDeviceInTimeRange(startTime int64, endTime int64, deviceId int,uId int64,activeNotifications *[]*notificationManager.Notification,T *translate.Translater, dbCon *sql.DB) (err error) {
	err = ProcessGPSData(startTime, endTime, deviceId, true,uId,activeNotifications,T, dbCon)
	return err
}
