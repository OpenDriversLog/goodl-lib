// Package addressManager is responsible for CRUD addresses and calculate & CRUD GeoZones & contacts.
// Also finds coordinates from addresses and addresses from coordinates using different GeoCoding services.
package addressManager

import (
	"database/sql"
	"errors"

	"github.com/Compufreak345/dbg"
	geo "github.com/kellydunn/golang-geo"

	"fmt"
	S "github.com/OpenDriversLog/goodl-lib/models/SQLite"
	. "github.com/OpenDriversLog/goodl-lib/tools"
	"math"
	"net/http"
	"time"

	"encoding/json"
	"io/ioutil"
	gc "github.com/OpenDriversLog/goodl-lib/models/odl-geocode"
	"net/url"
)

const TAG = "goodl-lib/jsonApi/addressManager.go"

// use these for a correct address query!
const AddressQueryFields = "_addressId,street,postal,city,additional1,additional2,HouseNumber,Addresses.title,latitude,longitude,fuel,geoCoder"
const NoOfAddressQueryFields = 12

// use these for a correct contact query!
const ContactQueryFields = "_contactId,type,Contacts.title,description,additional,addressId,tripTypeId,disabled,syncedWith"
const NoOfContactQueryFields = 9

// use these for a correct POI query!
const POIQueryFields = "_pointOfInterestId,description,type,pointOfInterestType,tripTypeId"
const NoOfPOIQueryFields = 5

const GeoFenceRegionQueryFields = `_geoFenceRegionId,outerMinLat,outerMinLon,outerMaxLat,outerMaxLon,color,rectangleId,
topLeftLat,topLeftLon,botRightLat,botRightLon`
const NoOfGeoFenceRegionQueryFields = 11
const NoOfNoGeometryGeoFenceRegionQueryFields = 6

type DecartaReverseResp struct {
	Addresses []map[string]map[string]string `json:"addresses"`
}
type DecartaForwardResp struct {
	Results []DecartaForwardResultItem `json:"results"`
}

type DecartaForwardResultItem struct {
	Position DecartaPosition   `json:"position"`
	Type     string            `json:"type"`
	Address  map[string]string `json:"address"`
}

type DecartaPosition struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lon"`
}

// returns address selected by Id (without GeoZones)
func GetAddress(addressId int64, dbCon *sql.DB) (address *Address, err error) {
	address = &Address{}
	q := "SELECT " + AddressQueryFields + " FROM Addresses WHERE _addressId=?"
	err = dbCon.QueryRow(q, addressId).Scan(&address.Id, &address.Street, &address.Postal, &address.City,
		&address.Additional1, &address.Additional2, &address.HouseNumber, &address.Title, &address.Latitude, &address.Longitude, &address.Fuel,&address.GeoCoder)
	if err != nil {
		dbg.E(TAG, "unable to get Address by Id %d", addressId)
		return
	}

	return
}

// func GetAddresseWithGeoFences gets addresses, including GeoFence-Regions
func GetAddressesWithGeoFences(where string, dbCon *sql.DB, params ...interface{}) (addresses []*Address, err error) {
	q := "SELECT " + AddressQueryFields + "," + GeoFenceRegionQueryFields + `,hasGeoFenceRegion FROM AddressesWithGeoZones AS Addresses`
	if where != "" {
		q += "WHERE " + where
	}
	q += " ORDER BY _addressId"
	rows, err := dbCon.Query(q, params...)
	if err != nil {
		dbg.E(TAG, "Error in 1st dbCon.Query(q, params...) for GetAddressesWithGeoFences: %v ", err)
		return
	}
	addresses = make([]*Address, 0)
	var prevAdd *Address = nil
	for rows.Next() {
		var add *Address

		add, err = GetAddressWithGeoZonesFromRes(rows, 0, 0)
		if err != nil {
			dbg.E(TAG, "Error in 1stGetAddressWithGeoZonesFromRes for GetAddressesWithGeoFences: %v ", err)
			return
		}
		if prevAdd != nil && prevAdd.Id == add.Id {
			prevAdd.GeoZones = append(prevAdd.GeoZones, add.GeoZones...)

		} else {
			addresses = append(addresses, add)
		}
		prevAdd = add

	}

	return
}


func GetAddressWithGeoZonesFromRes(row Scannable, skipBefore int, skipAfter int) (add *Address, err error) {
	add, err = GetAddressFromRes(row, skipBefore, NoOfGeoFenceRegionQueryFields+skipAfter+1)
	if err != nil {
		dbg.E(TAG, "Error in GetAddressFromRes for GetAddressWithGeoZonesFromRes: %v ", err)
		return
	}
	hasGz := -1
	err = SkippyScanRow(row, skipBefore+NoOfAddressQueryFields+NoOfGeoFenceRegionQueryFields, skipAfter, &hasGz)
	if err != nil {

		dbg.E(TAG, "Error in 1st SkippyScanRow for GetAddressWithGeoZonesFromRes: %v ", err)
		return
	}
	if hasGz != 0 {
		var gz *GeoFenceRegion
		gz, err = GetGeoZoneFromRes(row, skipBefore+NoOfAddressQueryFields, 1)
		if err != nil {
			dbg.E(TAG, "Error in 1st GetGeoZoneFromRes for GetAddressWithGeoZonesFromRes: %v ", err)
			return
		}

		add.GeoZones = []*GeoFenceRegion{}
		if gz != nil {
			add.GeoZones = append(add.GeoZones, gz)
		}
	}
	return
}

