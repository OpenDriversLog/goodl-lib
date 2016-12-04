package carManager

import (
	driverManager "github.com/OpenDriversLog/goodl-lib/jsonapi/driverManager"
	S "github.com/OpenDriversLog/goodl-lib/models/SQLite"
)

type Car struct {
	Id           S.NInt64
	Type         S.NString
	Owner        driverManager.Driver
	Plate        S.NString
	FirstMileage S.NInt64
	Mileage      S.NInt64
	FirstUseDate S.NInt64
}