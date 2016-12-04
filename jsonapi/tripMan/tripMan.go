// This package is responsible for managing Trips
package tripMan

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/dbMan/helpers"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/addressManager"
	S "github.com/OpenDriversLog/goodl-lib/models/SQLite"
	. "github.com/OpenDriversLog/goodl-lib/models/geo"
	"github.com/OpenDriversLog/goodl-lib/tools"
	. "github.com/OpenDriversLog/goodl-lib/jsonapi/tripMan/models"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/notificationManager"
	"github.com/OpenDriversLog/goodl-lib/translate"
)

const TAG = "glib/tripMan/tripMan.go"

func init() {
	// if not using goodl-lib from goodl, we have to register SQLITE
	tools.RegisterSqlite("SQLITE")
}

// TripType consts
const BUSINESS = 3
const COMMUTING = 2
const PRIVATE = 1

// GetAllTrips returns all trips
func GetAllTrips(activeNotifications *[]*notificationManager.Notification,T *translate.Translater,dbCon *sql.DB) (trips []*Trip, err error) {
	return GetTripsByWhere("1=1", false, false, false, activeNotifications,T,false, dbCon)
}

// GetTripsInTimeRange gets all trips in the given timerange for the given devices.
func GetTripsInTimeRange(minTime int64, maxTime int64, deviceIds []interface{}, detailedContactData bool, includeTracks bool, trackDetails bool,uId int64,activeNotifications *[]*notificationManager.Notification,T *translate.Translater,withHistory bool, dbCon *sql.DB) (trips []*Trip, err error) {
	addressManager.RetryAddresses(dbCon,uId)
	deviceIdsString := ""
	for i := 0; i < len(deviceIds); i++ {
		if i != 0 {
			deviceIdsString += ","
		}
		deviceIdsString += "?"
	}
	return GetTripsByWhere(fmt.Sprintf("sEndTime<=? AND eStartTime>=? AND sDeviceId IN (%s)", deviceIdsString), detailedContactData, includeTracks, trackDetails, activeNotifications,T,withHistory, dbCon, append([]interface{}{maxTime, minTime}, deviceIds...)...)
}

