package deviceManager

import (
	S "github.com/OpenDriversLog/goodl-lib/models/SQLite"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/colorManager"
	"github.com/OpenDriversLog/goodl-lib/models"
)

type Device struct {
	Id          S.NInt64
	Description S.NString
	Color		*colorManager.Color
	Checked     S.NInt64
	CarId 		S.NInt64
	Guid	    S.NString
}

type JSONDeleteDeviceAnswer struct {
	models.JSONDeleteAnswer
	Guid string
}