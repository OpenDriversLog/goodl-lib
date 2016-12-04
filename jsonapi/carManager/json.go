package carManager

import (
	"database/sql"
	"encoding/json"
	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/models"
	. "github.com/OpenDriversLog/goodl-lib/models/SQLite"
	. "github.com/OpenDriversLog/goodl-lib/tools"
)

type JSONCarsAnswer struct {
	models.JSONAnswer
	Cars []*Car
}

// JSONGetCars gets all cars as JSON
func JSONGetCars(dbCon *sql.DB) (res JSONCarsAnswer, err error) {
	res = JSONCarsAnswer{}
	res.Cars, err = GetCars(dbCon)
	if err != nil {
		dbg.E(TAG, "Error getting GetCars : ", err)
		err = nil
		res = GetBadJsonCarsManAnswer("Unknown error while getting cars")
		return
	}
	res.Success = true
	return
}

// GetBadJsonCarsManAnswer returns a bad JSONCarsAnswer if an error occured.
func GetBadJsonCarsManAnswer(message string) JSONCarsAnswer {
	return JSONCarsAnswer{
		JSONAnswer: models.GetBadJSONAnswer(message),
	}
}

// JSONCreateCar creates the given car.
func JSONCreateCar(carJson string, dbCon *sql.DB) (res models.JSONInsertAnswer, err error) {
	c := &Car{}
	if carJson == "" {
		res = models.GetBadJSONInsertAnswer(NoDataGiven)
		return
	}
	err = json.Unmarshal([]byte(carJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in JSONCreateCar : ", carJson, err)
		res = models.GetBadJSONInsertAnswer("Invalid format")
		err = nil
		return
	}
	var key int64
	key, err = CreateCar(c, dbCon)
	if err != nil {
		dbg.E(TAG, "Error in JSONCreateCar CreateCar: ", err)
		err = nil
		res = models.GetBadJSONInsertAnswer("Internal server error")
		return
	}
	res.LastKey = key
	c.Id = NInt64(res.LastKey)
	res.Success = true
	return

}

// JSONDeleteCar deletes the given car.
func JSONDeleteCar(carJson string, dbCon *sql.DB) (res models.JSONDeleteAnswer, err error) {
	c := &Car{}
	if carJson == "" {
		res = models.GetBadJSONDeleteAnswer(NoDataGiven, -1)
		return
	}
	err = json.Unmarshal([]byte(carJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in DeleteCarJSON : ", carJson, err)
		res = models.GetBadJSONDeleteAnswer("Invalid format", -1)
		err = nil
		return
	}
	var rowCount int64
	rowCount, err = DeleteCar(int64(c.Id), dbCon)
	if err != nil {
		dbg.E(TAG, "Error in DeleteCarJSON DeleteCar: ", err)

		errMsg := "Internal server error"
		if err == ErrCarBoundToTracks {
			errMsg = "Car already used for some tracks - can not delete car."
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

// JSONGetEmptyCar returns goodJsonAnswer with empty car-object
func JSONGetEmptyCar() (res models.JSONSelectAnswer, err error) {
	emptyCar, err := GetEmptyCar()
	if err != nil {
		dbg.E(TAG, "Error in GetEmptyContact: ", err)
	}
	res = models.GetGoodJSONSelectAnswer(emptyCar)
	return
}

// JSONUpdateCar updates the given car.
func JSONUpdateCar(carJson string, dbCon *sql.DB) (res models.JSONUpdateAnswer, err error) {
	c := &Car{}
	if carJson == "" {
		res = models.GetBadJSONUpdateAnswer(NoDataGiven, -1)
		return
	}
	err = json.Unmarshal([]byte(carJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in UpdateCarJSON : ", carJson, err)
		res = models.GetBadJSONUpdateAnswer("Invalid format", -1)
		err = nil
		return
	}
	var rowCount int64
	rowCount, err = UpdateCar(c, dbCon)
	if err != nil {
		if err == ErrNoChanges {
			err = nil
			res = models.GetBadJSONUpdateAnswer(NoDataGiven, int64(c.Id))
			return
		}
		dbg.E(TAG, "Error in JSONUpdateCar UpdateCar: ", err)
		err = nil
		res = models.GetBadJSONUpdateAnswer("Internal server error", int64(c.Id))
		return
	}
	res.RowCount = rowCount
	res.Id = int64(c.Id)
	res.Success = true
	return

}