// returns Contact selected by Id (without GeoZones)
func GetContact(contactId int64, dbCon *sql.DB, getAddr bool) (contact *Contact, err error) {
	contact = &Contact{}
	q := "SELECT " + ContactQueryFields + ` FROM Contacts WHERE _contactId=?`
	// const ContactQueryFields = "_contactId,type,title,addressId,tripTypeId"
	var adrId sql.NullInt64
	err = dbCon.QueryRow(q, contactId).Scan(&contact.Id, &contact.Type, &contact.Title, &contact.Description, &contact.Additional, &adrId, &contact.TripType, &contact.Disabled, &contact.SyncedWith)

	if err != nil {
		if err == sql.ErrNoRows {
			dbg.I(TAG, "Contact with id %d not found", contactId)
		} else {
			dbg.E(TAG, "unable to get Contact by Id %d : %v", contactId, err)
		}
		return
	}
	if !adrId.Valid {
		dbg.W(TAG, "Contact with id %v has no address", contact.Id)
	} else {
		if getAddr {
			dbg.W(TAG, "Get contacts address  not optimized")
			contact.Address, err = GetAddress(adrId.Int64, dbCon)
			if err != nil {
				dbg.E(TAG, "Error getting address with id %d!", adrId.Int64, err)
				return
			}
		} else {
			contact.Address = &Address{Id: S.NInt64(adrId.Int64)}
		}

	}
	return
}

// GetEmptyContact returns an empty contact-object
func GetEmptyContact() (contact *Contact, err error) {
	contact = &Contact{}
	contact.Address = &Address{}
	return
}

// GetContactsWithGeoZones gets the contacts matching the given where-string, together with their GeoZones
func GetContactsWithGeoZones(where string, dbCon *sql.DB, params ...interface{}) (contacts []*Contact, err error) {
	q := "SELECT " + ContactQueryFields + "," + AddressQueryFields + "," + GeoFenceRegionQueryFields +
		",hasGeoFenceRegion FROM Contacts LEFT JOIN AddressesWithGeoZones AS Addresses ON addressId = _addressId"
	if where != "" {
		q += " WHERE " + where
	}
	q += " ORDER BY disabled ASC"
	rows, err := dbCon.Query(q, params...)
	if err != nil {
		dbg.E(TAG, "Error in 1st dbCon.Query %v for GetContactsWithGeoZones: %v ", q, err)

		return
	}
	contacts = make([]*Contact, 0)
	var prevContact *Contact
	for rows.Next() {
		var contact *Contact
		contact, err = GetContactWithGeoZonesFromRes(rows, 0, 0, true)
		if err != nil {
			dbg.E(TAG, "Error in 1st GetContactWithGeoZonesFromRes for GetContactsWithGeoZones: %v ", err)

			return
		}
		if prevContact != nil && prevContact.Id == contact.Id {
			prevContact.Address.GeoZones = append(prevContact.Address.GeoZones, contact.Address.GeoZones...)
		} else {
			contacts = append(contacts, contact)
		}

	}

	return
}

// GetContactWithGeoZone gets a contact with its GeoZones by its ID.
func GetContactWithGeoZone(id int64, dbCon *sql.DB) (contact *Contact, err error) {
	v, err := GetContactsWithGeoZones("_contactId=?", dbCon, id)
	if err != nil {
		dbg.E(TAG, "Error in 1st GetContactsWithGeoZones for GetContactWithGeoZone: %v ", err)

		return
	}
	if len(v) == 0 {
		err = sql.ErrNoRows
		return
	}
	contact = v[0]
	return
}

// GetPointsOfInterest gets points of interest (creation not implemented yet)
func GetPointsOfInterest(where string, dbCon *sql.DB, params ...interface{}) (pois []*PointOfInterest, err error) {
	q := "SELECT " + POIQueryFields + "," + AddressQueryFields + " FROM POIs"
	if where != "" {
		q += "WHERE " + where
	}
	rows, err := dbCon.Query(q, params...)
	if err != nil {
		dbg.E(TAG, "Error in 1st dbCon.Query(q, params...) for GetPointsOfInterest: %v ", err)
		return
	}
	pois = make([]*PointOfInterest, 0)
	for rows.Next() {
		var poi *PointOfInterest
		poi, err = GetPOIFromRes(rows, 0, 0, true)
		if err != nil {
			dbg.E(TAG, "Error in 1stGetPOIFromRes for GetPointsOfInterest: %v ", err)
			return
		}
		pois = append(pois, poi)

	}

	return
}

var GeoZoneSize = 0.1

// CreateGeoZoneAddress creates a new Address and puts a default GeoZone of 50 meters in each direction around it if no geoZone is given
func CreateGeoZoneAddress(address *Address, dbCon *sql.DB) (key int64, err error) {
	insFields := "street,postal,city,additional1,additional2,HouseNumber,title,fuel,geoCoder"
	valString := "?,?,?,?,?,?,?,?,?"
	var gzKey int64 = 0
	vals := []interface{}{address.Street, address.Postal, address.City, address.Additional1, address.Additional2, address.HouseNumber, address.Title, address.Fuel,address.GeoCoder}
	if address.Latitude == 0 && address.Longitude == 0 {

	}
	if address.Latitude != 0 || address.Longitude != 0 {
		insFields += ",latitude,longitude"
		vals = append(vals, address.Latitude, address.Longitude)
		valString += ",?,?"

		if address.GeoZones == nil || len(address.GeoZones) == 0 { // Automatic create GeoZone if not given
			var gz *GeoFenceRegion
			gzKey, gz, err = CreateGeoZoneFromCoords(float64(address.Latitude), float64(address.Longitude), GeoZoneSize, dbCon)
			if err != nil {
				dbg.E(TAG, "Error in 1st CreateGeoZoneFromCoords for CreateGeoZoneAddress: %v ", err)

				return
			}
			address.GeoZones = make([]*GeoFenceRegion, 1)
			address.GeoZones[0] = gz
		} else { // Update or create GeoZone if given
			for _, geoZone := range address.GeoZones {
				if geoZone.Id != 0 { // Update = Delete and insert new ;)
					DeleteGeoZoneWithRectangle(geoZone, dbCon)
				}
				geoZone.Rectangle.Id = -1
				gzKey, err = CreateGeoZone(
					geoZone, dbCon)
				if err == nil {
					geoZone.Id = S.NInt64(gzKey)
				} else {
					dbg.E(TAG, "Error in 1st CreateGeoZone() for CreateGeoZoneAddress: %v ", err)

				}

			}
		}

	}
	q := "INSERT INTO Addresses(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in 1st dbCon.Exec for CreateGeoZoneAddress: %v ", err)
		return
	}
	key, err = res.LastInsertId()
	if err != nil {
		dbg.E(TAG, "Error in 1st res.LastInsertId() for CreateGeoZoneAddress: %v ", err)
		return
	}
	address.Id = S.NInt64(key)
	if gzKey != 0 {
		_, err = dbCon.Exec("Insert INTO Address_GeoFenceRegion (addressId,geoFenceRegionId) VALUES (?,?)", key, gzKey)
		if err != nil {
			dbg.E(TAG, "Error in 2nd dbCon.Exec for CreateGeoZoneAddress: %v ", err)
			return
		}
	}

	return
}

