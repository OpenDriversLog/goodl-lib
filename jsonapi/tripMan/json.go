// maps all api-relevant golang-objects to JSON-objects
// list of structures: see /models/geoLocationJSON.go
package tripMan

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"strings"

	"strconv"

	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/models"

	geo "github.com/OpenDriversLog/goodl-lib/models/geo"
	. "github.com/OpenDriversLog/goodl-lib/jsonapi/tripMan/models"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/notificationManager"
	"github.com/OpenDriversLog/goodl-lib/translate"
)

const jsonTAG = "glib/tripMan/json.go"
const NoDataGiven = "Please fill at least one entry."
const NotAllowed = "Operation is not allowed in this context"

// JSONSelectTripIdsInTimeframe returns JSON Array of INTs with ids of Trips between Mintime and MaxTime
func JSONSelectTripIdsInTimeframe(minTime int64, maxTime int64, deviceIds []interface{}, dbCon *sql.DB) (res models.JSONSelectAnswer, err error) {

	if len(deviceIds) == 0 || minTime == 0 || maxTime == 0 || minTime >= maxTime {
		dbg.E(TAG, "faulty select in tripidsintimereange: ", minTime, maxTime, deviceIds, err)
		res = models.GetBadJSONSelectAnswer("faulty selectTripIdsInTimeRange query")
		return
	}

	ids, err := GetTripIdsInTimeRange(minTime, maxTime, deviceIds, dbCon)

	if err != nil {
		if err == sql.ErrNoRows {
			res = models.GetBadJSONSelectAnswer("Not found")
		} else {
			res = models.GetBadJSONSelectAnswer("Internal server error")
		}
		err = nil
		return
	}
	res = models.GetGoodJSONSelectAnswer(ids)
	return
}

// JSONSelectTripsInTimeframe returns JSON Array of Trips  between Mintime and MaxTime
func JSONSelectTripsInTimeframe(minTime int64, maxTime int64, deviceIds []interface{}, includeTracks bool, trackDetails bool,uId int64,activeNotifications *[]*notificationManager.Notification,T *translate.Translater,withHistory bool, dbCon *sql.DB) (res JSONTripManAnswer, err error) {

	res = JSONTripManAnswer{}

	if len(deviceIds) == 0 || minTime == 0 || maxTime == 0 || minTime >= maxTime {
		dbg.E(TAG, "faulty select in tripsintimereange: ", minTime, maxTime, deviceIds, err)
		res = GetBadJsonTripManAnswer("selectTripsInTimeRange query")
		return
	}

	res.Trips, err = GetTripsInTimeRange(minTime, maxTime, deviceIds, false, includeTracks, trackDetails,uId,activeNotifications,T,withHistory, dbCon)

	if err != nil {
		if err == sql.ErrNoRows {
			res = GetBadJsonTripManAnswer("found")
		} else {
			res = GetBadJsonTripManAnswer("Internal server error")
		}
		err = nil
		return
	}
	return
}

// JSONSelectTrip gets the trip with the given ID.
func JSONSelectTrip(tripId int64, includeTracks bool, trackDetails bool,activeNotifications *[]*notificationManager.Notification,T *translate.Translater,withHistory bool, dbCon *sql.DB) (res JSONTripManAnswer, err error) {

	res = JSONTripManAnswer{}

	if tripId == 0 {
		res = GetBadJsonTripManAnswer("faulty tripId")
		return
	}

	trip, err := GetTrip(tripId, false, includeTracks, trackDetails,activeNotifications,T,withHistory, dbCon)

	if err != nil {
		dbg.E(TAG, "Error in GetContact JSONSelectTrip: ", err)
		if err == sql.ErrNoRows {
			res = GetBadJsonTripManAnswer("Not found")
		} else {
			res = GetBadJsonTripManAnswer("Internal server error")
		}
		err = nil
		return
	}
	res.Trips = append(res.Trips, trip)
	return
}

func GetBadJsonTripManAnswer(message string) JSONTripManAnswer {
	return JSONTripManAnswer{
		JSONAnswer: models.GetBadJSONAnswer(message),
	}
}

