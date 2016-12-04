package driverManager

import (
	"database/sql"
	"encoding/json"
	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/models"
	. "github.com/OpenDriversLog/goodl-lib/models/SQLite"
	. "github.com/OpenDriversLog/goodl-lib/tools"
)

type JSONDriversAnswer struct {
	models.JSONAnswer
	Drivers []*Driver
}

// JSONGetDrivers returns all drivers
func JSONGetDrivers(dbCon *sql.DB) (res JSONDriversAnswer, err error) {
	res = JSONDriversAnswer{}
	res.Drivers, err = GetDrivers(dbCon)
	if err != nil {
		dbg.E(TAG, "Error getting GetDrivers : ", err)
		err = nil
		res = GetBadJsonDriversManAnswer("Unknown error while getting drivers")
		return
	}
	res.Success = true
	return
}

// GetBadJsonDriversManAnswer returns a bad JSONDriversAnswer in case of an error
func GetBadJsonDriversManAnswer(message string) JSONDriversAnswer {
	return JSONDriversAnswer{
		JSONAnswer: models.GetBadJSONAnswer(message),
	}
}

// JSONCreateDriver creates a new driver.
func JSONCreateDriver(driverJson string, dbCon *sql.DB) (res models.JSONInsertAnswer, err error) {
	c := &Driver{}
	if driverJson == "" {
		res = models.GetBadJSONInsertAnswer(NoDataGiven)
		return
	}
	err = json.Unmarshal([]byte(driverJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in JSONCreateDriver : ", driverJson, err)
		res = models.GetBadJSONInsertAnswer("Invalid format")
		err = nil
		return
	}
	var key int64
	key, err = CreateDriver(c, dbCon)
	if err != nil {
		dbg.E(TAG, "Error in JSONCreateDriver CreateDriver: ", err)
		err = nil
		res = models.GetBadJSONInsertAnswer("Internal server error")
		return
	}
	res.LastKey = key
	c.Id = NInt64(res.LastKey)
	res.Success = true
	return

}

// JSONDeleteDriver deletes the given driver (by its ID)
func JSONDeleteDriver(driverJson string, dbCon *sql.DB) (res models.JSONDeleteAnswer, err error) {
	c := &Driver{}
	if driverJson == "" {
		res = models.GetBadJSONDeleteAnswer(NoDataGiven, -1)
		return
	}
	err = json.Unmarshal([]byte(driverJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in DeleteDriverJSON : ", driverJson, err)
		res = models.GetBadJSONDeleteAnswer("Invalid format", -1)
		err = nil
		return
	}
	var rowCount int64
	rowCount, err = DeleteDriver(int64(c.Id), dbCon)
	if err != nil {
		dbg.E(TAG, "Error in DeleteDriverJSON DeleteDriver: ", err)
		errMsg := "Internal server error"
		if err == ErrDriverBoundToCars {
			errMsg = "Driver still used for some cars or tracks - can't delete driver."
		}
		err = nil

		res = models.GetBadJSONDeleteAnswer(errMsg, int64(c.Id))
		return
	}
	res.RowCount = rowCount
	res.Id = int64(c.Id)
	res.Success = true
	return

}

const NoDataGiven = "Please fill at least one entry."

// JSONGetEmptyDriver returns JSONSelectAnswer with empty driver-object
func JSONGetEmptyDriver() (res models.JSONSelectAnswer, err error) {
	emptyDriver, err := GetEmptyDriver()
	if err != nil {
		dbg.E(TAG, "Error in GetEmptyContact: ", err)
	}
	res = models.GetGoodJSONSelectAnswer(emptyDriver)
	return
}

// JSONUpdateDriver updates the given driver
func JSONUpdateDriver(driverJson string, dbCon *sql.DB) (res models.JSONUpdateAnswer, err error) {
	c := &Driver{}
	if driverJson == "" {
		res = models.GetBadJSONUpdateAnswer(NoDataGiven, -1)
		return
	}
	err = json.Unmarshal([]byte(driverJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in UpdateDriverJSON : ", driverJson, err)
		res = models.GetBadJSONUpdateAnswer("Invalid format", -1)
		err = nil
		return
	}
	var rowCount int64
	rowCount, err = UpdateDriver(c, dbCon)
	if err != nil {
		if err == ErrNoChanges {
			err = nil
			res = models.GetBadJSONUpdateAnswer(NoDataGiven, int64(c.Id))
			return
		}
		dbg.E(TAG, "Error in JSONUpdateDriver UpdateDriver: ", err)
		err = nil
		res = models.GetBadJSONUpdateAnswer("Internal server error", int64(c.Id))
		return
	}
	res.RowCount = rowCount
	res.Id = int64(c.Id)
	res.Success = true
	return

}