// func UpdateAddress updates an Address - latitude and longitude can not be updated and update is only allowed
// when retryTime is not 0 (=geocoded address was not found yet)
func UpdateAddress(address *Address, dbCon *sql.DB) (err error) {

	var retryTime int64
	err = dbCon.QueryRow("SELECT retrytime FROM Addresses WHERE _addressId=?", address.Id).Scan(&retryTime)
	if err != nil {
		dbg.E(TAG, "Error updating address : getting retrytime failed : ", err)
		return
	}
	if retryTime == 0 {
		dbg.WTF(TAG, "Somebody tried to update an address (%d) with retryTime of 0 - this should not be possible",address.Id)
		return errors.New("Not allowed")
	}
	_, err = dbCon.Exec("UPDATE Addresses SET street=?, postal=?, city=?,houseNumber=?,geoCoder=?,retryTime=0 WHERE _addressId=?", address.Street, address.Postal, address.City, address.HouseNumber,address.GeoCoder, address.Id)

	return
}

const DefaultGeoZoneColor = "#FFCC00"

// FillUnknownAddress fills an unknown address object.
func FillUnknownAddress(add *Address) {
	add.Street = "Unbekannt"
	add.Postal = ""
	add.City = "Unbekannt"
	add.HouseNumber = ""
	add.GeoCoder=""
	return
}
var uIdToRetrying map[int64]bool

// RetryAddresses looks for incomplete addresses in the database, where "retrytime" has expired
// and tries to geocode them again.
func RetryAddresses(dbCon *sql.DB, uId int64) (adrCount int, err error) {

	if uIdToRetrying == nil {
		uIdToRetrying = make(map[int64]bool)
	}
	if uIdToRetrying[uId] {
		return
	}
	uIdToRetrying[uId] = true
	client := &http.Client{
		Timeout: time.Duration(10 * time.Second),
	}
	adrs, err := GetAddressesByWhere("retrytime!=0 AND retrytime<=?", dbCon, time.Now().Unix())
	if err != nil {
		dbg.E(TAG, "Error getting addresses open for retry", err)
		uIdToRetrying[uId] = false
		return
	}

	for _, addr := range adrs {
		err = FillAddressForLatLng(addr, float64(addr.Latitude), float64(addr.Longitude), client,uId, dbCon)
		if err != nil {
			var tryCount int64
			errTc := dbCon.QueryRow("SELECT trycount FROM Addresses WHERE _addressId=?", addr.Id).Scan(&tryCount)
			if errTc != nil {
				dbg.E(TAG, "Error getting trycount : ", err)
			}
			if err == ErrEmptyResult {
				dbg.W(TAG, "No geocoding result returned for coordinates ", addr.Latitude, addr.Longitude)
			}
			if err == ErrNeedFixBeforeRetry {
				dbg.WTF(TAG, "Issue initialising geoCoding! Please check!", addr.Latitude, addr.Longitude)
			}
			retryTime := CalcRetryTime(tryCount, err)
			tryCount++
			_, err = dbCon.Exec("Update ADDRESSES SET retrytime=?,trycount=? WHERE _addressId=?", retryTime, tryCount, addr.Id)
			if err != nil {
				dbg.E(TAG, "Error updating address retrytime : ", err)
			}
		} else {
			adrCount++
			err = UpdateAddress(addr, dbCon)
			if err != nil {
				dbg.E(TAG, "Error updating address : ", err)
			}
		}

	}
	uIdToRetrying[uId] = false
	return
}

var ErrEmptyResult = errors.New("Geocoder returned 0 results")
var ErrNeedFixBeforeRetry = errors.New("Need fix before retrying!")

// FillAddressForLatLng fills an address - object with the address found at the given latitude & longitude, first by looking
// if there already is a similar address in the users database, followed by asking the Geocoder if nothing was found.
// If client is nil, it will be initialised automatically.
func FillAddressForLatLng(addr *Address, lat float64, lng float64, client *http.Client,uId int64, dbCon *sql.DB) (err error) {
	k := float64(10) / 111111
	minLat := lat - k
	maxLat := lat + k
	minLng := lng - math.Cos(lat)/111111
	maxLng := lng + math.Cos(lat)/111111

	addr.Latitude = S.NFloat64(lat)
	addr.Longitude = S.NFloat64(lng)

	var addrId int64

	//dbg.WTF(TAG,"Found minLat %f maxLat %f minLng %f maxLng %f for lat %lat and lng %lng ")
	err = dbCon.QueryRow(`SELECT _addressId FROM Addresses WHERE HouseNumber!="" AND latitude>? AND longitude>? AND latitude<? AND longitude<?`, minLat, minLng, maxLat, maxLng).Scan(&addrId)
	// returns models.Address
	if err != nil {
		if err != sql.ErrNoRows {
			FillUnknownAddress(addr)
			dbg.E(TAG, "Error querying for existing addressId : ", err)
			return ErrNeedFixBeforeRetry
		} else {

			dbg.I(TAG, "No corresponding address found - ask geocoder!")
			err = FillAddrFromGeocoder(addr,lat,lng,client, uId)
			addr.Latitude = S.NFloat64(lat)
			addr.Longitude = S.NFloat64(lng)
			if err != nil {
				dbg.W(TAG, "Error filling address : ", err)

			}

		}
	} else { // We found an matching address in the database
		dbg.V(TAG, "Matching address (+/-1 10m) with id %d found in DB!", addrId)
		var ad *Address
		ad, err = GetAddressById(addrId, dbCon)
		if err != nil {
			dbg.E(TAG, "Error getting known address", err)
			FillUnknownAddress(addr)
			return
		}
		addr.Street = ad.Street
		addr.HouseNumber = ad.HouseNumber
		addr.City = ad.City
		addr.Postal = ad.Postal
		addr.Fuel = ad.Fuel
		addr.GeoCoder = ad.GeoCoder

		return
	}
	return
}

