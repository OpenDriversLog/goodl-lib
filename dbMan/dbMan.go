// manages DBconnections, validation & migrations for trackrecords
package dbMan

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"encoding/csv"
	"errors"

	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/datapolish"
	. "github.com/OpenDriversLog/goodl-lib/models/SQLite"
)

var existingDbs map[int64]bool

const dTag = "goodl-lib/dbMan.go"

// const dbFilename = "trackrecords.db"

// GetLocationDb creates location DB if not existent and returns a connection to the db
func GetLocationDb(basePath string,usrId int64) (db *sql.DB, err error) {

	savePath := basePath // + dbFilename
	if existingDbs == nil {
		existingDbs = make(map[int64]bool)
	}
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		CreateNewLocationDb(savePath)
		db, err = openDbCon(savePath)
	} else { //  there is a locationdb
		db, err = openDbCon(savePath)

		if err != nil {
			dbg.E(dTag, "Failed to get connection to available database() : ", err)
			return nil, err
		}

		err = CheckIfUpgradeNeeded(db,usrId, basePath)
		if err != nil {
			dbg.E(dTag, "Failed to get check if upgrade required : ", err)
			return nil, err
		}
	}

	if err != nil {
		dbg.E(dTag, "Creating location db failed: ", err)

		if _, err := os.Stat(savePath); !os.IsNotExist(err) {
			dbg.W(dTag, "Hmm but a db-file already exists - rename it to ...destroyed ")
			var now = time.Now()
			os.Rename(savePath, savePath+"_"+string(now.Unix())+"_"+string(time.Now().Nanosecond())+"_destroyed")
		}

		return
	}

	return
}

// GetLocationDbNumbers returns numbers of rows for all those tables
func GetLocationDbNumbers(dbPath string) (latestMigration string, devices int, trackrecords int64, tracks int64, keypoints int64, err error) {
	dbCon, err := openDbCon(dbPath)
	defer dbCon.Close()
	devicemap, err := datapolish.GetDeviceStrings(dbCon)
	devices = len(devicemap)
	t1 := time.Now().UnixNano()
	r, err := dbCon.Query(`SELECT Count(1) FROM TrackRecords UNION ALL
		SELECT Count(1) FROM tracks UNION ALL
		 SELECT Count(1) FROM keyPoints UNION ALL 
		SELECT id FROM gorp_migrations WHERE applied_at = (SELECT MAX(applied_at) FROM gorp_migrations)`)
	r.Next()
	r.Scan(&trackrecords)
	r.Next()
	r.Scan(&tracks)
	r.Next()
	r.Scan(&keypoints)
	r.Next()
	r.Scan(&latestMigration)
	t2 := time.Now().UnixNano()
	dbg.D(dTag, "Time for counts : ", (t2-t1)/1000/1000)
	return
}

// openDbCon tries to open database connection AND validates
// TODO: this should be done by crossplatform stuffs
func openDbCon(dbPath string) (database *sql.DB, err error) {

	database, err = sql.Open("SQLITE", "file:"+dbPath+"?Pooling=true") // &cache=shared

	if err != nil {
		dbg.E(dTag, "Failed to create DB handle at openDbCon() : ", err)
		return
	}
	if err = database.Ping(); err != nil {
		dbg.E(dTag, "Failed to keep connection alive at openDbCon() : ", err)
		return
	}
	var timeout int64
	err = database.QueryRow("PRAGMA busy_timeout").Scan(&timeout)
	if err != nil {
		dbg.E(dTag, "Error getting journal_mode!", err)
	} else if timeout < 10000 {
		dbg.W(dTag, "Setting journal_mode for %s to WAL!", dbPath)
		_, err = database.Exec(`PRAGMA journal_mode=WAL;
		PRAGMA automatic_index = 1;
		PRAGMA busy_timeout = 10000;
		PRAGMA mmap_size=10000000;
		PRAGMA temp_store=2;
		`)
		if err != nil {
			dbg.E(dTag, "Error setting pragmas!", err)
		}
	}
	_, err = database.Exec("PRAGMA cache_size=10000;")
	if err != nil {
		if err != nil {
			dbg.E(dTag, "Error setting cache size!", err)
		}
	}
	dbg.D(dTag, "Databaseconnection established", dbPath)

	return
}