// JSONGetTrackPointsForTrack returns the TrackPoints of the given track as multiline-GeoFeatures.
// TODO: figure out how to use accuracy, zoomlevel, speeds and maybe time at point
func JSONGetTrackPointsForTrack(dbCon *sql.DB, trackId int64) (marshaled []byte, err error) {
	defer func() { // Error handling, if this getviewdata panics (should not happen)
		if errr := recover(); errr != nil {
			marshaled = []byte("unable to get TrackPointsForTrack")
			err = errors.New(fmt.Sprintf("%s", errr))
		}
	}()

	points, err := GetTrackPointsForTrack(dbCon, trackId)
	if err != nil {
		marshaled = []byte("unable to convert TrackPointsforTrack to JSON")
		dbg.E(TAG, "unable to get datapolish.TrackPointsForTrack id=%d", trackId, err)
	}

	geoFeature := geo.NewGeoFeature()
	geoFeature.Id = "Track"
	geoFeature.Properties["id"] = trackId
	geoFeature.Properties["numberOfPoints"] = len(*points)

	multiline := geo.NewGeoLineStringGeometry()

	for _, val := range *points {
		// dbg.D(jaTag, "running through points id=%d", val)

		multiline.Coordinates = append(multiline.Coordinates, geo.NewGeoCoord(val.Longitude.Float64, val.Latitude.Float64))
	}

	geoFeature.Geometry = multiline

	marshaled, err = json.Marshal(geoFeature)
	if err != nil {
		marshaled = []byte("unable to convert TrackPointsforTrack to JSON")
		return marshaled, err
	}

	return marshaled, nil

}

// JSONUpdateTrips updates the given trips
func JSONUpdateTrips(tripJson string, getUpdatedTrips bool,isAdmin bool,activeNotifications *[]*notificationManager.Notification,T *translate.Translater, dbCon *sql.DB) (res JSONUpdateTripAnswer, err error) {

	ts := make([]*Trip, 0)
	if tripJson == "" {
		res = JSONUpdateTripAnswer{JSONUpdateAnswer: models.GetBadJSONUpdateAnswer(NoDataGiven, -1)}
		return
	}
	err = json.Unmarshal([]byte(tripJson), &ts)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in UpdateTripsJSON : ", tripJson, err)
		res = JSONUpdateTripAnswer{JSONUpdateAnswer: models.GetBadJSONUpdateAnswer("Invalid format", -1)}
		err = nil
		return
	}
	var affectedTrips []int64
	var notifsUpdated bool
	var changes []*CleanTripHistoryEntry
	for _, t := range ts {
		var tempAffectedTrips []int64
		var curChanges CleanTripHistoryEntry
		_, tempAffectedTrips,notifsUpdated,curChanges, err = UpdateTrip(t,isAdmin,activeNotifications,T, dbCon)
		if err != nil {
			dbg.E(TAG, "Error in JSONUpdateTrips UpdateTrip: ", err)
			if err == sql.ErrNoRows {
				res = JSONUpdateTripAnswer{JSONUpdateAnswer: models.GetBadJSONUpdateAnswer("Not found", int64(t.Id))}
			} else { // BEWARE: error strings from the function are sent to frontend
				res = JSONUpdateTripAnswer{JSONUpdateAnswer: models.GetBadJSONUpdateAnswer(err.Error(), int64(t.Id))}
			}
			err = nil
			return
		}
		changes = append(changes,&curChanges)
		affectedTrips = append(affectedTrips, t.Id)
		affectedTrips = append(affectedTrips, tempAffectedTrips...)
	}
	res.UpdatedNotifications = notifsUpdated
	res.Success = true
	res.RowCount = int64(len(affectedTrips))
	res.Changes = changes
	res.Id = int64(-1)
	if getUpdatedTrips {
		res.RemovedTrips, res.UpdatedTrips, err = GetUpdatedTrips(-1, affectedTrips,activeNotifications,T, dbCon)
	}
	return
}

