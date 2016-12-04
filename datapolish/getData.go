// Some helper methods for getting device- & trackrecord-data and the default location config

package datapolish

import (
	"database/sql"

	"math"

	"github.com/Compufreak345/dbg"
	geo "github.com/kellydunn/golang-geo"
	. "github.com/OpenDriversLog/goodl-lib/models/SQLite"
	. "github.com/OpenDriversLog/goodl-lib/models/geo"
)

const gdTag = "glib/t/getData.go"

var LocConfig *LocationConfig

//var geoCoder geo.OpenCageGeocoder
var geoCoder geo.MapQuestGeocoder

// GetDefaultLocationConfig returns the default Location-config
func GetDefaultLocationConfig() *LocationConfig {
	return &LocationConfig{
		MinMoveDist:       70,
		MinMoveTime:       3 * 60 * 1000, // ms
		AccuracyThreshold: 200,
		// TODO: add map for show at zoomstage, @see geo-enhanced
		// minDistAtZoom := make map specific size
	}
}

// GetDeviceStrings returns map[int]string of devices
func GetDeviceStrings(db *sql.DB) (devices map[int]string, err error) {
	devices = make(map[int]string)

	// TODO: CS use crossplattform DB stuff
	rows, err := db.Query("SELECT _deviceId,desc FROM devices")
	if err != nil {
		if err == sql.ErrNoRows { // no devices found
			return devices, nil
		}
		dbg.E(gdTag, "DbQuery-Error failed to get devices", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var key int
		var desc string
		err2 := rows.Scan(&key, &desc)
		if err2 != nil {
			dbg.E(gdTag, "DbQuery-Error failed to scan devices", err2)
			return nil, err2
		}
		devices[key] = desc
	}
	return devices, nil
}

// GetMinMaxTimeForAllDevices returns global minTime/maxTime for all generated keypoints over all devices
func GetMinMaxTimeForAllDevices(dbCon *sql.DB) (minTime int64, maxTime int64, err error) {
	// TODO: CS use crossplattform DB stuff
	err = dbCon.QueryRow("SELECT MIN(startTime),MAX(endTime) FROM KeyPoints").Scan(&minTime, &maxTime)
	if err != nil {
		dbg.I(gdTag, "DbQuery-Error failed to get times for device (probably no devices)", err)
		return 0, 0, nil
		// } else {
		//	dbg.I(gdTag, "got times for device", minTime, maxTime, deviceId)
	}
	return
}

// GetDeviceTimeRange returns min & max timeMillis for deviceId based on keyPoints table
func GetDeviceTimeRange(deviceId int, dbCon *sql.DB) (minTime int64, maxTime int64, err error) {
	// TODO: CS use crossplattform DB stuff
	dbCon.QueryRow("SELECT MIN(startTime),MAX(endTime) FROM KeyPoints WHERE deviceId=?", deviceId).Scan(&minTime, &maxTime)
	if err != nil {
		dbg.E(gdTag, "DbQuery-Error failed to get times for device", err)
		return 0, 0, err
		// } else {
		//	dbg.I(gdTag, "got times for device", minTime, maxTime, deviceId)
	}
	return
}

// GetTrackRecordsForDevice gets trackRecords for a specific deviceId (key)
func GetTrackRecordsForDevice(startTime int64, endTime int64, deviceId int, dbCon *sql.DB) ([]Location, error) {

	var loc Location
	tr := make([]Location, 0)

	// TODO: CS use crossplattform DB stuff
	rows2, err := dbCon.Query("SELECT _id,timeMillis,latitude,longitude,altitude,accuracy,provider,source,accuracyrating,speed FROM trackRecords WHERE ( timeMillis >= ? AND timeMillis <= ? AND deviceId=?) ORDER BY timeMillis ASC", startTime, endTime, deviceId)
	if err != nil {
		dbg.E(gdTag, "failed to get rows from trackrecords", err)
		return nil, err
	}
	// dbg.D(gdTag, "start running through rows for device ", deviceId, rows2)

	// TODO: CS use crossplattform DB stuff
	for rows2.Next() {
		loc = Location{}
		err = rows2.Scan(&loc.Id, &loc.TimeMillis, &loc.Latitude,
			&loc.Longitude, &loc.Altitude, &loc.Accuracy,
			&loc.Provider, &loc.Source, &loc.AccuracyRating,
			&loc.Speed)
		tr = append(tr, loc)
	}
	// dbg.I(gdTag, "got the trackrecords... resultsize for device", deviceId, len(tr))

	if err = rows2.Err(); err != nil {
		dbg.E(gdTag, "rows-iteration-Error in getTrackRecords", err)
		return nil, err
	}
	return tr, nil
}

// GetTrackRecords gets a map of trackRecords with deviceId as Key
func GetTrackRecords(startTime int64, endTime int64, db *sql.DB) (map[int][]Location, error) {
	tr := make(map[int][]Location, 0)

	devices, err := GetDeviceStrings(db)
	if err != nil {
		dbg.E(gdTag, "GetTrackRecords couldn't get devices", err)
		return nil, err
	}
	if devices == nil || len(devices) == 0 {
		dbg.E(gdTag, "GetTrackRecords: device list is empty", devices)
		return nil, err
	}

	for key, _ := range devices {
		tr[key], err = GetTrackRecordsForDevice(startTime, endTime, key, db)
	}
	return tr, nil
}

// GetLastKeyPointForDeviceBefore returns last known keypoint for a device, optional: before a specific time. uses math.maxInt64 if startTime == 0
func GetLastKeyPointForDeviceBefore(deviceId int64, startTime int64, dbCon *sql.DB) (kpi *KeyPointInfo, err error) {
	kpi = &KeyPointInfo{}

	if startTime == 0 {
		startTime = math.MaxInt64
	}

	err = dbCon.QueryRow(`SELECT
	kp._keyPointId,
		kp.latitude,
		kp.longitude,
		kp.startTime,
		kp.endTime,
		addr.postal,
		addr.city,
		addr.Street,
		(SELECT GROUP_CONCAT(_contactId) FROM NoKeyPoint_GeoFenceRegion_Contact NKGC WHERE NKGC.keyPointId=kp._keyPointId GROUP BY 1=1) AS GG
		FROM KeyPoints AS kp
		INNER JOIN Addresses AS addr ON addr._addressId = kp.addressId

		WHERE (deviceId=? AND startTime<?) ORDER BY kp.startTime DESC LIMIT 1`, deviceId, startTime).Scan(
		&kpi.KeyPointId,
		&kpi.Lat,
		&kpi.Lng,
		&kpi.MinTime,
		&kpi.MaxTime,
		&kpi.Postal,
		&kpi.City,
		&kpi.Street,
		&kpi.MatchingContactids)

	if err != nil {
		dbg.E(gdTag, "GetLastKeyPointForDeviceBefore: Failed to get lastKP for device %d from DB...", deviceId, startTime, err)
		return kpi, err
	}

	return kpi, nil
}