// GetTripIdsInTimeRange returns int-array of TripIds in the given timeRange for the given deviceIds.
func GetTripIdsInTimeRange(minTime int64, maxTime int64, deviceIds []interface{}, dbCon *sql.DB) ([]int64, error) {
	var id int64
	ids := make([]int64, 0)

	// get all the tracks in TimeRange, their trips are the asked tracks
	trackIds, err := GetTrackIdsInTimeRange(minTime, maxTime, deviceIds, dbCon)

	slstr := fmt.Sprint(trackIds)
	slstr = strings.Join(strings.Split(slstr, " "), ",")
	slstr = strings.TrimLeft(slstr, "[")
	slstr = strings.TrimRight(slstr, "]")

	q := fmt.Sprintf("SELECT t.tripId FROM Tracks_Trips AS t WHERE (trackId IN (%s))", slstr)

	rows, err := dbCon.Query(q)
	if err != nil {
		dbg.E(TAG, "GetTripsIdsInTimeRange: Failed to get trackIds from DB %d until %d", minTime, maxTime, err)
		return ids, err
	}

	for rows.Next() {
		err2 := rows.Scan(&id)
		if err2 != nil {
			dbg.E(TAG, "DbQuery-Error failed to scan trackIds", err2)
			return nil, err2
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// CreateOrReviveTripByTracks builds a trip from array of trackIds, defaults to businessTrip
// returns ID of new/updated trip, list of updated trip ids, bool if notifications were changed and error if occured.
// TODO: figure out default driver, partner, desc
func CreateOrReviveTripByTracks(trackIds []int64, tripType int, title string, description string, driverId int64, contactId int64, tryRevive bool, checkMergeAllowed bool, getAffectedTripIds bool,activeNotifications *[]*notificationManager.Notification, T *translate.Translater, dbCon *sql.DB) (int64, []int64,bool, error) {
	affectedTripIds := make([]int64, 0)
	if len(trackIds) == 0 {
		return -1, affectedTripIds, false, errors.New("No TrackId given")
	}
	if tripType == 0 { // tripType defaults to PRIVATE
		tripType = PRIVATE
	} else if tripType != BUSINESS && tripType != COMMUTING && tripType != PRIVATE { // completely wrong tripType changes to PRIVATE
		tripType = PRIVATE
	}
	if checkMergeAllowed {
		allowed, err := isMergeAllowed(trackIds, dbCon)
		if !allowed || err != nil {
			dbg.E(TAG, "isMergeAllowed failed : ", allowed, err)
			return -1, affectedTripIds, false, err
		}
	}

	var lastTripId int64

	// Check if there WAS a trip with this track already

	var possibleLastTripId *sql.NullInt64
	if tryRevive && len(trackIds) == 1 {
		err := dbCon.QueryRow(`SELECT DISTINCT tripIdOLD FROM Tracks_Trips_History WHERE sqlAction='DELETE'
		AND trackIdOLD = ? AND
		(SELECT COUNT(Tracks_Trips.tripId) FROM Tracks_Trips WHERE Tracks_Trips.tripId = Tracks_Trips_History.tripIdOLD) = 0 ORDER BY ID DESC
	`, trackIds[0]).Scan(&possibleLastTripId)
		if err == nil && possibleLastTripId.Valid {
			lastTripId = possibleLastTripId.Int64
		}
	}
	if lastTripId == 0 {
		resTrip, err := dbCon.Exec("INSERT INTO Trips(type, title,desc,driverId,contactId) VALUES (?, ?, ?, ?, ?)", tripType, title, description, driverId, contactId)
		if err != nil {
			dbg.E(TAG, "Failed to insert trip to Trips..", err)
			return -1, affectedTripIds, false, err
		}
		lastTripId, err = resTrip.LastInsertId()
		if err != nil {
			dbg.E(TAG, "Error getting last insertId for Trip", err)
			return -1, affectedTripIds, false, err
		}
	}
	for _, trackId := range trackIds {
		if getAffectedTripIds {
			res, err := dbCon.Query("SELECT tripId FROM Tracks_Trips WHERE trackId=?", trackId)
			if err != nil {
				dbg.E(TAG, "Error getting affected tripIds", err)
				return -1, affectedTripIds, false, err
			}
			for res.Next() {
				var id sql.NullInt64
				res.Scan(&id)
				if !id.Valid {
					dbg.E(TAG, "Unable to scan TripId", id)
					return -1, affectedTripIds, false, errors.New("Unable to scan TripId")
				}
				affectedTripIds = append(affectedTripIds, id.Int64)
			}
		}
		_, err := dbCon.Exec("DELETE FROM Tracks_Trips WHERE trackId=?;INSERT INTO Tracks_Trips(trackId, tripId) VALUES (?, ?)", trackId, trackId, lastTripId)
		if err != nil {
			dbg.E(TAG, "Failed to insert links into Tracks_Trips..", err)
			return -1, affectedTripIds, false, err
		}
	}
	DueUpd := false
	for _,id := range affectedTripIds {
		trip, err := GetTrip(id,false,false,true,activeNotifications,T,true,dbCon)
		origTOD := trip.TimeOverDue
		if err != nil {
			dbg.E(TAG,"Could not get trip in CreateOrReviveTripByTracks : ", err)
			return -1,affectedTripIds, false,err
		}
		err = calcOverDue(trip,dbCon)
		if err != nil {
			dbg.E(TAG,"Error calculating overdue : ", err)
			return -1,affectedTripIds, false,err
		}
		thisDueUpd := false
		if origTOD != trip.TimeOverDue {
			thisDueUpd = true
			_, err = dbCon.Exec("UPDATE TRIPS SET timeOverDue=? WHERE _tripId=?;", trip.TimeOverDue, trip.Id)
			if err != nil {
				dbg.E(TAG,"Error setting timeOverDue for trip %d : ",trip.Id, err)
				return -1,affectedTripIds, false,err
			}
		}
		if thisDueUpd {
			thisDueUpd,err = notificationManager.CheckForNotificationUpdate(trip,activeNotifications,T,dbCon)
			if err != nil {
				dbg.E(TAG,"Error checking for notification update in CreateOrReviveTrips", err)
			}
			if thisDueUpd {
				DueUpd = true
			}
		}

	}
	trip, err := GetTrip(lastTripId,false,false,true,activeNotifications,T,true,dbCon)
	if err != nil {
		dbg.WTF(TAG,"Error getting newly created trip...", err)
	}
	oldOd := trip.TimeOverDue
	err = calcOverDue(trip,dbCon)
	if oldOd != trip.TimeOverDue {
		_,_,notsUpd,_,err := UpdateTrip(trip,true,activeNotifications,T,dbCon)
		if err != nil {
			dbg.WTF(TAG,"Error updating trip with new TimeOverdue : ", err)
		}
		if notsUpd {
			DueUpd = true
		}
	}
	return lastTripId, affectedTripIds,DueUpd, nil
}

// isMergeAllowed checks if the given trackIds can be merged
func isMergeAllowed(trackIds []int64, dbCon *sql.DB) (allowed bool, err error) {
	var prevTime int64 = 0
	if len(trackIds) > 1 {
		first := true
		for _, trackId := range trackIds {
			var track Track
			track, err = GetTrackById(dbCon, int64(trackId))
			if prevTime > track.StartKeyPointInfo.MinTime {
				dbg.WTF(TAG, "Tracks are in wrong order : %+v", trackIds)
				err = errors.New("Can't merge tracks in wrong order")
				return
			}
			prevTime = track.StartKeyPointInfo.MinTime
			if first {
				first = false
				continue
			}

			if err != nil {
				dbg.E(TAG, "Could not find track %d", trackId, err)
				return
			}
			var twoHours = int64(7200000)
			if track.StartKeyPointInfo.MaxTime-track.StartKeyPointInfo.MinTime > twoHours {
				//stop >2 hours
				dbg.WTF(TAG, "Can't merge tracks with stops >2 hours - based on StartKeyPointInfo : %+v", track.StartKeyPointInfo)
				err = errors.New("Can't merge tracks with stops >2 hours")
				return
			}

		}
	}
	allowed = true
	return
}

// GetTripsByWhere returns all trips with tracks that match the given where- string with the given params.
func GetTripsByWhere(where string, detailedContactData bool, includeTracks bool, trackDetails bool,activeNotifications *[]*notificationManager.Notification,T *translate.Translater,withHistory bool, dbCon *sql.DB, params ...interface{}) (trips []*Trip, err error) {
	// TODO: CS use crossplattform DB stuff

	/*
		Might use this numbers later when we fill contact from this query instead of querying it seperately
		tripColCnt := 8
		keyPointColCnt := 2
		trackColCnt := 5*/
	//colCnt := tripColCnt + keyPointColCnt*2 + trackColCnt*2 + contactColCnt*3

	trips = make([]*Trip, 0)
	contactColCnt := 1
	contactCols := "__ContactId"
	lastTrip := &Trip{Id: -1000}
	ignore := &tools.SQLIgnoreField{}
	historyCols := ""
	historyJoin := ""
	var prevTh *TripHistoryEntry
	if withHistory {
		historyCols = `, Trip_History.id, Trip_History.changeDate, Trip_History.typeOLD, Trip_History.typeNEW, Trip_History.titleOLD,
	Trip_History.titleNEW,Trip_History.descOLD,Trip_History.descNEW,Trip_History.driverIdOLD,
	Trip_History.driverIdNEW,contactIdOLD,Trip_History.contactIdNEW,Trip_History.startContactIdOLD,
	Trip_History.startContactIdNEW,Trip_History.endContactIdOLD,Trip_History.endContactIdNEW,Trip_History.isReturnTripOLD,
	Trip_History.isReturnTripNEW,Trip_History.isReviewedOLD,Trip_History.isReviewedNEW`
		historyJoin = " LEFT JOIN Trip_History ON Trips_FullBlown.tripId = Trip_History.tripId "
	}
	if detailedContactData {
		dbg.W(TAG, "DetailedContactData is currently not 100% performance tuned.")
		// TODO: Test AND USE IT instead of querying contacts seperately later!
		contactColCnt = 16

		// If you update this query, please don't forget to update col counts above!!!
		contactCols = `
	__ContactId,
	__ContactType,
__ContactTitle,
__ContactDescription,
__ContactAdditional,
__ContactAddressId,
__ContactTripTypeId,
__ContactStreet,
__ContactPostal,
__ContactCity,
__ContactAdditional1,
__ContactAdditional2,
__ContactLatitude,
__ContactLongitude,
__ContactHouseNumber,
__ContactAddTitle
`
	}
	q := `SELECT DISTINCT
Trips_FullBlown.tripId,
tripType,
tripTitle,
tripDesc,
tripDriverId,
tripStartContactId,
tripEndContactId,
isReturnTrip,
sKeyPointId,
sEndTime,
sAddressId,
sStreet,
sPostal,
sGeoCoder,
sCity,
sAdditional1,
sAdditional2,
sAddLatitude,
sAddLongitude,
sHouseNumber,
sAddTitle,
eKeyPointId,
eStartTime,
eAddressId,
eStreet,
ePostal,
eGeoCoder,
eCity,
eAdditional1,
eAdditional2,
eAddLatitude,
eAddLongitude,
eHouseNumber,
eAddTitle,
trackId,
tripReviewed,
sDeviceId,
tripTimeOverDue,
` + strings.Replace(contactCols, "__", "proposedS", -1) +
		`,` + strings.Replace(contactCols, "__", "proposedE", -1) +
		`,` + strings.Replace(contactCols, "__", "s", -1) +
		`,` + strings.Replace(contactCols, "__", "e", -1) +
		`,` + strings.Replace(contactCols, "__", "trip", -1) +
		historyCols +
		`
	FROM Trips_FullBlown `+
		historyJoin +
		` WHERE
		` + where + " ORDER BY tripReviewed ASC,sStartTime ASC"
	//dbg.WTF(TAG, "Executing query : %v with params : ", append([]interface{}{interface{}(q)}, params...)...)
	res, err := dbCon.Query(q, params...)
	dbg.I(TAG, "Finished query.")
	if err != nil {
		dbg.E(TAG, "DbQuery-Error failed to get data for Trip with where %v \r\n %v", where, err)
		return trips, err
	}
	//c := make(chan interface{}, 1) // get 1 track in parallel (HAHA - but more don't seem to improve performance.)
	//defer close(c)
	//var wg sync.WaitGroup
	errs := make([]error, 0)
	trackBuffer := make(map[int64]*Track)

	trackPointBuffer := make(map[int64]*[]byte)
	keyPointBuffer := make(map[int64]*KeyPoint_Slim)
	contactBuffer := make(map[int64]*addressManager.Contact)
	var prevTrip *Trip
	for res.Next() {
		trip := Trip{
			EndAddress:   &addressManager.Address{},
			StartAddress: &addressManager.Address{},
			TrackDetails: make([]*Track, 0),
		}
		if withHistory {
			trip.History = make([]*CleanTripHistoryEntry,0)
		}
		th := TripHistoryEntry{}
		ignoreFields := []interface{}{}
		if detailedContactData {
			for i := 0; i < contactColCnt-1; i++ {
				ignoreFields = append(ignoreFields, &ignore)
			}
		}
		scanFields := []interface{}{&trip.Id, &trip.Type, &trip.Title, &trip.Description, &trip.DriverId,
			&trip.StartContactId, &trip.EndContactId, &trip.IsReturnTrip,
			&trip.StartKeyPointId, &trip.StartTime, &trip.StartAddress.Id,
			&trip.StartAddress.Street,
			&trip.StartAddress.Postal,&trip.StartAddress.GeoCoder, &trip.StartAddress.City, &trip.StartAddress.Additional1,
			&trip.StartAddress.Additional2, &trip.StartAddress.Latitude, &trip.StartAddress.Longitude,
			&trip.StartAddress.HouseNumber, &trip.StartAddress.Title,

			&trip.EndKeyPointId, &trip.EndTime, &trip.EndAddress.Id,
			&trip.EndAddress.Street,
			&trip.EndAddress.Postal,&trip.EndAddress.GeoCoder, &trip.EndAddress.City, &trip.EndAddress.Additional1,
			&trip.EndAddress.Additional2, &trip.EndAddress.Latitude, &trip.EndAddress.Longitude,
			&trip.EndAddress.HouseNumber, &trip.EndAddress.Title,
			&trip.TrackIds, &trip.Reviewed, &trip.DeviceId,&trip.TimeOverDue,
		}

		scanFields = append(scanFields, &trip.ProposedStartContactIds)
		if detailedContactData {
			scanFields = append(scanFields, ignoreFields...)
		}
		scanFields = append(scanFields, &trip.ProposedEndContactIds)
		if detailedContactData {
			scanFields = append(scanFields, ignoreFields...)
		}
		scanFields = append(scanFields, &trip.StartContactId)
		if detailedContactData {
			scanFields = append(scanFields, ignoreFields...)
		}
		scanFields = append(scanFields, &trip.EndContactId)
		if detailedContactData {
			scanFields = append(scanFields, ignoreFields...)
		}
		scanFields = append(scanFields, &trip.ContactId)
		if detailedContactData {
			scanFields = append(scanFields, ignoreFields...)
		}
		if withHistory {
			scanFields = append(scanFields,&th.Id, &th.ChangeDate, &th.TypeOld, &th.TypeNew, &th.TitleOld,&th.TitleNew,
				&th.DescOld, &th.DescNew, &th.DriverIdOld, &th.DriverIdNew, &th.ContactIdOld,
				&th.ContactIdNew, &th.StartContactIdOld, &th.StartContactIdNew, &th.EndContactIdOld,
				&th.EndContactIdNew, &th.IsReturnTripOld, &th.IsReturnTripNew, &th.IsReviewedOld,
				&th.IsReviewedNew)
		}
		err = res.Scan(scanFields...)
		if err != nil {
			dbg.E(TAG, "Error scanning row : ", err)
			return
		}
		if withHistory {
			ch := &CleanTripHistoryEntry{
				Id: int64(th.Id),
				ChangeDate:th.ChangeDate,
				Changes:make(map[string]Change),
			}
			if th.ContactIdNew != th.ContactIdOld {
				ch.Changes["ContactId"] = Change{NewVal:th.ContactIdNew,OldVal:th.ContactIdOld}
			}
			if th.DescNew != th.DescOld {
				ch.Changes["Desc"] = Change{NewVal:th.DescNew,OldVal:th.DescOld}
			}
			if th.DriverIdNew != th.DriverIdOld {
				ch.Changes["DriverId"] = Change{NewVal:th.DriverIdNew,OldVal:th.DriverIdOld}
			}
			if th.EndContactIdNew != th.EndContactIdOld {
				ch.Changes["EndContactId"] = Change{NewVal:th.EndContactIdNew,OldVal:th.EndContactIdOld}
			}
			if th.IsReturnTripNew != th.IsReturnTripOld {
				ch.Changes["IsReturnTrip"] = Change{NewVal:th.IsReturnTripNew,OldVal:th.IsReturnTripOld}
			}
			if th.IsReviewedNew != th.IsReviewedOld {
				ch.Changes["IsReviewed"] = Change{NewVal:th.IsReviewedNew,OldVal:th.IsReviewedOld}
			}
			if th.StartContactIdNew != th.StartContactIdOld {
				ch.Changes["StartContactId"] = Change{NewVal:th.StartContactIdNew,OldVal:th.StartContactIdOld}
			}
			if th.TitleNew != th.TitleOld {
				ch.Changes["Title"] = Change{NewVal:th.TitleNew,OldVal:th.TitleOld}
			}
			if th.TypeNew != th.TypeOld {
				ch.Changes["Type"] = Change{NewVal:th.TypeNew,OldVal:th.TypeOld}
			}
			if (prevTh==nil || th.Id!=prevTh.Id) && len(ch.Changes)>0 {
				if prevTrip!=nil && trip.Id == prevTrip.Id{
					prevTrip.History = append(prevTrip.History,ch)
					prevTh = &th
					continue
				} else {
					trip.History = append(trip.History,ch)
					prevTh = &th
				}
			}
		}

		if detailedContactData {
			// TODO : USE THE DATA WE QUERIED ANYWAYS!!!
			//trip.Contact = addressManager.GetContact(int64(trip.ContactId), dbCon)
			if trip.StartContactId != 0 {
				id := int64(trip.StartContactId)
				if contactBuffer[id] != nil {
					trip.StartContact = contactBuffer[id]
				} else {
					trip.StartContact, err = addressManager.GetContact(id, dbCon, true)
					if err != nil {
						dbg.E(TAG, "Error getting StartContact with id %d for trip : ", trip.StartContactId, err)
					}
					contactBuffer[id] = trip.StartContact
				}
			}
			if trip.EndContactId != 0 {
				id := int64(trip.EndContactId)
				if contactBuffer[id] != nil {
					trip.EndContact = contactBuffer[id]
				} else {
					trip.EndContact, err = addressManager.GetContact(id, dbCon, true)
					if err != nil {
						dbg.E(TAG, "Error getting EndContact with id %d for trip : ", trip.EndContactId, err)
					}
					contactBuffer[id] = trip.EndContact
				}
			}
			// TODO : Also get ProposedStartContacts and ProposedEndContacts
		}
		if len(trip.TrackIds) > 0 {
			t := string(trip.TrackIds) // we only have one trackid per query result because group by sucks
			var i int64
			i, err = strconv.ParseInt(t, 10, 64)
			if err != nil {
				dbg.E(TAG, "Error parsing trackId : ", t, err)
				return
			}
			trip.TrackIdInts = []int64{i}
			if trackDetails {
				var track *Track
				if trackBuffer[i] != nil {
					track = trackBuffer[i]
				} else {
					var _track Track
					_track, err = GetTrackById(dbCon, i)
					if err != nil {
						dbg.E(TAG, "Error getting track info : ", err)
						return
					}
					track = &_track
					trackBuffer[i] = track
				}
				trip.TrackDetails = append(trip.TrackDetails, track)

			}
		}
		if includeTracks {
			/*c <- nil
			wg.Add(1)

			go func(trip *Trip) {
				defer func() {
					if err := recover(); err != nil {
						dbg.E(TAG, "Error getting trackdata : ", err)
					}
				}()*/
			ts := make([]string, 0)

			if len(trip.TrackIds) > 0 {
				i := trip.TrackIdInts[0]
				var tr *[]byte
				if trackPointBuffer[i] != nil {
					tr = trackPointBuffer[i]
				} else {
					var _tr []byte
					_tr, err = JSONGetTrackPointsForTrack(dbCon, i)
					if err != nil {
						dbg.E(TAG, "Error getting trackpoints : ", err)
						errs = append(errs, err)
						//wg.Done()
						//<-c
						return
					}
					tr = &_tr
					trackPointBuffer[i] = tr
				}
				ts = append(ts, string(*tr))

				trip.Tracks = ts

				id := int64(trip.StartKeyPointId)
				var skp *KeyPoint_Slim

				if keyPointBuffer[id] != nil {
					skp = keyPointBuffer[id]
				} else {
					var _skp KeyPoint_Slim
					_skp, err = GetKeyPointById(dbCon, id)
					if err != nil {
						dbg.E(TAG, "Error getting StartKeyPoint %+v : ", trip, err)
						errs = append(errs, err)
						//wg.Done()
						//<-c
						return
					}
					skp = &_skp
					keyPointBuffer[id] = skp
				}
				trip.StartKeyPoint = skp
				var ekp *KeyPoint_Slim
				id = int64(trip.EndKeyPointId)
				if keyPointBuffer[id] != nil {
					ekp = keyPointBuffer[id]
				} else {
					var _ekp KeyPoint_Slim
					_ekp, err = GetKeyPointById(dbCon, id)
					if err != nil {
						dbg.E(TAG, "Error getting EndKeyPoint %d: ", trip.EndKeyPointId, err)
						errs = append(errs, err)
						//wg.Done()
						//<-c
						return
					}
					ekp = &_ekp
					keyPointBuffer[id] = ekp
				}
				trip.EndKeyPoint = ekp

			}
			/*<-c
				wg.Done()
			}(&trip)*/
		}

		if(trip.EndKeyPoint != nil) {
			overDueTime := trip.EndKeyPoint.StartTime + 7 * 24 * 60 * 60 * 1000;
			if trip.TimeOverDue != overDueTime {
				trip.TimeOverDue = overDueTime
				dbg.W(TAG,"Updating trip %d with new TimeOverDue of %d",trip.Id, overDueTime)
				_,_,_,_, err = UpdateTrip(&trip,true,activeNotifications,T, dbCon)
				if err != nil {
					dbg.E(TAG,"Error updating trip with new TimeOverDue")
				}
			}
			trip.EditableTime = trip.TimeOverDue - time.Now().Unix() * 1000

		}

		if lastTrip != nil && lastTrip.Id == trip.Id {

			if len(trip.TrackIdInts) > 0 && !containsInt64(lastTrip.TrackIdInts, trip.TrackIdInts[0]) {
				if includeTracks {
					lastTrip.Tracks = append(lastTrip.Tracks, trip.Tracks...)
				}
				if trackDetails {
					lastTrip.TrackDetails = append(lastTrip.TrackDetails, trip.TrackDetails...)
				}
				lastTrip.TrackIds = S.NString(string(lastTrip.TrackIds) + "," + string(trip.TrackIds))

				lastTrip.TrackIdInts = append(lastTrip.TrackIdInts, trip.TrackIdInts[0])
			}
			lastTrip.ProposedEndContactIds = S.NString(tools.AppendStringsByComma(string(lastTrip.ProposedEndContactIds),
				string(trip.ProposedEndContactIds)))
			lastTrip.ProposedStartContactIds = S.NString(tools.AppendStringsByComma(string(lastTrip.ProposedStartContactIds),
				string(trip.ProposedStartContactIds)))

		} else {
			trips = append(trips, &trip)
			lastTrip = &trip
		}

		prevTrip = &trip
	}

	//dbg.I(TAG, "Waiting for Waitgroup")
	//wg.Wait()

	dbg.I(TAG, "Finished GetTripsByWhere")
	if len(errs) != 0 {
		dbg.E(TAG, "Errors occured getting trips tracks")
		return nil, errors.New("Errors occured getting trips tracks")
	}
	return

}

// GetTrip returns the trip with the given ID.
func GetTrip(tripId int64, detailedContactData bool, includeTracks bool, trackDetails bool,activeNotifications *[]*notificationManager.Notification,T *translate.Translater,withHistory bool, dbCon *sql.DB) (trip *Trip, err error) {
	var trips []*Trip
	trips, err = GetTripsByWhere("Trips_FullBlown.tripId=?", detailedContactData, includeTracks, trackDetails,activeNotifications,T,withHistory, dbCon, tripId)
	if err != nil || len(trips) == 0 {
		return
	}
	trip = trips[0]
	return
}

// GetTrackIdsForTripId []int64 of trackIds for a tripId
func GetTrackIdsForTripId(tripId int64, dbCon *sql.DB) (tIds []int64, err error) {
	rows, err := dbCon.Query(`SELECT t.trackId FROM Tracks_Trips AS t WHERE Trips_FullBlown.tripId=?`, tripId)
	if err != nil {
		dbg.E(TAG, "GetTrackIdsForTrip: Failed to get trackIds from DB for Trip %d...", tripId, err)
		return nil, err
	}

	var id int64
	tIds = make([]int64, 0)
	for rows.Next() {
		err2 := rows.Scan(&id)
		if err2 != nil {
			dbg.E(TAG, "DbQuery-Error failed to scan trackIds", err2)
			return nil, err2
		}
		tIds = append(tIds, id)
	}
	return
}

// UpdateTrip updates trip, returns new tripData. Also automatically removes tracks from previous trips if used in this updated one.
func UpdateTrip(trip *Trip, isAdmin bool,activeNotifications *[]*notificationManager.Notification, T *translate.Translater, dbCon *sql.DB) (updatedTrip *Trip, affectedTripIds []int64,notificationsChanged bool,changes CleanTripHistoryEntry, err error) {

	changes = CleanTripHistoryEntry{Id:trip.Id, ChangeDate:S.NString(time.Now().Unix()*1000),Changes:make(map[string]Change)}
	recalcOverdue := false
	oldTrip, err := GetTrip(trip.Id, false, false, false,activeNotifications,T,false, dbCon)
	if err != nil {
		return trip, affectedTripIds,false,changes, errors.New("Trip to update could not be found")
	}

	if !isAdmin && int64(oldTrip.EndTime)+1000*60*60*24*7 < time.Now().Unix()*1000 {
		return trip, affectedTripIds,false, changes,errors.New("Trip is too old to review")
	}


	update := helpers.NewUpdateHelper(dbCon)

	if oldTrip.Title != trip.Title {
		update.AppendNString("title", &trip.Title)
		changes.Changes["title"] = Change{NewVal:trip.Title,OldVal:oldTrip.Title}
	}
	if oldTrip.Description != trip.Description {
		update.AppendNString("desc", &trip.Description)
		changes.Changes["description"] = Change{NewVal:trip.Description,OldVal:oldTrip.Description}

	}

	if trip.ContactId == 0 && trip.Contact != nil {
		trip.ContactId = S.NInt64(trip.Contact.Id)
		changes.Changes["contactId"] = Change{NewVal:trip.Contact.Id,OldVal:oldTrip.Contact.Id}

	}
	if trip.StartContactId == 0 && trip.StartContact != nil {
		trip.StartContactId = S.NInt64(trip.StartContact.Id)
		changes.Changes["startContactId"] = Change{NewVal:trip.StartContact.Id,OldVal:oldTrip.StartContact.Id}

	}
	if trip.EndContactId == 0 && trip.EndContact != nil {
		trip.EndContactId = S.NInt64(trip.EndContact.Id)
		changes.Changes["endContactId"] = Change{NewVal:trip.EndContact.Id,OldVal:oldTrip.EndContact.Id}

	}
	if oldTrip.ContactId != trip.ContactId {
		update.AppendNInt64("contactId", &trip.ContactId)
		changes.Changes["contactId"] = Change{NewVal:trip.ContactId,OldVal:oldTrip.ContactId}

	}
	if oldTrip.EndContactId != trip.EndContactId {
		update.AppendNInt64("endContactId", &trip.EndContactId)
		changes.Changes["endContactId"] = Change{NewVal:trip.EndContactId,OldVal:oldTrip.EndContactId}

	}
	if oldTrip.StartContactId != trip.StartContactId {
		update.AppendNInt64("startContactId", &trip.StartContactId)
		changes.Changes["startContactId"] = Change{NewVal:trip.StartContactId,OldVal:oldTrip.StartContactId}

	}
	if trip.Driver != nil && trip.DriverId == 0 {
		trip.DriverId = S.NInt64(trip.Driver.Id)
		changes.Changes["driverId"] = Change{NewVal:trip.Driver.Id,OldVal:oldTrip.Driver.Id}

	}
	if oldTrip.DriverId != trip.DriverId {
		update.AppendNInt64("driverId", &trip.DriverId)
		changes.Changes["driverId"] = Change{NewVal:trip.DriverId,OldVal:oldTrip.DriverId}

	}



	// at least update one field to don't get errors ;)
	update.AppendNInt64("isReturnTrip", &trip.IsReturnTrip)

	// make sure private cannot become business, etc
	if oldTrip.Type != trip.Type {
		if !dbg.Develop { // don't really care while developing
			if trip.Reviewed == 1 { // but only if it was already reviewed
				if oldTrip.Type == PRIVATE {
					// TODO : make passing through this error more user friendly (currently "Internal server error")
					return oldTrip, affectedTripIds,false, changes, errors.New("Trip Type change from private is not permitted")
					// TODO: should we still update all other fields?
				}
			}
		}
		tt := int64(trip.Type)
		update.AppendInt64("type", &tt)
		changes.Changes["type"] = Change{NewVal:trip.Type,OldVal:oldTrip.Type}

	}

	if oldTrip.Reviewed != trip.Reviewed {
		update.AppendInt64("reviewed", &trip.Reviewed)
		recalcOverdue = true
		changes.Changes["reviewed"] = Change{NewVal:trip.Reviewed,OldVal:oldTrip.Reviewed}

	}

	if trip.TrackIds != oldTrip.TrackIds {
		recalcOverdue = true
		oldTids := strings.Split(string(oldTrip.TrackIds), ",")
		newTids := strings.Split(string(trip.TrackIds), ",")
		if len(newTids) > 1 {

			tids := make([]int64, 0)
			for _, v := range newTids {
				k, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					dbg.E(TAG, "Error parsing trackIds:", err)
					return trip, affectedTripIds,false,changes, err
				}
				tids = append(tids, k)
			}
			allowed, err := isMergeAllowed(tids, dbCon)
			if !allowed || err != nil {
				dbg.E(TAG, "isMergeAllowed failed : ", allowed, err)
				return nil, affectedTripIds, false,changes,err
			}
		}
		cmd := ""
		pms := make([]interface{}, 0)
		doAnything := false
		for _, v := range newTids {
			if !containsString(oldTids, v) {
				doAnything = true
				res, err := dbCon.Query("SELECT tripId FROM Tracks_Trips WHERE trackId=?", v)
				for res.Next() {
					var id sql.NullInt64
					res.Scan(&id)
					if !id.Valid {
						dbg.E(TAG, "Unable to scan TripId", id)
						return nil, affectedTripIds, false,changes, errors.New("Unable to scan TripId")
					}
					affectedTripIds = append(affectedTripIds, id.Int64)
				}
				if err != nil {
					dbg.E(TAG, "Error getting affected tripIds:", err)
					return nil, affectedTripIds,false,changes, err
				}

				cmd += "DELETE FROM Tracks_Trips WHERE trackId=?;INSERT INTO Tracks_Trips (trackId,tripId) VALUES (?,?);"
				pms = append(pms, v, v, trip.Id)
			}
		}
		for _, v := range oldTids {
			if !containsString(newTids, v) {
				dbg.W(TAG, "Deleting tracks from trips is not supported if not done by adding the trackId to another trip.")
				//cmd += "DELETE FROM Tracks_Trips WHERE trackId=? AND tripId=?;"
				//pms = append(pms, v, trip.Id)
			}
		}
		if doAnything {
			_, err = dbCon.Exec(cmd, pms...)
			if err != nil {
				dbg.E(TAG, "UpdateTrip: error dbCon.Exec", err)
				return trip, affectedTripIds,false, changes,errors.New("Trip update not successful. Database error!")
			}
		}
		changes.Changes["trackIds"] = Change{NewVal:trip.TrackIds,OldVal:oldTrip.TrackIds}

	}
	if oldTrip.TimeOverDue != trip.TimeOverDue && !isAdmin {
		trip.TimeOverDue = oldTrip.TimeOverDue // non-admins can't manipulate timeOverDue
	}
	if recalcOverdue {
		err = calcOverDue(trip,dbCon)
		if err != nil {
			dbg.E(TAG,"Error calculating overdue : ", err)
			return
		}
	}
	var DueUpd bool
	if oldTrip.TimeOverDue != trip.TimeOverDue {
		update.AppendInt64("timeOverDue", &trip.TimeOverDue)
		DueUpd = true
	}


	res, err := update.ExecUpdate("Trips", "_tripId=?", trip.Id)
	if err != nil {
		dbg.E(TAG, "UpdateTrip: error execUpdate", err)
		return trip, affectedTripIds, false, changes, errors.New("Trip update failed internally.")
	} else if crows, err := res.RowsAffected(); err != nil || crows == 0 {
		return trip, affectedTripIds,false, changes, errors.New("Trip update affected nothing.")
	}

	//TODO: I really don't like the idea of querying it again if we trust our algorithm... But as long as it is no performance Issue, OK...
	//return trip, nil
	t, err := GetTrip(trip.Id, false, false, true,activeNotifications,T,true, dbCon)
	notificationsChanged = false
	if DueUpd {
		notificationsChanged,err = notificationManager.CheckForNotificationUpdate(trip,activeNotifications,T,dbCon)
		if err != nil {
			dbg.E(TAG,"Error checking for notification update in CreateOrReviveTrips", err)
		}
	}
	return t, affectedTripIds,DueUpd,changes, err
}


// calcOverDue calcs the time when the given trip is overdue and sets it for the given trip. Also gets the trips EndKeyPoint if not defined.
func calcOverDue(trip *Trip,dbCon *sql.DB) (err error) {
	if trip.EndKeyPoint == nil || int64(trip.EndKeyPoint.PreviousTrackId) != trip.TrackIdInts[len(trip.TrackIdInts)-1] {
		var lastTime int64 = 0
		if len(trip.TrackDetails) == 0 {
			dbg.I(TAG,"Getting TrackDetails as they are not given for calcOverDue")

			for _,id := range trip.TrackIdInts {
				var track Track
				track, err = GetTrackById(dbCon, id)
				if err != nil {
					dbg.E(TAG, "Error getting track info : ", err)
					return
				}
				trip.TrackDetails = append(trip.TrackDetails, &track)
			}
		}
		var EndKP *KeyPoint_Slim

		for _, kp := range trip.TrackDetails {
			if lastTime < kp.EndTime {
				EndKP =  GetSlimKpInfo(kp.EndKeyPointInfo,S.NInt64(0),S.NInt64(0))
			}
		}
		trip.EndKeyPoint = EndKP


	}
	if trip.EndKeyPoint != nil {
		overDueTime := trip.EndKeyPoint.StartTime + 7 * 24 * 60 * 60 * 1000;
		if trip.TimeOverDue != overDueTime {
			trip.TimeOverDue = overDueTime
		}
	}
	dbg.I(TAG,"Calculated overDue of %d ", trip.TimeOverDue)
	return
}


// GetSlimKpInfo converts a KeyPointInfo to a KeyPointInfo_Slim with less overhead.
func GetSlimKpInfo(kp *KeyPointInfo, previousTrackId S.NInt64, nextTrackId S.NInt64 ) (*KeyPoint_Slim){
	if kp != nil {
		return &KeyPoint_Slim{
			KeyPointId: kp.KeyPointId,
			Latitude: kp.Lat,
			Longitude: kp.Lng,
			StartTime: kp.MinTime,
			EndTime: kp.MaxTime,
			PreviousTrackId: previousTrackId,
			NextTrackId: nextTrackId,
		}
	}
	return nil
}

// containsString checks if a string slice contains the given string http://stackoverflow.com/questions/10485743/contains-method-for-a-slice
func containsString(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// containsInt64 checks if a int64 slice contains the given in64 http://stackoverflow.com/questions/10485743/contains-method-for-a-slice
func containsInt64(s []int64, e int64) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// GetTrackIdsInTimeRange returns []int64 with ids of tracks between Mintime and MaxTime, if maxTime == 0 end of Time is assumed
func GetTrackIdsInTimeRange(minTime int64, maxTime int64, deviceIds []interface{}, dbCon *sql.DB) ([]int64, error) {
	var id int64
	ids := make([]int64, 0)

	if maxTime == 0 {
		// http://golang.org/pkg/math/#pkg-constants
		// its the same as INTEGER in sqlite3 http://stackoverflow.com/questions/4448284/sqlite3-integer-max-value
		maxTime = math.MaxInt64
	}

	deviceIdsString := ""
	for i := 0; i < len(deviceIds); i++ {
		if i != 0 {
			deviceIdsString += ","
		}
		deviceIdsString += "?"
	}
	rows, err := dbCon.Query(fmt.Sprintf(`SELECT
		t._trackId
		FROM tracks AS t
		INNER JOIN keyPoints AS skp ON skp._keyPointId = t.startKeyPointId
		INNER JOIN keyPoints AS ekp ON ekp._keyPointId = t.endKeyPointId
		WHERE (skp.endTime<=? AND ekp.startTime>=? AND t.deviceId IN(%s))`, deviceIdsString), append([]interface{}{maxTime, minTime}, deviceIds...)...)
	if err != nil {
		dbg.E(TAG, "GetTrackIdsInTimeRange: Failed to get trackIds from DB (%d until...", minTime, maxTime, err)
		return ids, err
	}

	for rows.Next() {
		err2 := rows.Scan(&id)
		if err2 != nil {
			dbg.E(TAG, "DbQuery-Error failed to scan trackIds", err2)
			return nil, err2
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// GetKeyPointById returns the KeyPoint_Slim with the given ID
func GetKeyPointById(db *sql.DB, kpId int64) (kp KeyPoint_Slim, err error) {
	// TODO: CS use crossplattform DB stuff
	err = db.QueryRow(`SELECT
		_keyPointId,
		startTime,
		endTime,
		latitude,
		longitude,
		previousTrackId,
		nextTrackId
		FROM KeyPoints
		WHERE _keyPointId=?`, kpId).Scan(
		&kp.KeyPointId,
		&kp.StartTime,
		&kp.EndTime,
		&kp.Latitude,
		&kp.Longitude,
		&kp.PreviousTrackId,
		&kp.NextTrackId)

	if err != nil {
		dbg.E(TAG, "getKeyPointById: Failed to get KeyPoint for id %d from DB...", kpId, err)

		return KeyPoint_Slim{}, err
	}

	return kp, nil
}

// GetTrackById returns the track with the given ID.
func GetTrackById(db *sql.DB, trackId int64) (track Track, err error) {
	dbg.I(TAG, "Start GetTrackById for %d", trackId)
	track.TrackId = trackId
	track.StartKeyPointInfo = &KeyPointInfo{}
	track.EndKeyPointInfo = &KeyPointInfo{}

	// TODO: CS use crossplattform DB stuff
	err = db.QueryRow(`SELECT
		skp._keyPointId AS sId,
		skp.endTime AS startTime,
		skp.latitude AS startLat,
		skp.longitude AS startLng,
		skp.startTime AS startStartTime,
		skp.endTime AS startEndTime,
		sAddr.postal AS sPostal,
		sAddr.geoCoder AS sGeoCoder,
		sAddr.city AS sCity,
		sAddr.street AS sStreet,
		sAddr.HouseNumber AS sHouseNumber,
		(SELECT GROUP_CONCAT(_contactId) FROM NoKeyPoint_GeoFenceRegion_Contact NKGC WHERE NKGC.keyPointId=skp._keyPointId) AS sContacts,
		ekp._keyPointId AS eId,
		ekp.startTime AS endTime,
		ekp.latitude AS endLat,
		ekp.longitude AS endLng,
		ekp.startTime AS endStartTime,
		ekp.endTime AS endEndTime,
		eAddr.postal AS ePostal,
		eAddr.geoCoder AS eGeoCoder,
		eAddr.city AS eCity,
		eAddr.street AS eStreet,
		eAddr.HouseNumber AS eHouseNumber,
		(SELECT GROUP_CONCAT(_contactId) FROM NoKeyPoint_GeoFenceRegion_Contact NKGC WHERE NKGC.keyPointId=ekp._keyPointId) AS eContacts,
		t.startKeyPointId,
		t.endKeyPointId,
		t.distance,
		t.deviceId
		FROM tracks AS t
		LEFT JOIN keyPoints AS skp ON skp._keyPointId = t.startKeyPointId
		LEFT JOIN keyPoints AS ekp ON ekp._keyPointId = t.endKeyPointId
		LEFT JOIN Addresses as sAddr ON sAddr._addressId = skp.addressId
		LEFT JOIN Addresses as eAddr ON eAddr._addressId = ekp.addressId
		WHERE _trackId=?`, trackId).Scan(
		&track.StartKeyPointInfo.KeyPointId,
		&track.StartTime,
		&track.StartKeyPointInfo.Lat,
		&track.StartKeyPointInfo.Lng,
		&track.StartKeyPointInfo.MinTime,
		&track.StartKeyPointInfo.MaxTime,
		&track.StartKeyPointInfo.Postal,
		&track.StartKeyPointInfo.GeoCoder,
		&track.StartKeyPointInfo.City,
		&track.StartKeyPointInfo.Street,
		&track.StartKeyPointInfo.HouseNumber,
		&track.StartKeyPointInfo.MatchingContactids,
		&track.EndKeyPointInfo.KeyPointId,
		&track.EndTime,
		&track.EndKeyPointInfo.Lat,
		&track.EndKeyPointInfo.Lng,
		&track.EndKeyPointInfo.MinTime,
		&track.EndKeyPointInfo.MaxTime,
		&track.EndKeyPointInfo.Postal,
		&track.EndKeyPointInfo.GeoCoder,
		&track.EndKeyPointInfo.City,
		&track.EndKeyPointInfo.Street,
		&track.EndKeyPointInfo.HouseNumber,
		&track.EndKeyPointInfo.MatchingContactids,
		&track.StartKeyPointId,
		&track.EndKeyPointId,
		&track.Distance, &track.DeviceId)

	if err != nil {
		dbg.E(TAG, "getTrackById: Failed to get trackData for %d from DB...", trackId, err)

		return Track{}, err
	}
	dbg.I(TAG, "End GetTrackById for %d", trackId)
	return track, nil
}

// GetTrackPointsForTrack returns trackPoints for the track with the given ID.
func GetTrackPointsForTrack(db *sql.DB, trackId int64) (*[]S.TrackPoint, error) {
	var tp S.TrackPoint
	tps := make([]S.TrackPoint, 0)

	// TODO: CS use crossplattform DB stuff
	rows, err := db.Query("SELECT _trackPointId, trackId,timeMillis,latitude,longitude,accuracy,speed,minZoomLevel, maxZoomLevel FROM `trackPoints` WHERE (trackId=?) ORDER BY timeMillis ASC", trackId)
	if err != nil {
		dbg.E(TAG, "failed to get rows from trackPoints", err)
		return nil, err
	}

	// for rowFound := nextRow(); true; rowFound = nextRow() { // iterate through all rows now,
	tp = S.TrackPoint{}
	for rows.Next() {
		err = rows.Scan(&tp.TrackPointId, &tp.TrackId, &tp.TimeMillis, &tp.Latitude,
			&tp.Longitude, &tp.Accuracy, &tp.Speed, &tp.MinZoomLevel, &tp.MaxZoomLevel)
		if err != nil {
			dbg.E(TAG, "failed to get rows from trackPoints", err)
			return nil, err
		}
		tps = append(tps, tp)
	}
	// dbg.V(TAG, "getTrackPointsForTrack: got %d trackpoints for track %d... ", len(tps), trackId)

	if err = rows.Err(); err != nil {
		dbg.E(TAG, "getTrackPointsForTrack %d rows-iteration-Error", trackId, err)
		return nil, err
	}
	return &tps, nil
}