// JSONUpdateTrip updates the given trip
func JSONUpdateTrip(tripJson string, getUpdatedTrips bool, isAdmin bool,activeNotifications *[]*notificationManager.Notification,T *translate.Translater, dbCon *sql.DB) (res JSONUpdateTripAnswer, err error) {
	t := &Trip{}
	if tripJson == "" {
		res = JSONUpdateTripAnswer{JSONUpdateAnswer: models.GetBadJSONUpdateAnswer(NoDataGiven, -1)}
		return
	}
	err = json.Unmarshal([]byte(tripJson), t)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in UpdateTripJSON : ", tripJson, err)
		res = JSONUpdateTripAnswer{JSONUpdateAnswer: models.GetBadJSONUpdateAnswer("Invalid format", -1)}
		err = nil
		return
	}

	var affectedTrips []int64
	var changes CleanTripHistoryEntry
	_, affectedTrips, res.UpdatedNotifications,changes, err = UpdateTrip(t,isAdmin,activeNotifications,T, dbCon)
	if err != nil {
		dbg.E(TAG, "Error in JSONUpdateTrip UpdateTrip: ", err)
		if err == sql.ErrNoRows {
			res = JSONUpdateTripAnswer{JSONUpdateAnswer: models.GetBadJSONUpdateAnswer("Not found", int64(t.Id))}
		} else { // BEWARE: error strings from the function are sent to frontend
			res = JSONUpdateTripAnswer{JSONUpdateAnswer: models.GetBadJSONUpdateAnswer(err.Error(), int64(t.Id))}
		}
		err = nil
		return
	}

	res.Success = true
	res.RowCount = int64(len(affectedTrips) + 1)
	res.Id = int64(t.Id)
	res.Changes = []*CleanTripHistoryEntry{&changes}
	if getUpdatedTrips {
		res.RemovedTrips, res.UpdatedTrips, err = GetUpdatedTrips(res.Id, affectedTrips,activeNotifications,T, dbCon)
	}

	return
}

// GetUpdatedTrips gets arrays of trips that where removed or updated by the updatedTrip, affecting affectedTrips.
func GetUpdatedTrips(udTripId int64, affectedTrips []int64,activeNotifications *[]*notificationManager.Notification,T *translate.Translater, dbCon *sql.DB) (removedTrips []*Trip, updatedTrips []*Trip, err error) {

	q := "?"
	p := make([]interface{}, 0)
	p = append(p, udTripId)
	for _, t := range affectedTrips {
		q += ","
		q += "?"
		p = append(p, t)
	}
	var trips []*Trip
	trips, err = GetTripsByWhere("tripId IN("+q+")", false, true, true, activeNotifications,T, false,dbCon,p...)
	if err != nil {
		dbg.E(TAG, "Error getting trips in GetUpdatedTrips", err)
		return
	}
	for _, t := range trips {
		if len(t.TrackIds) == 0 {
			removedTrips = append(removedTrips, t)
		} else {
			updatedTrips = append(updatedTrips, t)
		}
	}
	return
}

// JSONCreateOrReviveTripByTrackIds Creates a new trip or revives a previous one with the given trackIds.
func JSONCreateOrReviveTripByTrackIds(trackIds string, getUpdatedTrips bool,activeNotifications *[]*notificationManager.Notification,T *translate.Translater, dbCon *sql.DB) (res JSONInsertTripAnswer, err error) {
	tracks := make([]int64, 0)
	for _, v := range strings.Split(trackIds, ",") {
		var i int64
		i, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			err = nil
			res.JSONInsertAnswer = models.GetBadJSONInsertAnswer("Invalid format")
			return
		}
		tracks = append(tracks, i)
	}
	updNots := false
	tripId, affectedTrips, updNots, err := CreateOrReviveTripByTracks(tracks, PRIVATE, "", "", 0, 0, true, true, getUpdatedTrips,activeNotifications,T, dbCon)
	if err != nil {
		dbg.E(TAG, "Error at at JSONCreateOrReviveTripByTrackIDs (CreateOrReviveTripByTracks) : ", err)
		err = nil
		res.JSONInsertAnswer = models.GetBadJSONInsertAnswer("Internal server error")
		return
	}
	res.UpdatedNotifications = updNots

	if getUpdatedTrips {
		res.RemovedTrips, res.UpdatedTrips, err = GetUpdatedTrips(tripId, affectedTrips,activeNotifications,T, dbCon)
	}
	if err != nil {
		dbg.E(TAG, "Error at at JSONCreateOrReviveTripByTrackIDs (GetTrip) : ", err)
		err = nil
		res.JSONInsertAnswer = models.GetBadJSONInsertAnswer("Internal server error")
		return
	}
	res.Success = true
	return
}
