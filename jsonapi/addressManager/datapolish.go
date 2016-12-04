package addressManager

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"strings"

	"github.com/Compufreak345/dbg"
)

var ErrEmptyFilter = errors.New("Empty filter")

// UpdateAllKeyPointsForGeoZones updates all keypoints inside the given geoZones (for finding automatic contacts)
func UpdateAllKeyPointsForGeoZones(geoZoneIds []int64, dbCon *sql.DB) (err error) {
	if len(geoZoneIds) == 0 {
		return ErrEmptyFilter
	}
	gInString := getInString(geoZoneIds)
	return UpdateKeyPointsForGeoZonesByWhereQuery("KeyPoints_GeoFenceRegions.geoFenceRegionId IN ("+gInString+")", "G._geoFenceRegionId IN ("+gInString+")", dbCon)
}

// UpdateKeyPointsForAllGeozones updates the given KeyPoints for all GeoZones (for finding automatic contacts)
func UpdateKeyPointsForAllGeozones(keyPointIds []int64, dbCon *sql.DB) (err error) {
	if len(keyPointIds) == 0 {
		return ErrEmptyFilter
	}
	kInString := getInString(keyPointIds)
	return UpdateKeyPointsForGeoZonesByWhereQuery("keyPointId IN ("+kInString+")", "K._keyPointId IN ("+kInString+")", dbCon)
}

// UpdateAllKeyPointsForAllGeozones updates all keypoints for all geoZones (for finding automatic contacts)
func UpdateAllKeyPointsForAllGeozones(dbCon *sql.DB) (err error) {
	return UpdateKeyPointsForGeoZonesByWhereQuery("", "", dbCon)
}

// UpdateKeyPointsForGeoZones updates the given keypoints in the given geoZones (for finding automatic contacts)
func UpdateKeyPointsForGeoZones(geoZoneIds []int64, keyPointIds []int64, dbCon *sql.DB) (err error) {
	if len(geoZoneIds) == 0 || len(keyPointIds) == 0 {
		return ErrEmptyFilter
	}
	kInString := getInString(keyPointIds)
	gInString := getInString(geoZoneIds)
	return UpdateKeyPointsForGeoZonesByWhereQuery("keyPointId IN ("+kInString+
		") AND KeyPoints_GeoFenceRegions.geoFenceRegionId IN ("+gInString+")", "K._keyPointId IN ("+kInString+
		") AND G._geoFenceRegionId IN ("+gInString+")", dbCon)
}

