package deviceManager

import (
	"database/sql"
	"encoding/json"
	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/models"
	. "github.com/OpenDriversLog/goodl-lib/models/SQLite"
	. "github.com/OpenDriversLog/goodl-lib/tools"
)

type JSONDevicesAnswer struct {
	models.JSONAnswer
	Devices []*Device
}

// JSONGetDevices returns all devices.
func JSONGetDevices(dbCon *sql.DB) (res JSONDevicesAnswer, err error) {
	res = JSONDevicesAnswer{}
	res.Devices, err = GetDevices(dbCon)
	if err != nil {
		dbg.E(TAG, "Error getting GetDevices : ", err)
		err = nil
		res = GetBadJsonDevicesManAnswer("Unknown error while getting devices")
		return
	}
	res.Success = true
	return
}


// GetBadJsonDevicesManAnswer returns a bad JSONDevicesAnswer in case of an error
func GetBadJsonDevicesManAnswer(message string) JSONDevicesAnswer {
	return JSONDevicesAnswer{
		JSONAnswer: models.GetBadJSONAnswer(message),
	}
}

// JSONCreateDevice creates the given device.
func JSONCreateDevice(deviceJson string, dbCon *sql.DB) (res models.JSONInsertAnswer, err error) {
	c := &Device{}
	if deviceJson == "" {
		res = models.GetBadJSONInsertAnswer(NoDataGiven)
		return
	}
	err = json.Unmarshal([]byte(deviceJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in JSONCreateDevice : ", deviceJson, err)
		res = models.GetBadJSONInsertAnswer("Invalid format")
		err = nil
		return
	}
	var key int64
	key, err = CreateDevice(c, dbCon)
	if err != nil {
		dbg.E(TAG, "Error in JSONCreateDevice CreateDevice: ", err)
		err = nil
		res = models.GetBadJSONInsertAnswer("Internal server error")
		return
	}
	res.LastKey = key
	c.Id = NInt64(res.LastKey)
	res.Success = true
	return

}

// JSONDeleteDevice deletes the given device.
func JSONDeleteDevice(deviceJson string, dbCon *sql.DB) (res JSONDeleteDeviceAnswer, err error) {
	c := &Device{}
	if deviceJson == "" {
		res = JSONDeleteDeviceAnswer{JSONDeleteAnswer:models.GetBadJSONDeleteAnswer(NoDataGiven, -1)}
		return
	}
	err = json.Unmarshal([]byte(deviceJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in DeleteDeviceJSON : ", deviceJson, err)
		res = JSONDeleteDeviceAnswer{JSONDeleteAnswer:models.GetBadJSONDeleteAnswer("Invalid format", -1)}
		err = nil
		return
	}
	var rowCount int64
	rowCount, err = DeleteDevice(int64(c.Id), dbCon)
	if err != nil {
		dbg.E(TAG, "Error in DeleteDeviceJSON DeleteDevice: ", err)

		errMsg := "Internal server error"
		if err == ErrDeviceBoundToData {
			errMsg = "Device already has data - can't be deleted."
		}
		err = nil
		res = JSONDeleteDeviceAnswer{JSONDeleteAnswer:models.GetBadJSONDeleteAnswer(errMsg, int64(c.Id))}
		return
	}
	res.RowCount = rowCount
	res.Id = int64(c.Id)
	res.Guid = string(c.Guid)
	res.Success = true
	return

}

const NoDataGiven = "Please fill at least one entry."

// JSONGetEmptyDevice goodJsonAnswer with empty device-object.
func JSONGetEmptyDevice() (res models.JSONSelectAnswer, err error) {
	emptyDevice, err := GetEmptyDevice()
	if err != nil {
		dbg.E(TAG, "Error in GetEmptyContact: ", err)
	}
	res = models.GetGoodJSONSelectAnswer(emptyDevice)
	return
}

// JSONUpdateDevice updates the given device.
func JSONUpdateDevice(deviceJson string, dbCon *sql.DB) (res models.JSONUpdateAnswer, err error) {
	c := &Device{}
	if deviceJson == "" {
		res = models.GetBadJSONUpdateAnswer(NoDataGiven, -1)
		return
	}
	err = json.Unmarshal([]byte(deviceJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in UpdateDeviceJSON : ", deviceJson, err)
		res = models.GetBadJSONUpdateAnswer("Invalid format", -1)
		err = nil
		return
	}
	var rowCount int64
	rowCount, err = UpdateDevice(c, dbCon)
	if err != nil {
		if err == ErrNoChanges {
			err = nil
			res = models.GetBadJSONUpdateAnswer(NoDataGiven, int64(c.Id))
			return
		}
		dbg.E(TAG, "Error in JSONUpdateDevice UpdateDevice: ", err)
		err = nil
		res = models.GetBadJSONUpdateAnswer("Internal server error", int64(c.Id))
		return
	}
	res.RowCount = rowCount
	res.Id = int64(c.Id)
	res.Success = true
	return

}
