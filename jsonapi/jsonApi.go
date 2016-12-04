// maps all api-relevant golang-objects to JSON-objects
// list of structures: see /models/geoLocationJSON.go
package jsonapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/datapolish"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/tripMan"
	geo "github.com/OpenDriversLog/goodl-lib/models/geo"
	"github.com/OpenDriversLog/goodl-lib/tools"
	"math"
)

const jaTag = "glib/jsonapi/jsonApi.go"

// GetDeviceTimeRange gets the time-range where data is available for the device with the given deviceId.
func GetDeviceTimeRange(deviceId int, dbCon *sql.DB) ([]byte, error) {
	var marshaled []byte
	var err error
	defer func() { // Error handling, if this getviewdata panics (should not happen)
		if errr := recover(); errr != nil {
			marshaled = []byte("unable to get TimeFrame for device")
			err = errors.New(fmt.Sprintf("%s", errr))
		}
	}()
	minTime, maxTime, err := datapolish.GetDeviceTimeRange(deviceId, dbCon)
	if err != nil {
		marshaled = []byte("unable to convert DeviceTimeRange to JSON")
		dbg.E(jaTag, "unable to get Track id=%d", err)
	}
	marshaled, err = json.Marshal(struct {
		Id  int   `json:"deviceId"`
		Min int64 `json:"start"`
		Max int64 `json:"end"`
	}{deviceId, minTime, maxTime})
	if err != nil {
		return nil, err
	}

	return marshaled, nil
}

// GetTrackIdsInTimeRange returns JSON Array of INTs with ids of tracks between Mintime and MaxTime
func GetTrackIdsInTimeRange(minTime int64, maxTime int64, devices []interface{}, dbCon *sql.DB) ([]byte, error) {
	if maxTime == 0 {
		maxTime = math.MaxInt64
	}
	var marshaled []byte
	dbg.D(jaTag, "Start TrackIds", devices)
	var err error
	defer func() { // Error handling, if this getviewdata panics (should not happen)
		if errr := recover(); errr != nil {
			marshaled = []byte("unable to get TimeFrame for device")
			err = errors.New(fmt.Sprintf("%s", errr))
		}
	}()

	ids, err := tripMan.GetTrackIdsInTimeRange(minTime, maxTime, devices, dbCon)
	if err != nil {
		return nil, err
	}
	dbg.D(jaTag, "Got TrackIds", ids)
	marshaled, err = json.Marshal(struct {
		Ids []int64 `json:"trackIds"`
	}{ids})
	if err != nil {
		return nil, err
	}
	dbg.D(jaTag, "Marshaled TrackIds")

	return marshaled, nil
}

// GetTrackById returns geofeature collection with 3 features for a given trackId: TrackInfo, StartKeyPoint,  EndKeyPoint
func GetTrackById(db *sql.DB, trackId int64) (marshaled []byte, err error) {
	defer func() { // Error handling, if this getviewdata panics (should not happen)
		if errr := recover(); errr != nil {
			marshaled = []byte("unable to complete jsonApi.GetTrackById")
			dbg.E(jaTag, "recovering jsonApi.getTrackById: unable to complete getTrackById w/ id=", errr)
			err = errors.New(fmt.Sprintf("%s", errr))
		}
	}()

	track, err := tripMan.GetTrackById(db, trackId)
	if err != nil {
		marshaled = []byte("unable to convert TrackById to JSON")
		dbg.E(jaTag, "unable to get Track id=%d", trackId, err)
	}

	// TODO: Make timeconfig language dependent!
	timeConfig := tools.GetDefaultTimeConfig()

	geoTrack := geo.NewGeoFeatureCollection()

	geoTrack.Properties = make(map[string]interface{})
	geoTrack.Properties["track"] = trackId
	geoTrack.Properties["device"] = track.DeviceId

	gfSKP := geo.NewGeoFeature()
	gfSKP.Id = "StartKeyPoint"
	addAttributesToKeyPoint(gfSKP, track.StartKeyPointInfo, timeConfig)

	gfEKP := geo.NewGeoFeature()
	gfEKP.Id = "EndKeyPoint"
	addAttributesToKeyPoint(gfEKP, track.EndKeyPointInfo, timeConfig)

	geoTrack.Features = append(geoTrack.Features, gfSKP)
	geoTrack.Features = append(geoTrack.Features, gfEKP)

	marshaled, err = json.Marshal(geoTrack)
	if err != nil {
		return nil, err
		dbg.E(jaTag, "unable to marshal geoTrack id=%d", geoTrack, err)
	}

	return marshaled, nil
}