// CreateNewLocationDb creates a new LocationDB with newest db-schema
func CreateNewLocationDb(dbPath string) (err error) {
	dbCon, err := openDbCon(dbPath)
	defer dbCon.Close()
	dbg.I(dTag, "Creating new User-DB at ", dbPath)
	if err != nil {
		return
	}
	defer dbCon.Close()

	numExec, err := ExecMigrations(dbCon)

	if err != nil {
		dbg.E(dTag, "Failed to create Location-DB. That is bad : ", err)
		return err
	}
	dbg.I(dTag, "Location DB created with %d migrations executed", numExec)
	return
}

// GetDbCSV returns CSV of trackrecords for device in timerange
func GetDbCSV(dbPath string, sinceTimeMillis int64, beforeTimeMillis int64, deviceId string,usrId int64) (csv string, err error) {

	var buf bytes.Buffer
	buf.WriteString("timeMillis,latitude,longitude,altitude,accuracy,provider,source,accuracyRating,speed")

	dbCon, err := GetLocationDb(dbPath,usrId)
	defer dbCon.Close()
	if err != nil {
		dbg.E(dTag, "Error getting location Db for CSV : %+v", err)
		return "", err
	}
	var entries []Location
	// if deviceId empty, put nothing
	var devId sql.NullInt64
	var iDevKey int
	if deviceId != "" {
		row := dbCon.QueryRow("SELECT _deviceId FROM devices WHERE desc=?", deviceId)

		err = row.Scan(&devId)
		if err != nil {
			return "", err
		}
		iDevKey = int(devId.Int64)
		dbg.D(dTag, "DeviceKey : %v, sinceTime : %v, beforeTime : %v", iDevKey, sinceTimeMillis, beforeTimeMillis)
	}
	// entries, err = GetLocDataByWhereString("timeMillis>=? AND timeMillis<? "+additional, db, sinceTimeMillis, beforeTimeMillis, iDevKey)
	entries, err = datapolish.GetTrackRecordsForDevice(sinceTimeMillis, beforeTimeMillis, iDevKey, dbCon)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		buf.WriteString(fmt.Sprintf("\r\n%d,%f,%f,%f,%f,%s,%d,%d,%f", entry.TimeMillis.Int64, entry.Latitude.Float64, entry.Longitude.Float64, entry.Altitude.Float64, entry.Accuracy.Float64, entry.Provider.String, entry.Source.Int64, entry.AccuracyRating.Int64, entry.Speed.Float64))
	}

	return buf.String(), err
}

// initNewDevice inserts new Device with description into DB
func initNewDevice(desc string, dbCon *sql.DB) (key int, err error) {
	var row = dbCon.QueryRow("SELECT MAX(_deviceId) FROM devices")
	var id sql.NullInt64

	err = row.Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		return -1, err
	}
	if !id.Valid {
		id.Int64 = 0
	}
	id.Int64++
	_, err = dbCon.Exec(`INSERT INTO devices(_deviceId,desc,colorId) VALUES(?,?,COALESCE(
	(SELECT _colorId FROM Colors WHERE (
		(SELECT COUNT(*) FROM Devices WHERE Devices.colorId=colors._colorId) = 0
) ORDER BY RANDOM() LIMIT 1),
(SELECT _colorId FROM Colors ORDER BY RANDOM() LIMIT 1)
))`, id, desc)

	return int(id.Int64), err
}

var EMissingHeading = errors.New("Missing Heading")
var EForbiddenHeading = errors.New("Missing Heading")