// TODO : Set to your odl-geocoder-Server  https://github.com/OpenDriversLog/odl-geocoder
var GeoChainAddr = "http://127.0.0.1:6091"

// FillAddrFromGeocoder fills an address - object with the address found at the given latitude & longitude by the Geocoder.
// Please use FillAddressForLatLng to allow for address buffering.
// If client is nil, it will be initialised automatically.
func FillAddrFromGeocoder(addr *Address,lat float64, lng float64, client *http.Client,uId int64) (err error){
	var req *http.Request

	req, err = http.NewRequest("GET", fmt.Sprintf(GeoChainAddr+"/reverse/%d/b/abc/%f/%f",uId, lat, lng), nil)
	if err != nil {
		dbg.E(TAG, "Error initializing httpRequest : ", err)
		FillUnknownAddress(addr)
		return
	}
	if client == nil {
		client = &http.Client{
			Timeout: time.Duration(10 * time.Second),
		}
	}
	/* Get Details */
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		dbg.E(TAG, "Error executing reverse geocoda request: %s", err)
		FillUnknownAddress(addr)
		return
	}
	var _body []byte
	_body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		if err != nil {
			dbg.E(TAG, "Error reading reverse geocoda response: %s", err)
			FillUnknownAddress(addr)
			return ErrNeedFixBeforeRetry
		}
	}
	err = FillAddrFromGeocoderResp(string(_body), addr, false)

	return
}

// FillAddrFromGeocoderResp fills an address-object using the given GeoCoder-response.
func FillAddrFromGeocoderResp(resp string, addr *Address, addLatLng bool) (err error) {
	if addr == nil {
		dbg.E(TAG, "Error : Got nil address to fil")
		return errors.New("Address object is nil.")
	}
	var adr gc.Address
	res := gc.GeoResp{}
	err = json.Unmarshal([]byte(resp), &res)
	if err != nil {
		dbg.E(TAG,"Error unmarshaling geocoder resp : ", err)
		if dbg.Develop {
			dbg.WTF(TAG,"Response : ", string(resp))
		}
	}
	adr = res.Address
	if adr.City!="" {

		addr.HouseNumber = S.NString(adr.HouseNumber)
		addr.City = S.NString(adr.City)
		addr.Street = S.NString(adr.Street)
		addr.Postal = S.NString(adr.Postal)
		addr.Latitude = S.NFloat64(adr.Lat)
		addr.Longitude = S.NFloat64(adr.Lng)
		addr.Fuel = S.NString(adr.Fuel)
		addr.Title = S.NString(adr.Title)
		addr.GeoCoder = S.NString(res.Provider)
		return
	} else {
		if dbg.Develop {
			dbg.I(TAG,"Response for not found adress: ", string(resp))
			dbg.I(TAG,"Parsed response for not found adress: %+v", res)

		}
		FillUnknownAddress(addr)
		return ErrEmptyResult
	}
}

const OneDay int64 = int64(60 * 60 * 24)

// GetAddressIdForLatLng finds a given address by its lat-lng, creates it in the database and returns its primary key.
// give nil client if it should be initialized automatically. But that way it needs to auth himself for each request.
func GetAddressIdForLatLng(lat float64, lng float64, client *http.Client, uId int64, dbCon *sql.DB) (addrId int64, err error) {

	addrId = -1
	var addr Address
	err = FillAddressForLatLng(&addr, lat, lng, client,uId, dbCon)
	var retryTime int64
	if err != nil {
		if err == ErrEmptyResult { // Our GeoCoder did not find anything at the given address - retry tomorrow.
			dbg.W(TAG, "No geocoding result returned for coordinates ", lat, lng)
			retryTime = int64(time.Now().Unix() + OneDay)
		} else if err == ErrNeedFixBeforeRetry { // We probably got a bug - retry next day
			dbg.WTF(TAG, "Issue initialising geoCoding! Please check!", lat, lng)
			retryTime = int64(time.Now().Unix() + OneDay)
		} else { // GeoCoding-service was probably not reachable - retry in 30 seconds.
			dbg.E(TAG, "Error getting Address from location : ", err)
			retryTime = int64(time.Now().Unix() + 30)
		}
	}

	ires, errPTID := dbCon.Exec("INSERT INTO Addresses(street, postal, city,houseNumber,latitude,longitude,GeoCoder,retryTime,tryCount) VALUES (?, ?, ?,?,?,?,?,?,1)",
		addr.Street, addr.Postal, addr.City, addr.HouseNumber, addr.Latitude, addr.Longitude,addr.GeoCoder, retryTime)
	if errPTID != nil {
		dbg.E(TAG, "Failed to insert newAddress..", errPTID)
		return -1, errPTID
	}
	addrId, err = ires.LastInsertId()
	if err != nil {
		dbg.E(TAG, "Failed to get Id of just inserted Address...", err)
		return -1, err
	}

	return
}

