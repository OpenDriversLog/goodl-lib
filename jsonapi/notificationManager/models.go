package notificationManager

import (
	tripMan "github.com/OpenDriversLog/goodl-lib/jsonapi/tripMan/models"
)

type Notification struct {
	Id         int64
	ExpirationTime  int64
	WasSent int
	WasUiDisabled int
	Type int
	TripId int64
	Trip *tripMan.Trip
	Subject string
	Message string
	ShortMessage string
}
