// Package deviceManager is responsible for CRUD devices.
package deviceManager

import (
	"database/sql"
	"errors"

	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/dbMan/helpers"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/colorManager"
	"github.com/OpenDriversLog/goodl-lib/models/SQLite"
)

const TAG = "goodl-lib/jsonApi/deviceManager"

// GetDevices returns all devices
func GetDevices(dbCon *sql.DB) (devices []*Device, err error) {

	return GetDevicesByWhere(dbCon,"")
}

// GetDevicesByWhere returns the devices matching the given where-string with the given parameters
func GetDevicesByWhere(dbCon *sql.DB, where string, params ...interface{}) (devices []*Device, err error) {
	devices = make([]*Device, 0)
	q := "SELECT _deviceId, desc, checked, colorId,carId,Guid FROM Devices"
	if where !="" {
		q += " WHERE " + where
	}
	res, err := dbCon.Query(q, params...)
	if err != nil {
		if err != nil {
			dbg.E(TAG, "unable to get Devices", err)
			return
		}
	}

	for res.Next() {
		device := &Device{Color: &colorManager.Color{}}
		err = res.Scan(&device.Id, &device.Description, &device.Checked, &device.Color.Id, &device.CarId,&device.Guid)
		if err != nil {
			dbg.E(TAG, "Unable to scan device!", err)
			return
		}
		if device.Color.Id > 0 {
			device.Color, err = colorManager.GetColor(int64(device.Color.Id), dbCon)
			if err != nil {
				dbg.E(TAG, "Unabled to get color for device!", err)
				return
			}
		}
		devices = append(devices, device)
	}
	return
}

// GetDeviceByGUID gets a device by its GUID
func GetDeviceByGUID(dbCon *sql.DB, guid string) (device *Device, err error) {
	devices, err := GetDevicesByWhere(dbCon,"guid=?",guid)
	if err != nil {
		return
	}
	if len(devices) == 0 {
		err = errors.New("No device with given GUID found")
		return
	}
	return devices[0],err
}

// CreateDevice creates a new device. Also picks a random preferably not-used color when no color was given.
func CreateDevice(device *Device, dbCon *sql.DB) (key int64, err error) {

	vals := []interface{}{device.Description}
	valString := "?"

	insFields := "desc"
	if device.Color != nil && device.Color.Id != 0 {
		insFields += ",colorId"
		valString += ",?"
		vals = append(vals, device.Color.Id)
	} else {
		insFields += ",colorId"
		valString += `,(SELECT COALESCE(
			(SELECT _colorId FROM Colors WHERE (
			(SELECT COUNT(*) FROM Devices WHERE Devices.colorId=colors._colorId) = 0
) ORDER BY RANDOM() LIMIT 1),
(SELECT _colorId FROM Colors ORDER BY RANDOM() LIMIT 1)))`
	}
	if device.Checked != 0 {
		insFields += ",checked"
		valString += ",?"
		vals = append(vals, device.Checked)
	}
	if device.CarId != 0 {
		insFields += ",carId"
		valString += ",?"
		vals = append(vals, device.CarId)
	}
	if device.Guid != "" {
		insFields += ",Guid"
		valString += ",?"
		vals = append(vals, device.Guid)
	}
	q := "INSERT INTO Devices(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for CreateDevices: %v ", err)
		return
	}
	key, err = res.LastInsertId()
	return
}

// UpdateDevice updates a device
func UpdateDevice(d *Device, dbCon *sql.DB) (rowCount int64, err error) {

	update := helpers.NewUpdateHelper(dbCon)
	if d.Checked != 0 {
		update.AppendNInt64("checked", &d.Checked)
	}
	if d.Color != nil && d.Color.Id != 0 {
		update.AppendNInt64("colorId", &d.Color.Id)
	}
	if d.Description != "" {
		update.AppendNString("desc", &d.Description)
	}
	if d.CarId != 0 {
		update.AppendNInt64("carId", &d.CarId)
	}
	if d.Guid != "" {
		if d.Guid=="-" {
			d.Guid = models.NString("")
		}
		update.AppendNString("Guid", &d.Guid)
	}
	// at least update one field to don't get errors ;)
	update.AppendNString("desc", &d.Description)

	res, err := update.ExecUpdate("Devices", "_deviceId=?", d.Id)
	if err != nil {
		dbg.E(TAG, "UpdateDevice: error execUpdate", err)
		return
	} else if rowCount, err = res.RowsAffected(); err != nil || rowCount == 0 {
		err = errors.New("did update nothing")
		return
	}
	if d.CarId > 0 {
		// TODO : Also make an option for changing carIds where carId is already set (for non-confirmed trips only)
		res, err = dbCon.Exec("UPDATE Tracks SET carId=? WHERE deviceId=? AND carId IS NULL")
		if err != nil {
			dbg.E(TAG, "Error updating Tracks with new carId : ", err)
			return
		}
		res, err = dbCon.Exec("UPDATE KeyPoints SET carId=? WHERE deviceId=? AND carId IS NULL")
		if err != nil {
			dbg.E(TAG, "Error updating KeyPoints with new carId : ", err)
			return
		}
	}
	return
}

var ErrDeviceBoundToData = errors.New("Device already has data - can't be deleted.")

// DeleteDevice deletes the device with the given ID
func DeleteDevice(id int64, dbCon *sql.DB) (rowCount int64, err error) {
	var res sql.Result
	var cnt = 0
	err = dbCon.QueryRow("SELECT COUNT(*) FROM TrackRecords WHERE deviceId=?", id).Scan(&cnt)
	if err != nil {
		dbg.E(TAG, "Error scanning for count of Tracks with device : ", err)
		return
	}
	if cnt != 0 {
		dbg.W(TAG, "Tried to device car %d with %d Tracks - we don't allow this.", id, cnt)
		err = ErrDeviceBoundToData
		return
	}
	res, err = dbCon.Exec("DELETE FROM Devices WHERE _deviceId=?", id)
	if err != nil {
		dbg.E(TAG, "Error in DeleteDevice : ", err)
	} else {
		rowCount, err = res.RowsAffected()
		if err != nil {
			dbg.E(TAG, "Error in DeleteDevice get RowsAffected : ", err)
		}
	}

	return
}

// GetEmptyDevice returns an empty device Object
func GetEmptyDevice() (device *Device, err error) {
	device = &Device{}
	return
}