// CreateGeoZoneFromCoords creates a new Geozone in the given distance (size) (in km) around the coord center point.
// e.g. distance of 50 meters diagonally in each direction
func CreateGeoZoneFromCoords(latitude float64, longitude float64, size float64, dbCon *sql.DB) (key int64, geoZone *GeoFenceRegion, err error) {
	p := geo.NewPoint(latitude, longitude)
	pTopLeft := p.PointAtDistanceAndBearing(size, 225)
	pBotRight := p.PointAtDistanceAndBearing(size, 45)

	geoZone = &GeoFenceRegion{
		OuterMinLat: S.NFloat64(pTopLeft.Lat()),
		OuterMaxLat: S.NFloat64(pBotRight.Lat()),
		OuterMinLon: S.NFloat64(pTopLeft.Lng()),
		OuterMaxLon: S.NFloat64(pBotRight.Lng()),
		Color:       DefaultGeoZoneColor,
		Rectangle: &GeoRectangle{
			TopLeftLat:  S.NFloat64(pTopLeft.Lat()),
			TopLeftLon:  S.NFloat64(pTopLeft.Lng()),
			BotRightLat: S.NFloat64(pBotRight.Lat()),
			BotRightLon: S.NFloat64(pBotRight.Lng()),
		},
	}
	key, err = CreateGeoZone(
		geoZone, dbCon)
	if err == nil {
		geoZone.Id = S.NInt64(key)
	} else {
		dbg.E(TAG, "Error in 1st CreateGeoZone() for CreateGeoZoneFromCoords: %v ", err)

	}
	return
}

// DeleteGeoZoneWithRectangle deletes the given GeoZone and the according GeoRectangle.
func DeleteGeoZoneWithRectangle(geoZone *GeoFenceRegion, dbCon *sql.DB) (rowCount int64, err error) {
	var res sql.Result
	if geoZone.Rectangle != nil && geoZone.Rectangle.Id != 0 {
		_, err = DeleteRectangle(int64(geoZone.Rectangle.Id), dbCon)
		if err != nil {
			dbg.E(TAG, "Error deleting rectangle : ", err)
			return
		}
	}
	res, err = dbCon.Exec("DELETE FROM GeoFenceRegions WHERE _geoFenceRegionId=?", geoZone.Id)
	if err != nil {
		dbg.E(TAG, "Error in DeleteGeoZoneWithRectangle : ", err)
		return
	}

	rowCount, err = res.RowsAffected()
	if err != nil {
		dbg.E(TAG, "Error in DeleteGeoZoneWithRectangle get RowsAffected : ", err)
		return
	}

	_, err = dbCon.Exec("DELETE FROM Address_GeoFenceRegion where geoFenceRegionId=?", geoZone.Id)
	if err != nil {
		dbg.E(TAG, "Error in DeleteGeoZoneWithRectangle Address_GeoFenceRegion : ", err)
		return
	}
	_, err = dbCon.Exec("DELETE FROM KeyPoints_GeoFenceRegions where geoFenceRegionId=?", geoZone.Id)
	if err != nil {
		dbg.E(TAG, "Error in DeleteGeoZoneWithRectangle KeyPoints_GeoFenceRegions : ", err)
		return
	}
	return
}

// DeleteRectangle deletes a GeoRectangle.
func DeleteRectangle(id int64, dbCon *sql.DB) (rowCount int64, err error) {
	var res sql.Result
	res, err = dbCon.Exec("DELETE FROM Rectangles WHERE _rectangleId=?", id)
	if err != nil {
		dbg.E(TAG, "Error in DeleteRectangle : ", err)
	} else {
		rowCount, err = res.RowsAffected()
		if err != nil {
			dbg.E(TAG, "Error in DeleteRectangle get RowsAffected : ", err)
		}
	}

	return
}

// CreateGeoZone creates a new GeoZone, with a given GeoRectangle or a new one.
func CreateGeoZone(geoZone *GeoFenceRegion, dbCon *sql.DB) (key int64, err error) {

	var rectKey int64 = -1
	if geoZone.Rectangle != nil {
		rectKey = int64(geoZone.Rectangle.Id)
		if rectKey < 1 {
			rectKey, err = CreateGeoRectangle(geoZone.Rectangle, dbCon)
			if err != nil {
				dbg.E(TAG, "Error in 1st CreateGeoRectangle for CreateGeoZone: %v ", err)

				return
			}
		}

	}

	vals := []interface{}{geoZone.OuterMinLat, geoZone.OuterMinLon, geoZone.OuterMaxLat, geoZone.OuterMaxLon, geoZone.Color}
	valString := "?,?,?,?,?"

	insFields := "outerMinLat,outerMinLon,outerMaxLat,outerMaxLon,color"
	if rectKey != -1 {
		insFields += ",rectangleId"
		valString += ",?"
		vals = append(vals, rectKey)
		geoZone.Rectangle.Id = S.NInt64(rectKey)
	}

	q := "INSERT INTO GeoFenceRegions(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in 1st dbCon.Exec for CreateGeoZone: %v ", err)

		return
	}
	key, err = res.LastInsertId()
	if err != nil {
		dbg.E(TAG, "Error in 1st res.LastInsertId() for CreateGeoZone: %v ", err)
		return
	}

	return
}

// CreateGeoRectangle creates a new GeoRectangle.
func CreateGeoRectangle(geoRectangle *GeoRectangle, dbCon *sql.DB) (key int64, err error) {
	vals := []interface{}{geoRectangle.BotRightLat, geoRectangle.BotRightLon, geoRectangle.TopLeftLat, geoRectangle.TopLeftLon}
	valString := "?,?,?,?"

	insFields := "botRightLat,botRightLon,topLeftLat,topLeftLon"

	q := "INSERT INTO Rectangles(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in 1st dbCon.Exec for CreateGeoRectangle: %v ", err)
		return
	}
	key, err = res.LastInsertId()
	return
}