// GetLastKeyPointForDeviceBefore returns geofeature for the last KeyPoint before the given timeStamp
func GetLastKeyPointForDeviceBefore(deviceId int64, startTime int64, dbCon *sql.DB) (marshaled []byte, err error) {
	defer func() { // Error handling, if this getviewdata panics (should not happen)
		if errr := recover(); errr != nil {
			marshaled = []byte("unable to complete GetLastKeyPointForDeviceBefore")
			dbg.E(jaTag, "recovering GetLastKeyPointForDeviceBefore: unable to complete geoTrack id=", errr)
			err = errors.New(fmt.Sprintf("%s", errr))
		}
	}()

	// TODO: Make timeconfig language dependent!
	timeConfig := tools.GetDefaultTimeConfig()

	kpi, err := datapolish.GetLastKeyPointForDeviceBefore(deviceId, startTime, dbCon)

	gfKP := geo.NewGeoFeature()
	gfKP.Id = "LastKeyPoint"
	addAttributesToKeyPoint(gfKP, kpi, timeConfig)

	marshaled, err = json.Marshal(gfKP)
	if err != nil {
		dbg.E(jaTag, "unable to marshal geoTrack id=%d", gfKP, err)
	}

	return marshaled, nil
}

// addAttributesToKeyPoint adds attributes of KeyPointInfo as properties of NewGeoFeature
func addAttributesToKeyPoint(kp *geo.GeoFeature, kpi *geo.KeyPointInfo, timeConfig *tools.TimeConfig) {
	pnt := geo.NewGeoPointGeometry()
	pnt.Coordinates = geo.NewGeoCoord(kpi.Lng, kpi.Lat)
	kp.Geometry = pnt
	kp.Properties["Postal"] = fmt.Sprintf("%05v", kpi.Postal)
	kp.Properties["City"] = kpi.City
	kp.Properties["Street"] = kpi.Street
	kp.Properties["MinTime"] = kpi.MinTime
	kp.Properties["MaxTime"] = kpi.MaxTime
	kp.Properties["MatchingContactids"] = kpi.MatchingContactids

	kp.Properties["KeyPointId"] = kpi.KeyPointId
	sTime := time.Unix(0, kpi.MinTime*int64(time.Millisecond))
	eTime := time.Unix(0, kpi.MaxTime*int64(time.Millisecond))
	timeStr := sTime.Format(timeConfig.LongTimeFormatString) + "-" +
		eTime.Format(timeConfig.LongTimeFormatString)

	kp.Properties["name"] = timeStr

	kp.Properties["class"] = "marker"
}

// GetTracksByDevice IS NOT IMPLEMENTED YET
func GetTracksByDevice(db *sql.DB, deviceId int64) (marshaled []byte, err error) {
	defer func() { // Error handling, if this getviewdata panics (should not happen)
		if errr := recover(); errr != nil {
			marshaled = []byte("unable to get TracksForDevice")
			err = errors.New(fmt.Sprintf("%s", errr))
		}
	}()

	return nil, nil
}

// GetMinMaxTime gets the minimum & maximum time stamp for all tracks in the database.
func GetMinMaxTime(dbCon *sql.DB) (marshaled []byte, err error) {

	defer func() { // Error handling, if this getviewdata panics (should not happen)
		if errr := recover(); errr != nil {
			marshaled = []byte("unable to get TrackPointsForTrack")
			err = errors.New(fmt.Sprintf("%s", errr))
		}
	}()

	min, max, _ := datapolish.GetMinMaxTimeForAllDevices(dbCon)
	res := make(map[string]int64)
	res["MaxTime"] = max
	res["MinTime"] = min

	marshaled, err = json.Marshal(res)
	if err != nil {
		dbg.E(jaTag, "GetMinMaxTime: Failed to convert minMax for alle devices to JSON ")
		marshaled = []byte("unable to convert minMax for alle devices to JSON ")
		return marshaled, err
	}
	return marshaled, nil
}
