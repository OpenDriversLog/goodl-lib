// Package driverManager is responsible for CRUD drivers.
package driverManager

import (
	"database/sql"
	"errors"
	"github.com/Compufreak345/dbg"
	. "github.com/OpenDriversLog/goodl-lib/tools"

	"github.com/OpenDriversLog/goodl-lib/jsonapi/addressManager"
	. "github.com/OpenDriversLog/goodl-lib/models/SQLite"
)

const TAG = "goodl-lib/jsonApi/driverManager"
const SelectColumns = "_driverId,priority,name,additional,_addressId,street,postal,city,additional1,additional2,latitude,longitude,HouseNumber,title,fuel"

// GetDrivers returns all drivers
func GetDrivers(dbCon *sql.DB) (drivers []*Driver, err error) {
	drivers = make([]*Driver, 0)
	q := "SELECT " + SelectColumns + " FROM Drivers LEFT JOIN Addresses ON _addressId=addressId"

	res, err := dbCon.Query(q)
	if err != nil {
		if err != nil {
			dbg.E(TAG, "unable to get Drivers", err)
			return
		}
	}

	for res.Next() {
		driver := &Driver{Address: addressManager.Address{}}
		err = res.Scan(&driver.Id, &driver.Priority, &driver.Name, &driver.Additional, &driver.Address.Id, &driver.Address.Street, &driver.Address.Postal, &driver.Address.City, &driver.Address.Additional1, &driver.Address.Additional2, &driver.Address.Latitude, &driver.Address.Longitude, &driver.Address.HouseNumber, &driver.Address.Title, &driver.Address.Fuel)
		if err != nil {
			dbg.E(TAG, "Unable to scan driver!", err)
			return
		}
		drivers = append(drivers, driver)
	}
	return
}

// GetDriverById returns the driver with the given ID
func GetDriverById(id int64, dbCon *sql.DB) (driver *Driver, err error) {
	driver = &Driver{Address: addressManager.Address{}}
	q := "SELECT " + SelectColumns + " FROM Drivers LEFT JOIN Addresses ON _addressId=addressId WHERE _driverId=?"

	err = dbCon.QueryRow(q, id).Scan(&driver.Id, &driver.Priority, &driver.Name, &driver.Additional, &driver.Address.Id, &driver.Address.Street, &driver.Address.Postal, &driver.Address.City, &driver.Address.Additional1, &driver.Address.Additional2, &driver.Address.Latitude, &driver.Address.Longitude, &driver.Address.HouseNumber, &driver.Address.Title, &driver.Address.Fuel)
	if err != nil {
		dbg.E(TAG, "unable to get Driver", err)
		return
	}
	return
}

// CreateDriver creates a new driver
func CreateDriver(driver *Driver, dbCon *sql.DB) (key int64, err error) {

	vals := []interface{}{driver.Name}
	valString := "?"

	insFields := "name"
	if driver.Additional != "" {
		insFields += ",additional"
		valString += ",?"
		vals = append(vals, driver.Additional)
	}
	if driver.Priority != 0 {
		insFields += ",priority"
		valString += ",?"
		vals = append(vals, driver.Priority)
	}
	if driver.Address.City != "" {
		var id int64
		id, err = addressManager.CreateGeoZoneAddress(&driver.Address, dbCon)
		if err != nil {
			dbg.E(TAG, "Error creating address for driver", err)
			return
		}
		insFields += ",addressId"
		valString += ",?"
		driver.Address.Id = NInt64(id)
		vals = append(vals, driver.Address.Id)
	}

	q := "INSERT INTO Drivers(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for CreateDrivers: %v ", err)
		return
	}
	key, err = res.LastInsertId()
	return
}

// UpdateDriver updates a driver
func UpdateDriver(c *Driver, dbCon *sql.DB) (rowCount int64, err error) {

	vals := []interface{}{}
	firstVal := true
	valString := ""

	if c.Address.City != "" {
		var id int64
		id, err = addressManager.CreateGeoZoneAddress(&c.Address, dbCon)
		if err != nil {
			dbg.E(TAG, "Error creating address for driver", err)
			return
		}
		c.Address.Id = NInt64(id)
		AppendNInt64UpdateField("addressId", &c.Address.Id, &firstVal, &vals, &valString)
	}
	if c.Name != "" {
		AppendNStringUpdateField("name", &c.Name, &firstVal, &vals, &valString)
	}
	if c.Additional != "" {
		AppendNStringUpdateField("additional", &c.Additional, &firstVal, &vals, &valString)
	}
	if c.Priority != 0 {
		AppendNInt64UpdateField("priority", &c.Priority, &firstVal, &vals, &valString)
	}

	if firstVal {
		err = ErrNoChanges
		return
	}
	q := "UPDATE Drivers SET " + valString + " WHERE _driverId=?"
	vals = append(vals, c.Id)
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for UpdateDriver: %v ", err)

		return
	}
	rowCount, err = res.RowsAffected()

	return
}

var ErrDriverBoundToCars = errors.New("Driver still used for some cars or tracks - can't delete driver.")

// DeleteDriver deletes the driver with the given ID
func DeleteDriver(id int64, dbCon *sql.DB) (rowCount int64, err error) {
	var res sql.Result
	var cnt = 0
	err = dbCon.QueryRow("SELECT COUNT(*) FROM Cars WHERE ownerId=?", id).Scan(&cnt)
	if err != nil {
		dbg.E(TAG, "Error scanning for count of Cars with driver : ", err)
		return
	}
	var cnt2 = 0
	err = dbCon.QueryRow("SELECT COUNT(*) FROM Trips WHERE driverId=?", id).Scan(&cnt2)
	if err != nil {
		dbg.E(TAG, "Error scanning for count of Trips with driver : ", err)
		return
	}
	if cnt != 0 || cnt2 != 0 {
		dbg.W(TAG, "Tried to delete driver %d with %d Cars / %d trips - we don't allow this.", id, cnt, cnt2)
		err = ErrDriverBoundToCars
		return
	}

	res, err = dbCon.Exec("DELETE FROM Drivers WHERE _driverId=?", id)
	if err != nil {
		dbg.E(TAG, "Error in DeleteDriver : ", err)
	} else {
		rowCount, err = res.RowsAffected()
		if err != nil {
			dbg.E(TAG, "Error in DeleteDriver get RowsAffected : ", err)
		}
	}

	return
}

// GetEmptyDriver returns an empty driver-object
func GetEmptyDriver() (driver *Driver, err error) {
	driver = &Driver{}
	return
}
