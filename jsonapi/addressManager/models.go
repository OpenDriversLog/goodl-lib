package addressManager

import (
	"github.com/OpenDriversLog/goodl-lib/models"
	S "github.com/OpenDriversLog/goodl-lib/models/SQLite"
)

type Address struct {
	Id          S.NInt64
	Latitude    S.NFloat64
	Longitude   S.NFloat64
	Street      S.NString
	Postal      S.NString
	City        S.NString
	Additional1 S.NString
	Additional2 S.NString
	HouseNumber S.NString
	Title       S.NString
	Fuel        S.NString
	GeoCoder S.NString
	GeoZones    []*GeoFenceRegion
}
type Contact struct {
	Id          S.NInt64
	Title       S.NString
	Description S.NString
	Additional  S.NString
	TripType    S.NInt64
	Type        S.NInt64
	Disabled    S.NInt64
	Address     *Address
	SyncedWith  S.NString
}
type PointOfInterest struct {
	Id                  int64
	Description         string
	TripType            int64
	PointOfInterestType S.NInt64
	Address             *Address
}

type GeoFenceRegion struct {
	Id          S.NInt64
	OuterMinLat S.NFloat64
	OuterMinLon S.NFloat64
	OuterMaxLat S.NFloat64
	OuterMaxLon S.NFloat64
	Color       S.NString
	Rectangle   *GeoRectangle
}

type GeoRectangle struct {
	Id          S.NInt64
	TopLeftLat  S.NFloat64
	TopLeftLon  S.NFloat64
	BotRightLat S.NFloat64
	BotRightLon S.NFloat64
}

type JSONAddressManAnswer struct {
	models.JSONAnswer
	Addresses []*Address
	Contacts  []*Contact
}

type JSONUpdateAddressManAnswer struct {
	models.JSONUpdateAnswer
	GeoZones                     []*GeoFenceRegion
	MatchingTripsForEndContact   string
	MatchingTripsForStartContact string
	NewContact                   *Contact
	UpdatedDto		     *Contact
}

type JSONInsertAddressManAnswer struct {
	models.JSONInsertAnswer
	GeoZones                     []*GeoFenceRegion
	MatchingTripsForEndContact   string
	MatchingTripsForStartContact string
}
