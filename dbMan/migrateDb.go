// responsible for updating the database based on the files in the "migrations"-folder.
package dbMan

import (
	"database/sql"
	"path"
	"runtime"

	"math"

	"github.com/Compufreak345/dbg"
	migrate "github.com/fschl/sql-migrate"
	"github.com/OpenDriversLog/goodl-lib/datapolish"
)

// var existingDbs map[int64]bool

const mdbTag = "goodl-lib/migrateDb.go"

// ExecMigrations executes all existing migrations
func ExecMigrations(db *sql.DB) (int, error) {

	_, filename, _, _ := runtime.Caller(1)
	rootDir := path.Dir(filename)

	migrations := &migrate.FileMigrationSource{
		Dir: rootDir + "/migrations",
	}
	dbg.D(mdbTag, "migrations dir:", migrations.Dir)

	rec, err := migrate.GetMigrationRecords(db, "sqlite3")
	dbg.D(mdbTag, "migration records & migrations: ", rec, migrations)

	n, err := migrate.Exec(db, "sqlite3", migrations, migrate.Up)
	if err != nil {
		dbg.E(mdbTag, "Failed to migrate Database up() : ", err)
		// TODO: Handle errors!
	}

	return n, err
}

var checkedDbPaths = make(map[string]bool)

// CheckIfUpgradeNeeded checks for old dbSchema and if upgrade is required
// Assuming, old DB hat no tracks, keypoints, etc. => we process all those things
func CheckIfUpgradeNeeded(dbCon *sql.DB,usrId int64, dbPath string) error {
	if checkedDbPaths[dbPath] {
		return nil
	}
	checkedDbPaths[dbPath] = true
	// does it have correct tracksDB?
	rec, err := migrate.GetMigrationRecords(dbCon, "sqlite3")
	dbg.D(mdbTag, "checkifupgradeneeded migration records: ", rec, len(rec))

	copyOldData := false

	var dmp sql.NullString
	e := dbCon.QueryRow("SELECT id from gorp_migrations WHERE id='1_initial.sql'").Scan(&dmp)
	if e != nil {
		if e != sql.ErrNoRows {
			dbg.E(mdbTag, "Error trying to check migrations for old entries : ", e)
		}
	} else {
		_, e = dbCon.Exec(`
			DELETE FROM gorp_migrations where substr(id,0,3)='00';
			UPDATE gorp_migrations SET id='001_initial.sql' WHERE id='1_initial.sql';
UPDATE gorp_migrations SET id='002_tracksDbs.sql' WHERE id='2_tracksDbs.sql';
UPDATE gorp_migrations SET id='003_trackGroups.sql' WHERE id='3_trackGroups.sql';
UPDATE gorp_migrations SET id='004_contactManager.sql' WHERE id='4_contactManager.sql';
UPDATE gorp_migrations SET id='005_addressAutoIncrement.sql' WHERE id='5_addressAutoIncrement.sql';
UPDATE gorp_migrations SET id='006_HouseNumberInView.sql' WHERE id='6_HouseNumberInView.sql';
UPDATE gorp_migrations SET id='007_TripTypeIdNotNilInContacts.sql' WHERE id='7_TripTypeIdNotNilInContacts.sql';
UPDATE gorp_migrations SET id='008_addressTitle.sql' WHERE id='8_addressTitle.sql';
UPDATE  gorp_migrations SET id='009_GeoFenceColor.sql' WHERE id='9_GeoFenceColor.sql';
			UPDATE gorp_migrations SET id='010_KeyPointsToGeoFenceRegion.sql' WHERE id='10_KeyPointsToGeoFenceRegion.sql'

			`)
		if e != nil {
			dbg.E(mdbTag, "Error updating migrations to new entries : ", e)
			return e
		}

	}
	dbCon.Exec("UPDATE gorp_migrations SET id='017_FixTripHistoryTrigger.sql' WHERE id='016_FixTripHistoryTrigger.sql'")
	rows, err := dbCon.Query("SELECT _id, deviceId FROM TrackRecords WHERE 0=1")

	if err != nil { // seems like it has the old version with "deviceKey" instead of deviceId

		dbg.I(mdbTag, "res select deviceId from devices", rows)
		res, err := dbCon.Exec("ALTER TABLE `TrackRecords` RENAME TO `TrackRecords_old`")
		if err != nil {
			dbg.E(mdbTag, "failed to rename trackrecords to _old", err)
			return err
		} else {
			copyOldData = true
			dbg.I(mdbTag, "successfully renamed old track records", res)
		}
	}

	res, err := ExecMigrations(dbCon)
	if err != nil {
		dbg.E(mdbTag, "failed to use migrations after renaming trackrecords", err)
		return err
	} else {
		dbg.I(mdbTag, "successfully migrated up", res)
	}

	if copyOldData {
		// check if there was data in old TrackRecords, copy if neccessary
		var countOldTrackRecords int
		var countNewTrackRecords int
		dbCon.QueryRow("SELECT Count(*) FROM TrackRecords_old").Scan(&countOldTrackRecords)

		dbCon.QueryRow("SELECT Count(*), deviceKey FROM TrackRecords").Scan(&countNewTrackRecords)
		dbg.I(mdbTag, "found trackRecord Counts: old %d new %d", countOldTrackRecords, countNewTrackRecords)

		if countOldTrackRecords != 0 && countNewTrackRecords != countOldTrackRecords { // copy from old db
			result, err := dbCon.Exec("INSERT INTO TrackRecords" +
				" (deviceId,timeMillis,latitude,longitude,altitude,accuracy,provider,source,accuracyRating,speed) " +
				"SELECT deviceKey AS deviceId,timeMillis,latitude,longitude,altitude,accuracy,provider,source,accuracyRating,speed " +
				"FROM TrackRecords_old")
			if err != nil {
				dbg.E(mdbTag, "failed to copy old trackRecords", err)
				return err
			} else {
				count, _ := result.RowsAffected()
				dbg.I(mdbTag, "successfully copied old track records... going to build tracks & keypoints", count)
				result, err = dbCon.Exec("DROP TABLE TrackRecords_old")
				if err != nil {
					dbg.E(mdbTag, "failed to drop old trackRecords", err)
					return err
				}
			}
		} else {
			dbg.V(mdbTag, "found deviceId from trackrecords.. checking if processing required", rows)
		}

		deviceMap, err := datapolish.GetDeviceStrings(dbCon)
		if err != nil {
			dbg.E(mdbTag, "failed to get DeviceMap in preperation to process data", err)
			return err
		} else {
			for key, _ := range deviceMap {
				dbg.I(mdbTag, "successfully got devicelist (%d), starting processing", len(deviceMap), key)

				err = datapolish.ProcessGPSData(0, 0, key, false,-1,nil,nil, dbCon)
			}
			err = nil
		}
	} // trackRecords had old version (deviceKey)

	deviceMap, err := datapolish.GetDeviceStrings(dbCon)
	if err != nil {
		dbg.E(mdbTag, "failed to get DeviceMap in preperation to process data", err)
		return err
	} else {
		for key, _ := range deviceMap {
			tracks := 0
			dbCon.QueryRow("SELECT Count(*) FROM Tracks WHERE deviceId=?", key).Scan(&tracks)

			dbg.I(mdbTag, "successfully got %d devices, starting processing %d ...", len(deviceMap), key)
			if tracks == 0 {
				err = datapolish.ProcessGPSData(0, math.MaxInt64, key, false,usrId,nil,nil, dbCon)
			}
		}
		err = nil
	}

	return nil
}