// CreateContact creates a new contact.
func CreateContact(contact *Contact, dbCon *sql.DB) (key int64, err error) {
	var addrKey int64 = -1
	if contact.Address != nil {
		if contact.Address.Id < 1 { // no address existent ------ create it, El Duderino!
			addrKey, err = CreateGeoZoneAddress(contact.Address, dbCon)
			if err != nil {
				dbg.E(TAG, "Error in 1st CreateGeoZoneAddress for CreateGeoRectangle: %v ", err)

				return
			}
		} else {
			addrKey = int64(contact.Address.Id)
		}
	}

	vals := []interface{}{contact.Title, contact.Description, contact.Additional}
	valString := "?,?,?"

	insFields := "title,description,additional"
	if addrKey != -1 {
		insFields += ",addressId"
		valString += ",?"
		vals = append(vals, addrKey)
	}
	if contact.TripType != 0 {
		insFields += ",tripTypeId"
		valString += ",?"
		vals = append(vals, contact.TripType)
	}
	if contact.Type != 0 {
		insFields += ",type"
		valString += ",?"
		vals = append(vals, contact.Type)
	}
	if contact.Disabled != 0 {
		insFields += ",disabled"
		valString += ",?"
		vals = append(vals, contact.Disabled)
	}
	if contact.SyncedWith != "" {
		insFields += ",syncedWith"
		valString += ",?"
		vals = append(vals, contact.SyncedWith)
	}
	q := "INSERT INTO Contacts(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in 2nd dbCon.Exec for CreateGeoRectangle: %v ", err)

		return
	}
	key, err = res.LastInsertId()
	contact.Id = S.NInt64(key)
	if contact.Address != nil && len(contact.Address.GeoZones) != 0 {
		for _, gz := range contact.Address.GeoZones {
			err = UpdateAllKeyPointsForGeoZones([]int64{int64(gz.Id)}, dbCon)
			if err != nil {
				dbg.E(TAG, "Error in UpdateAllKeyPointsForGeoZones for CreateGeoZone: %v ", err)
				return
			}
		}
	}
	return
}

// DeleteContact deletes the contact with the given ID
func DeleteContact(id int64, dbCon *sql.DB) (rowCount int64, err error) {
	var res sql.Result
	res, err = dbCon.Exec(`UPDATE TRIPS SET startContactId=null WHERE startContactId=?;
	UPDATE TRIPS SET endContactId=null WHERE endContactId=?;
	DELETE FROM Contacts WHERE _contactId=?;

	`, id, id, id)
	if err != nil {
		dbg.E(TAG, "Error in DeleteContact : ", err)
	} else {
		rowCount, err = res.RowsAffected()
		if err != nil {
			dbg.E(TAG, "Error in DeleteContact get RowsAffected : ", err)
		}
	}

	return
}

// UpdateContact updates contact, including GeoZone and address and updates the given contact with ids of new stuff
// (Address update / GeoZone update means new ID for these)
// oldUpdatedContact gets filled if we needed to clone the contact (because of data integrity)
// in this case c will point to a new contact and oldUpdatedContact will be the previos entry, just with Disabled=1
func UpdateContact(c *Contact, dbCon *sql.DB) (rowCount int64, oldUpdatedContact *Contact, err error) {

	vals := []interface{}{}
	firstVal := true
	valString := ""

	updateGz := false

	if c.Address != nil {
		prevContact, err := GetContact(int64(c.Id), dbCon, true)
		if err != nil {
			dbg.E(TAG, "Error getting contact before change : ", err)
			return 0,nil, err
		}
		if (prevContact.Address != nil && (prevContact.Address.Street != c.Address.Street ||
			prevContact.Address.Postal != c.Address.Postal ||
			prevContact.Address.HouseNumber != c.Address.HouseNumber ||
			prevContact.Address.City != c.Address.City)) || prevContact.Title != c.Title { // To prevent older entries from being messed with,
			prevId := c.Id
			// clone the contact.
			c.Id = -1
			c.Address.Id = -1
			var id int64
			id, err = CreateContact(c, dbCon)
			if err != nil {
				dbg.E(TAG, "Error cloning contact: ", err)
				return 0,nil, err
			}
			cs := make([]*Contact, 0)
			cs, err = GetContactsWithGeoZones("_contactId=?", dbCon, id)
			if err != nil {
				dbg.E(TAG, "Error getting cloned contact: ", err)
				return 0,nil, err
			}
			if len(cs) != 1 {
				dbg.E(TAG, "Error getting cloned contact - did not find exactly one contact with cloned id %d : Found %d", id, len(cs))
			}
			c = cs[0]

			_, err = dbCon.Exec("UPDATE Contacts SET disabled=1 WHERE _contactId=?", prevId)
			if err != nil {
				dbg.E(TAG, "Error disabling previous contact with id %d : ", prevId, err)
				return 0,nil, err
			}
			prevContact.Disabled = 1
			return 1, prevContact, err
		}
		updateGz = true
		var addId int64 = 0
		// WE ALWAYS CREATE A NEW ADDRESS!
		addId, err = CreateGeoZoneAddress(c.Address, dbCon)
		if err != nil {
			dbg.E(TAG, "Error creating address in UpdateContact : ", err)
			return 0,nil, err
		}
		AppendInt64UpdateField("addressId", &addId, &firstVal, &vals, &valString)
	}
	if c.Description != "" {
		AppendNStringUpdateField("description", &c.Description, &firstVal, &vals, &valString)
	}
	if c.Disabled != 0 {
		AppendNInt64UpdateField("disabled", &c.Disabled, &firstVal, &vals, &valString)
	}
	if c.Additional != "" {
		AppendNStringUpdateField("Additional", &c.Additional, &firstVal, &vals, &valString)
	}
	if c.SyncedWith != "" {
		AppendNStringUpdateField("syncedWith", &c.SyncedWith, &firstVal, &vals, &valString)
	}
	if c.Title != "" {
		AppendNStringUpdateField("Title", &c.Title, &firstVal, &vals, &valString)
	}
	if int64(c.TripType) != 0 {
		AppendNInt64UpdateField("tripTypeId", &c.TripType, &firstVal, &vals, &valString)
	}
	if int64(c.Type) != 0 {
		AppendNInt64UpdateField("type", &c.Type, &firstVal, &vals, &valString)
	}

	if firstVal {
		err = ErrNoChanges
		return
	}
	q := "UPDATE Contacts SET " + valString + " WHERE _contactId=?"
	vals = append(vals, c.Id)
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in 2nd dbCon.Exec for UpdateContact: %v ", err)

		return
	}
	rowCount, err = res.RowsAffected()

	if updateGz {
		for _, gz := range c.Address.GeoZones {
			err = UpdateAllKeyPointsForGeoZones([]int64{int64(gz.Id)}, dbCon)
			if err != nil {
				dbg.E(TAG, "Error in UpdateAllKeyPointsForGeoZones for CreateGeoZone: %v ", err)
				return
			}
		}
	}
	return
}

