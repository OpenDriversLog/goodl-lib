package syncMan

import (
	"encoding/xml"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/addressManager"
	tripMan "github.com/OpenDriversLog/goodl-lib/jsonapi/tripMan/models"
	"github.com/OpenDriversLog/goodl-lib/models"
	S "github.com/OpenDriversLog/goodl-lib/models/SQLite"
	"time"
)

type JSONRefreshSyncManAnswer struct {
	models.JSONAnswer
	UpdatedContacts []*addressManager.Contact
	UpdatedTrips    []*tripMan.Trip
}

type JSONSyncManAnswer struct {
	models.JSONAnswer
	Syncs []*Sync
}

type JSONUpdateSyncManAnswer struct {
	models.JSONUpdateAnswer
}

type JSONTokenSyncManAnswer struct {
	models.JSONAnswer
	Id   int64
	Type string
	Name string
}

type JSONInsertSyncManAnswer struct {
	models.JSONInsertAnswer
}

/* type GoogleRefreshTokenJSONAnswer describes the answer returned by google oAuth when requesting a refresh token */
type GoogleRefreshTokenJSONAnswer struct {
	Access_token  string `json:"access_token"`
	Token_type    string `json:"token_type"`
	Expires_in    int    `json:"expires_in"`
	Refresh_token string `json:"refresh_token"`
	Id_token      string `json:"id_token"`
}

type Sync struct {
	Id              int64
	Name            string
	Type            string
	Priority        int64
	LastUpdate      int64
	Created         int64
	UpdateFrequency int64
	NextUpdate      int64
	CardDavConfig   *CardDavConfig
	CalDavConfig    *CalDavConfig
	OAuth           *OAuth
	HttpBasicAuth   *HttpBasicAuth
	HttpDigestAuth  *HttpDigestAuth
}

// SyncCollection might be useful for later CardDav-/CalDav-Stuff
type SyncCollection struct {
	XMLName   xml.Name `xml:"D:sync-collection"`
	D         string   `xml:"xmlns:D,attr"`
	SyncLevel int      `xml:"D:sync-level"`
	SyncToken string   `xml:"D:sync-token"`
	Prop      string   `xml:"D:prop"`
}

// PropFind might be useful for later CardDav-/CalDav-Stuff
type PropFind struct {
	XMLName   xml.Name `xml:"D:propfind"`
	D         string   `xml:"xmlns:D,attr"`
	SyncLevel int      `xml:"D:sync-level"`
	Prop      string   `xml:"D:prop"`
}

type GoogleFeed struct {
	XMLName      xml.Name `xml:"feed"`
	TotalResults int      `xml:"totalResults"`
	StartIndex   int      `xml:"startIndex"`
	ItemsPerPage int      `xml:"itemsPerPage"`
}

type GoogleContactsFeed struct {
	XMLName       xml.Name             `xml:"feed"`
	TotalResults  int                  `xml:"totalResults"`
	StartIndex    int                  `xml:"startIndex"`
	ItemsPerPage  int                  `xml:"itemsPerPage"`
	Entries       []GoogleContactsItem `xml:"entry"`
	UpdatedString string               `xml:"updated"` //time.RFC3339
}

type GoogleGroupsFeed struct {
	XMLName       xml.Name           `xml:"feed"`
	TotalResults  int                `xml:"totalResults"`
	StartIndex    int                `xml:"startIndex"`
	ItemsPerPage  int                `xml:"itemsPerPage"`
	Entries       []GoogleGroupsItem `xml:"entry"`
	UpdatedString string             `xml:"updated"` //time.RFC3339
}

type GoogleContactsItem struct {
	XMLName   xml.Name                  `xml:"entry"`
	Edited    time.Time                 `xml:"edited"`
	Title     string                    `xml:"title"`
	Addresses []StructuredPostalAddress `xml:"structuredPostalAddress"`
	Id        string                    `xml:"id"`
}

type GoogleAddress struct {
	Id               S.NInt64
	Sync             *Sync
	Contact          *addressManager.Contact
	FormattedAddress S.NString
	Rel              S.NString
	TripType         S.NInt64
	RetryTime        S.NInt64
	TryCount         S.NInt64
}

type StructuredPostalAddress struct {
	FormattedAddress string `xml:"formattedAddress"`
	Rel              string `xml:"rel,attr"`
}

type GroupMemberShipInfo struct {
	XMLName xml.Name `xml:"groupMembershipInfo"`
	Href    string   `xml:"href,attr"`
	Deleted bool     `xml:"deleted,attr"`
}

type GoogleGroupsItem struct {
	XMLName xml.Name   `xml:"entry"`
	Edited  time.Time  `xml:"edited"`
	Title   string     `xml:"title"`
	Links   []LinkItem `xml:"link"`
	Id      string     `xml:"id"`
}

type LinkItem struct {
	XMLName xml.Name `xml:"link"`
	Rel     string   `xml:"rel,attr"`
	Href    string   `xml:"href,attr"`
	Type    string   `xml:"type,attr"`
}

type GoogleContactsName struct {
}

type GoogleContact struct {
	Id         S.NInt64
	Sync       *Sync
	Groups     []*GoogleGroup
	Key        S.NString
	LastUpdate S.NInt64
	Name       S.NString
	Addresses  []*GoogleAddress
}

type GoogleGroup struct {
	Id         S.NInt64
	Sync       *Sync
	Key        S.NString
	Name       S.NString
	TripType   S.NInt64
	LastUpdate S.NInt64
}

type OAuth struct {
	Id             S.NInt64
	RefreshToken   S.NString `json:"-"`
	AccessToken    S.NString `json:"-"`
	ExpirationTime S.NInt64
}

type CardDavConfig struct {
	Id              S.NInt64
	Type            S.NString
	RootUri         S.NString
	AddressBookName S.NString
	PrincipalName   S.NString
	LastSyncKey     S.NString
}

type CalDavConfig struct {
	Id            S.NInt64
	Type          S.NString
	RootUri       S.NString
	CalendarName  S.NString
	PrincipalName S.NString
	SyncToken     S.NString
}

type HttpBasicAuth struct {
	Id       S.NInt64
	Usr      S.NString
	Password S.NString `json:"-"`
}

type HttpDigestAuth struct {
	Id       S.NInt64
	Usr      S.NString
	Password S.NString `json:"-"`
}