// InsertCSVToDb converts a csv into trackrecords-database-entries.
func InsertCSVToDb(data string, key string, usrId int64, dbCon *sql.DB) (cnt int, minTime int64, maxTime int64, err error) {

	row := dbCon.QueryRow("SELECT _deviceId FROM devices WHERE desc=?",
		key)
	minTime = 9223372036854775807
	maxTime = -9223372036854775807
	var devId sql.NullInt64
	err = row.Scan(&devId)

	var deviceId = 0
	if err != nil && err != sql.ErrNoRows {
		return 0, 0, 0, err
	}

	if !devId.Valid {
		deviceId, err = initNewDevice(key, dbCon)
		if err != nil {
			return 0, 0, 0, err
		}

	} else {
		deviceId = int(devId.Int64)
	}
	var curInsCnt = 0

	// TODO : Remove duplicates here or in a daily cleanup job or sth.?
	rd := csv.NewReader(strings.NewReader(data))

	headings := make(map[string]*int)
	requiredHeadings := []string{"timeMillis", "latitude", "longitude", "altitude", "accuracy", "speed"}
	allowedHeadings := map[string]struct{}{"timeMillis": struct{}{}, "latitude": struct{}{}, "longitude": struct{}{},
		"altitude": struct{}{}, "accuracy": struct{}{}, "provider": struct{}{}, "source": struct{}{},
		"accuracyRating": struct{}{}, "speed": struct{}{}}
	var qmString = "("

	first := true
	rCount := 0
	// http://stackoverflow.com/a/25192138/3085985 - insert performance
	valueStrings := make([]string, 0)
	valueArgs := make([]interface{}, 0)
	var recLen = 0
	var headingsString = ""
	var timeIdx = 0
	cnt = -1
	for {
		rCount++
		var record []string
		record, err = rd.Read()
		if err == io.EOF {
			err = nil
			break
		} else if err != nil {
			dbg.E(dTag, "Failed to read CSV : %v", err)
			return
		}

		if first {
			recLen = len(record)
			for i, h := range record {
				if _, ok := allowedHeadings[h]; !ok {
					dbg.I(dTag, "Forbidden header %v", h)
					err = EForbiddenHeading
					return
				}
				if h == "timeMillis" {
					timeIdx = i
				}
				headings[h] = &i
				if i != 0 {
					headingsString += ","
				}
				if i != 0 {
					qmString += ",?"
				} else {
					qmString += "?"
				}
				headingsString += h
			}
			headingsString += ",deviceId"
			qmString += ",?)"
			for _, h := range requiredHeadings {
				if headings[h] == nil {
					dbg.I(dTag, "Missing header %v", h)
					err = EMissingHeading
					return
				}
			}
			first = false
		} else {
			valueStrings = append(valueStrings, qmString)
			new := make([]interface{}, recLen+1)
			for i, v := range record {
				if i > recLen-1 {
					err = EMissingHeading
					return
				}
				new[i] = v
			}
			curTime, _err := strconv.ParseInt(record[timeIdx], 10, 64)
			if _err != nil {
				err = _err
				dbg.E(dTag, "Could not parse time %s at row %d : %s", record[timeIdx], rCount, err)
				return
			}
			if curTime == 0 {
				dbg.WTF(dTag, "How can a record have a timestamp of zero?", record)
				err = errors.New("Invalid data")
				return
			}
			if curTime < minTime {
				minTime = curTime
			}
			if curTime > maxTime {
				maxTime = curTime
			}
			curInsCnt += recLen + 1
			new[recLen] = deviceId
			valueArgs = append(valueArgs, new...)
		}
		if curInsCnt+recLen+1 > 999 {
			err = commitInsertCsv(headingsString, valueArgs, valueStrings, dbCon)
			curInsCnt = 0
			valueStrings = make([]string, 0)
			valueArgs = make([]interface{}, 0)
			if err != nil {
				return 0, 0, 0, err
			}
		}
		cnt++
	}

	if curInsCnt != 0 {
		err = commitInsertCsv(headingsString, valueArgs, valueStrings, dbCon)
	}

	return
}

// commitInsertCsv is a helper function to batch-insert into DB
func commitInsertCsv(headingsString string, valueArgs []interface{}, valueStrings []string, dbCon *sql.DB) error {

	stmt := fmt.Sprintf("INSERT INTO TrackRecords (%s) VALUES %s", headingsString, strings.Join(valueStrings, ","))

	_, err := dbCon.Exec(stmt, valueArgs...)
	return err
}