func DeletePOI(id int64, dbCon *sql.DB) (err error) {
	_, err = dbCon.Exec("DELETE FROM PointsOfInterest WHERE _pointOfInterestId=?", id)
	return
}

// GetPOIFromRes creates an address from an result - the order of fields needs to be correct
// (use POIQueryFields and if needed +","+AddressQueryFields),
// but they can be shifted (e.g. because of a join)
func GetPOIFromRes(row Scannable, fieldsBefore int, fieldsAfter int, withAddressData bool) (poi *PointOfInterest, err error) {
	return nil, errors.New("Not implemented")
}


// GetContactWithGeoZonesFromRes creates an contact, including GeoZones and address from an result
// the order of fields needs to be correct (ContactQueryFields + "," + AddressQueryFields + "," + GeoFenceRegionQueryFields)
// but they can be shifted (e.g. because of a join) by fieldsBefore and/or fieldsAfter
func GetContactWithGeoZonesFromRes(row Scannable, fieldsBefore int, fieldsAfter int, withAddressData bool) (contact *Contact, err error) {
	contact = &Contact{Address: &Address{}}
	var adrId sql.NullInt64
	err = SkippyScanRow(row, fieldsBefore, fieldsAfter+NoOfAddressQueryFields+NoOfGeoFenceRegionQueryFields+1, &contact.Id, &contact.Type, &contact.Title, &contact.Description, &contact.Additional, &adrId, &contact.TripType, &contact.Disabled, &contact.SyncedWith)
	if err != nil {
		dbg.E(TAG, "Error in first SkippyScanRow for GetContactWithGeoZonesFromRes: %v ", err)
		return
	}
	if !adrId.Valid || adrId.Int64 == 0 {
		dbg.W(TAG, "Contact with id %v has no address", contact.Id)
		return
	}

	contact.Address, err = GetAddressWithGeoZonesFromRes(row, NoOfContactQueryFields, fieldsAfter)

	return
}

// GetAddressFromRes creates an address from an result - the Order of address fields needs to be correct,
// (use AddressQueryFields)
// but they can be shifted (e.g. because of a join)
func GetAddressFromRes(row Scannable, fieldsBefore int, fieldsAfter int) (address *Address, err error) {

	address = &Address{}
	err = SkippyScanRow(row, fieldsBefore, fieldsAfter, &address.Id, &address.Street, &address.Postal, &address.City,
		&address.Additional1, &address.Additional2, &address.HouseNumber, &address.Title, &address.Latitude, &address.Longitude, &address.Fuel,&address.GeoCoder)
	if err != nil {
		dbg.E(TAG, "Error in GetAddressFromRes : %v ", err)
	}

	return
}

// GetGeoZoneFromRes scans a row containing a GeoZone into a GeoZone-object.
func GetGeoZoneFromRes(row Scannable, fieldsBefore int, fieldsAfter int) (geoZone *GeoFenceRegion, err error) {

	geoZone = &GeoFenceRegion{}

	var rectId sql.NullInt64
	err = SkippyScanRow(row, fieldsBefore, fieldsAfter+NoOfGeoFenceRegionQueryFields-NoOfNoGeometryGeoFenceRegionQueryFields-1, &geoZone.Id, &geoZone.OuterMinLat, &geoZone.OuterMinLon, &geoZone.OuterMaxLat, &geoZone.OuterMaxLon, &geoZone.Color, &rectId)
	if err != nil {
		dbg.E(TAG, "Error in 1st SkippyScanRow for GetGeoZoneFromRes: %v ", err)
		return
	}
	if rectId.Valid {

		geoZone.Rectangle, err = GetGeoRectangleFromRes(row, fieldsBefore+NoOfNoGeometryGeoFenceRegionQueryFields, fieldsAfter)

		if err != nil {
			dbg.E(TAG, "Error in GetGeoRectangleFromRes for GetGeoZoneFromRes: %v ", err)
			return
		}

	}
	return
}

// GetGeoRectangleFromRes scans a row containing a GeoRectangle into a GeoRectangle-object.
func GetGeoRectangleFromRes(row Scannable, fieldsBefore int, fieldsAfter int) (geoRectangle *GeoRectangle, err error) {
	geoRectangle = &GeoRectangle{}
	err = SkippyScanRow(row, fieldsBefore, fieldsAfter, &geoRectangle.Id, &geoRectangle.TopLeftLat, &geoRectangle.TopLeftLon, &geoRectangle.BotRightLat, &geoRectangle.BotRightLon)
	if err != nil {
		dbg.E(TAG, "Error in 1st SkippyScanRow for GetGeoRectangleFromRes: %v ", err)
	}
	return
}

// GetTripIdsWithContact returns trips with endKeyPoint in GeoZone if endKeyPoint is true,
// otherwise returns trips with startKeyPoint in GeoZone
func GetTripIdsWithContact(cId int64, isEndKeyPoint bool, dbCon *sql.DB) (tripIds string, err error) {
	var res *sql.Rows
	if isEndKeyPoint {
		res, err = dbCon.Query("SELECT _tripId FROM Trips WHERE endContactId=?", cId)
		if err != nil {
			dbg.E(TAG, "Error selecting tripId in GetTripIdsWithContact", err)
			return
		}
	} else {
		res, err = dbCon.Query("SELECT _tripId FROM Trips WHERE startContactId=?", cId)
		if err != nil {
			dbg.E(TAG, "Error selecting tripId in GetTripIdsWithContact", err)
			return
		}
	}
	firstRes := true
	for res.Next() {

		if firstRes {
			firstRes = false
		} else {
			tripIds += ","
		}

		var id string
		res.Scan(&id)
		tripIds += id
	}

	return
}