// UpdateKeyPointsForGeoZonesByWhereQuery updates all keypoints matching the given geozones, updating the KeyPoint-GeofenceRegion-mapping-table.
// firstWhere referencing table KeyPoints_GeoFenceRegions, secondWhere referencing Keypoints/GeoFenceRegions-Table directly - BOTH are required and need to contain the same results!
func UpdateKeyPointsForGeoZonesByWhereQuery(firstWhere string, secondWhere string, dbCon *sql.DB) (err error) {
	var cmd string
	if (firstWhere != "" && secondWhere == "") || (secondWhere != "" && firstWhere == "") {
		return ErrEmptyFilter
	}

	if firstWhere != "" {
		cmd = fmt.Sprintf(keyPointsUpdateQuery, " WHERE "+firstWhere, " AND "+secondWhere)
	} else {
		cmd = fmt.Sprintf(keyPointsUpdateQuery, "", "")
	}

	var res sql.Result
	tx, err := dbCon.Begin()
	if err != nil {
		dbg.E(TAG, "Error starting transaction : ", err)
		return
	}
	cmd = strings.Replace(cmd, "\n", " ", -1)
	dbg.D(TAG, "I will execute : ", cmd)
	res, err = dbCon.Exec(cmd)

	if err != nil {
		dbg.E(TAG, "UpdateKeyPointsForGeoZonesByWhereQuery (%v,%v,dbCon) failed at deleting : %v", firstWhere, secondWhere, err)
		return
	}

	var affRows int64
	affRows, err = res.RowsAffected()

	if err != nil {
		dbg.E(TAG, "UpdateKeyPointsForGeoZonesByWhereQuery (%v,%v,dbCon,%v) failed while getting RowsAffected for delete : %v", firstWhere, secondWhere)
		return
	}
	dbg.D(TAG, dbg.KCYN+"Changed some stuff regarding at least "+strconv.FormatInt(affRows, 10)+" GeoFence-related entries")

	sContactsQuery := `SELECT GROUP_CONCAT(tripId) AS TRIPS,_ContactId FROM KeyPoints_GeoFenceRegions
LEFT JOIN Tracks T ON T.StartKeyPointId=KeyPoints_GeoFenceRegions.keyPointId
 LEFT JOIN Address_GeoFenceRegion AG ON AG.geoFenceRegionId=KeyPoints_GeoFenceRegions.geoFenceRegionId
 LEFT JOIN Trips_Start_EndTrack TSE ON TSE.StartTrackId=T._TrackId
 LEFT JOIN CONTACTS C ON AG.addressId=C.addressId
 WHERE  %v _trackId IS NOT NULL AND _contactId IS NOT NULL AND C.disabled=0 GROUP BY _contactId`
	if firstWhere != "" {
		sContactsQuery = fmt.Sprintf(sContactsQuery, firstWhere+" AND ")
	} else {
		sContactsQuery = fmt.Sprintf(sContactsQuery, "")

	}
	eContactsQuery := `SELECT GROUP_CONCAT(tripId) AS TRIPS,_ContactId FROM KeyPoints_GeoFenceRegions
	LEFT JOIN Tracks T ON T.EndKeyPointId=KeyPoints_GeoFenceRegions.keyPointId
	LEFT JOIN Address_GeoFenceRegion AG ON AG.geoFenceRegionId=KeyPoints_GeoFenceRegions.geoFenceRegionId
	LEFT JOIN Trips_START_EndTrack TSE ON TSE.EndTrackId=T._TrackId
	LEFT JOIN CONTACTS C ON AG.addressId=C.addressId
	WHERE %v  _trackId IS NOT NULL AND _contactId IS NOT NULL  AND C.disabled=0 GROUP BY _contactId`
	if firstWhere != "" {
		eContactsQuery = fmt.Sprintf(eContactsQuery, firstWhere+" AND ")
	} else {
		eContactsQuery = fmt.Sprintf(eContactsQuery, "")
	}
	var rows *sql.Rows

	q := ""
	var tIds string
	var contactId int64
	pms := make([]interface{}, 0)
	var rc int64  //rowCount
	var trc int64 //temporary rowCount
	var r sql.Result

	/**
	Set StartContactId for Trips without StartContact, but matching the new/updated KeyPoint/GeoZone
	*/
	rows, err = dbCon.Query(sContactsQuery)
	if err != nil {
		dbg.E(TAG, "Error getting new StartContacts : ", err)
		return
	}
	for rows.Next() {
		err = rows.Scan(&tIds, &contactId)
		if err != nil {
			dbg.E(TAG, "Error scanning StartContactsRow : ", err)
			return
		}
		q += fmt.Sprintf("UPDATE TRIPS SET startContactId=? WHERE _tripId IN(%v) AND startContactId IS NULL;", tIds)
		pms = append(pms, contactId)
		if len(pms) > 500 {
			r, err = dbCon.Exec(q, pms...)
			if err != nil {
				dbg.E(TAG, "Error setting new startContactIds : ", err)
				return
			}
			trc, err = r.RowsAffected()
			if err != nil {
				dbg.E(TAG, "Error getting RosAffected for new startContactIds : ", err)
				return
			}
			rc += trc
			q = ""
			pms = make([]interface{}, 0)
		}
	}
	if len(pms) > 0 {
		r, err = dbCon.Exec(q, pms...)
		if err != nil {
			dbg.E(TAG, "Error setting new startContactIds : ", err)
			return
		}
		trc, err = r.RowsAffected()
		if err != nil {
			dbg.E(TAG, "Error getting RowsAffected for new startContactIds : ", err)
			return
		}
		rc += trc
	}
	dbg.I(TAG, "Updated %d trips with new startContactIds ", rc)

	/**
	Set EndContactId for Trips without EndContact, but matching the new/updated KeyPoint/GeoZone
	*/
	rows, err = dbCon.Query(eContactsQuery)
	if err != nil {
		dbg.E(TAG, "Error getting new EndContacts : ", err)
		return
	}
	for rows.Next() {
		err = rows.Scan(&tIds, &contactId)
		if err != nil {
			dbg.E(TAG, "Error scanning EndContactsRow : ", err)
			return
		}
		q += fmt.Sprintf("UPDATE TRIPS SET endContactId=? WHERE _tripId IN(%v) AND endContactId IS NULL;", tIds)
		pms = append(pms, contactId)
		if len(pms) > 500 {
			r, err = dbCon.Exec(q, pms...)
			if err != nil {
				dbg.E(TAG, "Error setting new endContactIds : ", err)
				return
			}
			trc, err = r.RowsAffected()
			if err != nil {
				dbg.E(TAG, "Error getting RosAffected for new endContactIds : ", err)
				return
			}
			rc += trc
			q = ""
			pms = make([]interface{}, 0)
		}
	}
	if len(pms) > 0 {
		r, err = dbCon.Exec(q, pms...)
		if err != nil {
			dbg.E(TAG, "Error setting new endContactIds : ", err)
			return
		}
		trc, err = r.RowsAffected()
		if err != nil {
			dbg.E(TAG, "Error getting RowsAffected for new endContactIds : ", err)
			return
		}
		rc += trc
	}
	dbg.I(TAG, "Updated %d trips with new endContactIds", rc)
	err = tx.Commit()
	if err != nil {
		dbg.E(TAG, "Error commiting transaction : ", err)
	}

	return
}

// getInString converts a collection of ids to a comma-separated string with the ids.
func getInString(ids []int64) (res string) {
	var buf bytes.Buffer
	first := true
	for _, v := range ids {
		if first {
			first = false
		} else {
			buf.WriteString(",")
		}
		buf.WriteString(strconv.FormatInt(v, 10))

	}
	return buf.String()
}

const keyPointsUpdateQuery = `
DELETE FROM KeyPoints_GeoFenceRegions %v;
INSERT INTO KeyPoints_GeoFenceRegions(keyPointId,geoFenceRegionId)

SELECT K._keyPointId,G._geoFenceRegionId FROM
GeoFenceRegions G
LEFT JOIN KeyPoints K
ON (
K.latitude>G.OuterMinLat AND
K.latitude<G.OuterMaxLat AND
K.longitude>G.OuterMinLon AND
K.longitude<G.OuterMaxLon
)
LEFT JOIN Addresses A  ON K.addressId=A._addressId
WHERE K._keyPointId IS NOT NULL %v;
`
