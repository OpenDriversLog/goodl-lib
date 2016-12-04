// Package carManager is responsible for CRUD cars.
package carManager

import (
	"database/sql"
	"errors"
	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/driverManager"
	. "github.com/OpenDriversLog/goodl-lib/tools"
)

const TAG = "goodl-lib/jsonApi/carManager"

// GetCarById gets a car by its ID.
func GetCarById(dbCon *sql.DB, carId int64) (car *Car, err error) {
	cars, err := GetCarsByWhere(dbCon, "_carId=?", carId)
	if err != nil {
		return
	}
	if len(cars) == 0 {
		err = errors.New("Car not found")
		return
	}
	car = cars[0]
	return
}

// GetCars returns all cars.
func GetCars(dbCon *sql.DB) (cars []*Car, err error) {
	return GetCarsByWhere(dbCon, "")
}

// GetCarsByWhere gets cars by given where-string / params
func GetCarsByWhere(dbCon *sql.DB, where string, params ...interface{}) (cars []*Car, err error) {
	cars = make([]*Car, 0)
	q := "SELECT _carId, type, plate, firstMileage, mileage, firstUseDate,ownerId FROM Cars"
	if where != "" {
		q += " WHERE " + where
	}
	res, err := dbCon.Query(q, params...)
	if err != nil {
		if err != nil {
			dbg.E(TAG, "unable to get Cars", err)
			return
		}
	}

	for res.Next() {
		car := &Car{}
		var ownerId sql.NullInt64
		err = res.Scan(&car.Id, &car.Type, &car.Plate, &car.FirstMileage, &car.Mileage, &car.FirstUseDate, &ownerId)
		if err != nil {
			dbg.E(TAG, "Unable to scan car!", err)
			return
		}
		if ownerId.Valid {
			var owner *driverManager.Driver
			owner, err = driverManager.GetDriverById(ownerId.Int64, dbCon)
			if err != nil {
				if err == sql.ErrNoRows {
					dbg.W(TAG, "Could not find owner/driver with Id : ", ownerId.Int64)
				} else {
					dbg.E(TAG, "Unable to get owner!", err)
					return
				}
			}
			car.Owner = *owner
		}
		cars = append(cars, car)
	}
	return
}

// CreateCar creates a new car
func CreateCar(car *Car, dbCon *sql.DB) (key int64, err error) {

	vals := []interface{}{car.Plate}
	valString := "?"

	insFields := "plate"
	if car.Owner.Id != 0 {
		insFields += ",ownerId"
		valString += ",?"
		vals = append(vals, car.Owner.Id)
	}
	if car.Type != "" {
		insFields += ",type"
		valString += ",?"
		vals = append(vals, car.Type)
	}
	if car.FirstMileage != 0 {
		insFields += ",firstMileage"
		valString += ",?"
		vals = append(vals, car.FirstMileage)
	}
	if car.Mileage != 0 {
		insFields += ",mileage"
		valString += ",?"
		vals = append(vals, car.Mileage)
	}
	if car.FirstUseDate != 0 {
		insFields += ",firstUseDate"
		valString += ",?"
		vals = append(vals, car.FirstUseDate)
	}
	q := "INSERT INTO Cars(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for CreateCars: %v ", err)
		return
	}

	key, err = res.LastInsertId()
	return
}

// UpdateCar updates a car
func UpdateCar(c *Car, dbCon *sql.DB) (rowCount int64, err error) {

	vals := []interface{}{}
	firstVal := true
	valString := ""

	if c.Type != "" {
		AppendNStringUpdateField("type", &c.Type, &firstVal, &vals, &valString)
	}
	if c.Owner.Id != 0 {
		AppendNInt64UpdateField("ownerId", &c.Owner.Id, &firstVal, &vals, &valString)
	}
	if c.Plate != "" {
		AppendNStringUpdateField("plate", &c.Plate, &firstVal, &vals, &valString)
	}
	if int64(c.FirstMileage) != 0 {
		AppendNInt64UpdateField("firstMileage", &c.FirstMileage, &firstVal, &vals, &valString)
	}
	if int64(c.Mileage) != 0 {
		AppendNInt64UpdateField("mileage", &c.Mileage, &firstVal, &vals, &valString)
	}
	if int64(c.FirstUseDate) != 0 {
		AppendNInt64UpdateField("firstUseDate", &c.FirstUseDate, &firstVal, &vals, &valString)
	}

	if firstVal {
		err = ErrNoChanges
		return
	}
	q := "UPDATE Cars SET " + valString + " WHERE _carId=?"
	vals = append(vals, c.Id)
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for UpdateCar: %v ", err)

		return
	}
	rowCount, err = res.RowsAffected()

	return
}

var ErrCarBoundToTracks = errors.New("Car already bound to tracks - can't delete car")

// DeleteCar deletes the car with the given ID.
func DeleteCar(id int64, dbCon *sql.DB) (rowCount int64, err error) {
	var res sql.Result
	var cnt = 0
	err = dbCon.QueryRow("SELECT COUNT(*) FROM Tracks WHERE carId=?", id).Scan(&cnt)
	if err != nil {
		dbg.E(TAG, "Error scanning for count of Tracks with car : ", err)
		return
	}
	if cnt != 0 {
		dbg.W(TAG, "Tried to delete car %d with %d Tracks - we don't allow this.", id, cnt)
		err = ErrCarBoundToTracks
		return
	}

	res, err = dbCon.Exec("DELETE FROM CARS WHERE _carId=?", id)
	if err != nil {
		dbg.E(TAG, "Error in DeleteCar : ", err)
	} else {
		rowCount, err = res.RowsAffected()
		if err != nil {
			dbg.E(TAG, "Error in DeleteCar get RowsAffected : ", err)
		}
	}

	return
}

// GetEmptyCar returns an empty car Object
func GetEmptyCar() (car *Car, err error) {
	car = &Car{}
	return
}