// CalcRetryTime determines the unix-timestamp when to retry an geocoding request. The more often or the more bad the failure
// the longer we need to wait.
func CalcRetryTime(tryCount int64, err error) (retryTime int64) {
	tryCount++
	if err == ErrEmptyResult { // Our GeoCoder did not find anything at the given address - retry tomorrow.

		retryTime = int64(time.Now().Unix() + OneDay)
		if tryCount > 20 {
			retryTime = int64(time.Now().Unix() + 7*OneDay)
		}
		if tryCount > 50 {
			retryTime = int64(time.Now().Unix() + 30*OneDay)
		}
		dbg.I(TAG, "We will retry on next trip request after one day (or 7 days if >20 tries,30 days if >50 tries), at unix time %d", retryTime)
	} else if err == ErrNeedFixBeforeRetry { // We probably got a bug - retry next day
		retryTime = int64(time.Now().Unix() + OneDay)

		dbg.I(TAG, "We will retry on next trip request after one day, at unix time %d", retryTime)
	} else { // GeoCoding-service was probably not reachable - retry in 30 seconds * tryCount^1.6, max. 12 hours.
		// so e.g. on the 20th try we are waiting one hour
		// TODO : Factor in our allowed daily usage

		repeatTime := int64(30 * math.Pow(float64(tryCount), 1.6))
		if repeatTime > OneDay/2 {
			repeatTime = OneDay / 2
		}
		dbg.E(TAG, "Error getting Address in retry from location : ", err)
		retryTime = int64(time.Now().Unix() + repeatTime)
		dbg.I(TAG, "We will retry on next trip request after %d seconds as this is try number %d, will retry at unix time %d", repeatTime, tryCount, retryTime)
	}
	return
}

// GetAdressFromString tries parsing an address string & returns a geocoded address.
func GetAddressFromString(addr string, client *http.Client,uId int64) (a *Address, err error) {
	a = &Address{}
	var req *http.Request
	q := fmt.Sprintf(GeoChainAddr+"/forward/%d/b/abc/%s",uId, url.QueryEscape(addr))
	req, err = http.NewRequest("GET", q, nil)
	if err != nil {
		dbg.E(TAG, "Error initializing httpRequest : ", err)
		FillUnknownAddress(a)
		return
	}
	if client == nil {
		client = &http.Client{}
	}
	/* Get Details */
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		dbg.E(TAG, "Error executing geocoda request: %s", err)
		FillUnknownAddress(a)
		return
	}
	var _body []byte
	_body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		if err != nil {
			dbg.E(TAG, "Error reading geocoda response: %s", err)
			FillUnknownAddress(a)
			return nil, ErrNeedFixBeforeRetry
		}
	}
	err = FillAddrFromGeocoderResp(string(_body), a,true)
	if err != nil {
		dbg.W(TAG, "Error filling address from resp : ", err)
		if dbg.Debugging {
			dbg.W(TAG, "Resp : %s", string(_body))
		}
	}
	return
}

// GetAddressesByWhere returns the addresses matching the given where-string & parameters.
func GetAddressesByWhere(where string, dbCon *sql.DB, params ...interface{}) (addresses []*Address, err error) {
	res, err := dbCon.Query("SELECT _addressId, street, postal,houseNumber, city, additional1, additional2,latitude,longitude,geoCoder FROM `Addresses` WHERE "+where, params...)
	if err != nil {
		dbg.E(TAG, "failed to get specific Addresses from DB", err)
		return nil, err
	}
	for res.Next() {
		address := &Address{}
		err = res.Scan(&address.Id, &address.Street, &address.Postal, &address.HouseNumber, &address.City, &address.Additional1, &address.Additional2, &address.Latitude, &address.Longitude,&address.GeoCoder)
		if err != nil {
			dbg.E(TAG, "failed to get specific Address from DB", err)
			return nil, err
		}
		addresses = append(addresses, address)
	}

	return addresses, nil
}

// GetAddressById returns an address by its given ID
func GetAddressById(addressId int64, dbCon *sql.DB) (address *Address, err error) {
	address = &Address{}
	err = dbCon.QueryRow("SELECT _addressId, street, postal,houseNumber, city, additional1, additional2,latitude,longitude,geoCoder FROM `Addresses` WHERE _addressId=?", addressId).Scan(&address.Id, &address.Street, &address.Postal, &address.HouseNumber, &address.City, &address.Additional1, &address.Additional2, &address.Latitude, &address.Longitude,&address.GeoCoder)
	if err != nil {
		dbg.E(TAG, "failed to get specific Addresses from DB", err)
		return nil, err
	}

	return address, nil
}

// GetAddressHashMap returns all addressIds from DB with key=postal_city_street_housenumber
func GetAddressHashMap(dbCon *sql.DB) (map[string]int64, error) {
	addresses := make(map[string]int64)
	var key string
	var err error

	rows, err := dbCon.Query("SELECT _addressId, street, postal, city, HouseNumber FROM `Addresses`")
	if err != nil {
		dbg.E(TAG, "failed to get rows from Addresses", err)
		return nil, err
	}

	// dirty hack to prepend a zero to incomplete postals
	// http://stackoverflow.com/questions/25637440/golang-how-to-pad-a-number-with-zeros-when-printing
	for rows.Next() {
		addr := Address{}
		err = rows.Scan(&addr.Id, &addr.Street, &addr.Postal, &addr.City, &addr.HouseNumber)
		key = fmt.Sprintf("%05v_%v_%v_%v", addr.Postal, addr.City, addr.Street, addr.HouseNumber)
		addresses[key] = int64(addr.Id)
	}
	// dbg.V(TAG, "getTrackPointsForTrack: got %d trackpoints for track %d... ", len(tps), trackId)

	if err = rows.Err(); err != nil {
		dbg.E(TAG, "getAddressHashMap rows-iteration-Error", err)
		return nil, err
	}

	return addresses, err
}
