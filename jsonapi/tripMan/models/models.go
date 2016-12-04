package models

import (
	"database/sql"

	"github.com/OpenDriversLog/goodl-lib/jsonapi/addressManager"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/driverManager"
	"github.com/OpenDriversLog/goodl-lib/models"
	. "github.com/OpenDriversLog/goodl-lib/models/SQLite"
	geo "github.com/OpenDriversLog/goodl-lib/models/geo"
)

type TripSQL struct {
	TripId      sql.NullInt64
	Type        sql.NullInt64
	Title       sql.NullString
	Description sql.NullString
	DriverId    sql.NullInt64
	ContactId   sql.NullInt64
}

// for contacts transmitted over "the wire", we only transmit the IDs as the ContactManager has the Contacts already.
// for contacts used on the server (e.g. while creating pdf or inside the Android-App) we can directly fill the contact-objects
type Trip struct {
	Id                      int64
	TrackIds                NString
	TrackIdInts             []int64
	Tracks                  []string `json:",omitempty"`
	Type                    int
	TrackDetails            []*geo.Track
	Title                   NString
	Description             NString
	DeviceId                NInt64
	Driver                  *driverManager.Driver `json:",omitempty"`
	DriverId                NInt64
	Contact                 *addressManager.Contact `json:",omitempty"`
	ContactId               NInt64                  `json:",omitempty"`
	StartTime               NInt64
	EndTime                 NInt64
	StartAddress            *addressManager.Address
	EndAddress              *addressManager.Address
	StartContact            *addressManager.Contact   `json:",omitempty"`
	EndContact              *addressManager.Contact   `json:",omitempty"`
	ProposedStartContacts   []*addressManager.Contact `json:",omitempty"`
	ProposedEndContacts     []*addressManager.Contact `json:",omitempty"`
	StartContactId          NInt64                    `json:",omitempty"`
	EndContactId            NInt64                    `json:",omitempty"`
	ProposedStartContactIds NString                   `json:",omitempty"`
	ProposedEndContactIds   NString                   `json:",omitempty"`
	Distance                NFloat64
	IsReturnTrip            NInt64
	StartKeyPointId         NInt64
	EndKeyPointId           NInt64
	StartKeyPoint           *KeyPoint_Slim `json:",omitempty"`
	EndKeyPoint             *KeyPoint_Slim `json:",omitempty"`
	Reviewed                int64
	EditableTime		int64
	TimeOverDue int64
	History 		[]*CleanTripHistoryEntry	`json:",omitempty"`
}

type Change struct {
	OldVal interface{}
	NewVal interface{}

}
type TripHistoryEntry struct {
	Id                NInt64
	ChangeDate        NString
	TypeOld           NInt64
	TypeNew           NInt64
	TitleOld          NString
	TitleNew          NString
	DescOld           NString
	DescNew           NString
	DriverIdOld       NInt64
	DriverIdNew       NInt64
	ContactIdOld      NInt64
	ContactIdNew      NInt64
	StartContactIdOld NInt64
	StartContactIdNew NInt64
	EndContactIdOld   NInt64
	EndContactIdNew   NInt64
	IsReturnTripOld   NInt64
	IsReturnTripNew   NInt64
	IsReviewedOld     NInt64
	IsReviewedNew     NInt64
}

type CleanTripHistoryEntry struct {
	Id int64
	ChangeDate NString
	Changes map[string]Change
}

type KeyPoint_Slim struct {
	KeyPointId      int64
	Latitude        float64
	Longitude       float64
	StartTime       int64
	EndTime         int64
	PreviousTrackId NInt64
	NextTrackId     NInt64
}

type JSONTripManAnswer struct {
	models.JSONAnswer
	Trips []*Trip
}

type JSONInsertTripAnswer struct {
	models.JSONInsertAnswer
	UpdatedNotifications bool
	UpdatedTrips []*Trip
	RemovedTrips []*Trip
}

type JSONCheckTripAnswer struct {
	models.JSONAnswer
	CheckedTripIds []interface{}
}

type JSONUpdateTripAnswer struct {
	models.JSONUpdateAnswer
	UpdatedNotifications bool
	UpdatedTrips []*Trip
	RemovedTrips []*Trip
	Changes []*CleanTripHistoryEntry
}
