package driverManager

import (
	"github.com/OpenDriversLog/goodl-lib/jsonapi/addressManager"
	S "github.com/OpenDriversLog/goodl-lib/models/SQLite"
)

type Driver struct {
	Id         S.NInt64
	Priority   S.NInt64
	Address    addressManager.Address `json:",omitempty"`
	Name       S.NString
	Additional S.NString
}
