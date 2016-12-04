// package syncMan is responsible for synchronizing Addressbooks (currently google) with the ODL-application to clone contacts.
package syncMan

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/addressManager"
	tripMan "github.com/OpenDriversLog/goodl-lib/jsonapi/tripMan/models"
	S "github.com/OpenDriversLog/goodl-lib/models/SQLite"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const TAG = "goodl-lib/jsonApi/syncMan"

// AutoRefresh checks if there are any changes in the users address book(s) and updates the contacts accordingly.
func AutoRefresh(clientsecretFilecontent []byte,uId int64, dbCon *sql.DB) (err error) {
	dbg.I(TAG, "Let's see if we have any retryTime-GoogleAddresses")
	_, err = RetryGoogleAddresses(uId,dbCon)
	if err != nil {
		dbg.E(TAG, "Error retrying google addresses : ", err)
		return
	}
	dbg.I(TAG, "Auto-refreshing stuff!")

	syncs, err := GetSyncs(dbCon)
	if err != nil {
		dbg.E(TAG, "Error getting syncs for autorefresh : ", err)
	}
	for _, s := range syncs {
		if s.UpdateFrequency > 0 && s.LastUpdate+s.UpdateFrequency < time.Now().Unix() {
			_, _, err := RefreshSync(clientsecretFilecontent, s.Id, uId, dbCon)
			if err != nil {
				dbg.E(TAG, "Error refreshing sync with id %d : ", s.Id, err)
				return err
			}
		}
	}
	return
}

// RefreshSync updates the given synchronisation, checking if there are any changes and updates the contacts accordingly.
func RefreshSync(clientsecretFilecontent []byte, syncId int64,uId int64, dbCon *sql.DB) (updatedContacts []*addressManager.Contact, updatedTrips []*tripMan.Trip, err error) {
	sync, err := GetSyncById(syncId, dbCon)
	if err != nil {
		dbg.E(TAG, "Error getting sync by Id in RefreshSync : ", err)
		return
	}
	var client *http.Client
	if sync.OAuth != nil {
		ctx, _ := context.WithTimeout(context.Background(), 120*time.Second)
		config, err := google.ConfigFromJSON(clientsecretFilecontent, "https://www.googleapis.com/auth/calendar")
		if err != nil {
			dbg.E(TAG, "Error getting Google OAuth-config", err)
			return nil, nil, err
		}
		exp := int64(sync.OAuth.ExpirationTime)
		if exp == 0 {
			exp = 1
		}
		token := &oauth2.Token{
			AccessToken:  string(sync.OAuth.AccessToken),
			TokenType:    "Bearer",
			RefreshToken: string(sync.OAuth.RefreshToken),
			Expiry:       time.Unix(exp, 0),
		}
		ts := config.TokenSource(ctx, token)
		token, err = ts.Token()
		if err != nil {
			dbg.E(TAG, "Error getting token : ", err)
			return nil, nil, err
		}
		client = config.Client(ctx, token)
		changedCs := make([]*addressManager.Contact, 0)
		idx := 1
		lastEntryPassed := false
		syncTime := time.Now().Unix()
		var firstUpdatedString string
		lastSyncString := string(sync.CardDavConfig.LastSyncKey)
		for !lastEntryPassed {
			dbg.I(TAG, "Getting google contact groups")

			q := fmt.Sprintf("https://www.google.com/m8/feeds/groups/default/full?start-index=%d", idx)
			if lastSyncString != "" {
				q += "&updated-min=" + lastSyncString

			}
			req, err := http.NewRequest("GET", q, nil)
			req.Header.Add("GData-Version", "3.0")
			if err != nil {
				dbg.E(TAG, "Error intialising group fetch request", err)
				return nil, nil, err
			}
			res, err := client.Do(req)
			if err != nil {
				dbg.E(TAG, "Error executing groups fetch request", err)
				return nil, nil, err
			}
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				dbg.E(TAG, "Error reading response body : ", err)
				return nil, nil, err
			}
			gs := &GoogleGroupsFeed{Entries: make([]GoogleGroupsItem, 0)}
			err = xml.Unmarshal([]byte(body), gs)
			if err != nil {
				dbg.E(TAG, "Error unmarshaling groups response : ", err)
				if dbg.Debugging {
					dbg.WTF(TAG, "Unreadable response : %s", body)
				}
				return nil, nil, err
			}

			if firstUpdatedString == "" && gs.UpdatedString != "" {
				firstUpdatedString = gs.UpdatedString
			}
			for _, g := range gs.Entries {
				gg := &GoogleGroup{
					Sync:       &Sync{Id: syncId},
					Key:        S.NString(g.Id),
					Name:       S.NString(g.Title),
					TripType:   3,
					LastUpdate: S.NInt64(g.Edited.Unix()),
				}
				prevgrps, err := GetGoogleGroupsByWhere("key=?", dbCon, gg.Key)
				if len(prevgrps) > 0 {
					gg.Id = prevgrps[0].Id
					_, err = UpdateGoogleGroup(gg, dbCon)
					if err != nil {
						dbg.E(TAG, "Error updating google group :", err)
						return nil, nil, err
					}
				} else {
					err = CreateGoogleGroup(gg, dbCon)
					if err != nil {
						dbg.E(TAG, "Error creating google group :", err)
						return nil, nil, err
					}
				}
			}

			idx = gs.StartIndex + gs.ItemsPerPage
			if idx >= gs.TotalResults {
				lastEntryPassed = true
			}
		}
		lastEntryPassed = false
		idx = 1
		geoCodingClient := &http.Client{}

		for !lastEntryPassed {
			dbg.I(TAG, "Getting google contacts.")
			q := fmt.Sprintf("https://www.google.com/m8/feeds/contacts/default/full?start-index=%d", idx)
			if lastSyncString != "" {
				q += "&updated-min=" + lastSyncString

			}
			req, err := http.NewRequest("GET", q, nil)
			req.Header.Add("GData-Version", "3.0")
			if err != nil {
				dbg.E(TAG, "Error intialising contact fetch request", err)
				return nil, nil, err
			}
			res, err := client.Do(req)
			if err != nil {
				dbg.E(TAG, "Error executing contact fetch request", err)
				return nil, nil, err
			}
			body, err := ioutil.ReadAll(res.Body)
			cs := &GoogleContactsFeed{Entries: make([]GoogleContactsItem, 0)}
			err = xml.Unmarshal([]byte(body), cs)
			if err != nil {
				dbg.E(TAG, "Error unmarshaling contacts response : ", err)
				return nil, nil, err
			}
			for _, c := range cs.Entries {
				if len(c.Addresses) == 0 {
					// no address given - do not do anything for this contact.
					continue
				}
				changedCss, err := UpdateByGoogleContact(c, syncId, sync.Name, dbCon,uId, geoCodingClient)
				if err != nil {
					dbg.E(TAG, "Error calling UpdateByGoogleContact : ", err)
					return nil, nil, err
				}
				if changedCss != nil {
					changedCs = append(changedCs, changedCss...)
				}
			}
			idx = cs.StartIndex + cs.ItemsPerPage
			if idx >= cs.TotalResults {
				lastEntryPassed = true
			}
		}

		/*affectedGzs := make([]int64, 0)
		for _, c := range changedCs {
			if c.Disabled < 1 {
				for _, gz := range c.Address.GeoZones {
					affectedGzs = append(affectedGzs, int64(gz.Id))
				}
			}
		}
		if len(affectedGzs) > 0 {
			err = addressManager.UpdateAllKeyPointsForGeoZones(affectedGzs, dbCon)
			if err != nil {
				dbg.E(TAG, "Error updating keypoints for affected geozones", err)
				return nil, nil, err
			}
		}*/
		// TODO : CreateContact might return affected Trips => sum them up
		updatedContacts = changedCs

		sync.LastUpdate = syncTime
		if S.NString(firstUpdatedString) != "" {
			sync.CardDavConfig.LastSyncKey = S.NString(firstUpdatedString)
		}
		_, err = UpdateCardDavConfig(sync.CardDavConfig, dbCon)
		if err != nil {
			dbg.E(TAG, "Error updating CardDavConfig : ", err)
			return nil, nil, err
		}
		_, err = UpdateSync(sync, dbCon)
		if err != nil {
			dbg.E(TAG, "Error updating Sync in DB : ", err)
			return nil, nil, err
		}

	} else {

		dbg.E(TAG, "Anything that is not OAuth is currently not implemented")
		err = errors.New("Authentication method not supported")
		return
	}

	dbg.W(TAG, "Returning updatedTrips currently not supported.")
	return
}

// UpdateByGoogleContact takes an google-contact and checks if any ODL-contacts need to be created or updated.
func UpdateByGoogleContact(c GoogleContactsItem, syncId int64, syncName string, dbCon *sql.DB, uId int64, geoCodingClient *http.Client) (changedContacts []*addressManager.Contact, err error) {
	if len(c.Addresses) == 0 {
		// no address given - do not do anything for this contact.
		return
	}
	sync := &Sync{Id: syncId}
	prevcs, err := GetGoogleContactsByWhere("key=?", dbCon, c.Id)
	gc := &GoogleContact{
		Sync:       sync,
		Key:        S.NString(c.Id),
		LastUpdate: S.NInt64(c.Edited.Unix()),
		Name:       S.NString(c.Title),
		Addresses:  make([]*GoogleAddress, 0),
	}
	changedContacts = make([]*addressManager.Contact, 0)
	newAddresses := make([]StructuredPostalAddress, 0)
	//addedAddresses := make(map[string]bool)
	if len(prevcs) > 0 {
		prevc := prevcs[0]
		gc.Id = prevc.Id
		nameChanged := gc.Name != prevc.Name
		for _, pa := range prevc.Addresses {
			if pa.Contact != nil && pa.Contact.Id > 0 {

				pc, err := addressManager.GetContact(int64(pa.Contact.Id), dbCon, true)
				if err != nil {
					if err == sql.ErrNoRows { // contact was deleted
						continue
					}
					dbg.E(TAG, "Error getting previous contact : ", c)
					return nil, err
				}
				if nameChanged && pc.Title == prevc.Name {
					pc.Title = gc.Name
				}
				syncNameChanged := false
				if string(pc.SyncedWith) != syncName {
					pc.SyncedWith = S.NString(syncName)
					syncNameChanged = true
				}

				addrRemoved := true
				for _, a := range c.Addresses {
					if a.FormattedAddress == string(pa.FormattedAddress) {
						addrRemoved = false
						break
					}
				}
				if addrRemoved {
					pc.Disabled = S.NInt64(1)
				}
				if addrRemoved || nameChanged || syncNameChanged {
					_, _, err = addressManager.UpdateContact(pc, dbCon)
					if err != nil {
						dbg.E(TAG, "Error updating previous contact : ", err)
						return nil, err
					}
					changedContacts = append(changedContacts, pc)
				}

			}
		}
		for _, a := range c.Addresses {
			found := false
			for _, pa := range prevc.Addresses {
				if a.FormattedAddress == string(pa.FormattedAddress) {
					found = true
					if string(pa.Rel) != a.Rel {
						pa.Rel = S.NString(a.Rel)
						_, err := UpdateGoogleAddress(pa, dbCon)
						if err != nil {
							dbg.E(TAG, "Error updating pa.Rel : ", err)
						}
						if pa.Contact != nil {
							pc, err := addressManager.GetContact(int64(pa.Contact.Id), dbCon, false)
							if err != nil {
								dbg.E(TAG, "Error getting previous contact : ", err)
								return nil, err
							}
							tt := GetTripTypeFromRel(string(pa.Rel))
							if tt != int64(pc.TripType) {
								pc.TripType = S.NInt64(tt)
								_, _, err = addressManager.UpdateContact(pc, dbCon)
								if err != nil {
									dbg.E(TAG, "Error updating previous contact : ", err)
									return nil, err
								}
							}
						}

					}
					gc.Addresses = append(gc.Addresses, pa)
					break
				}
			}
			if !found {
				newAddresses = append(newAddresses, a)
			}
		}

		_, err = UpdateGoogleContact(gc, dbCon)
		if err != nil {
			dbg.E(TAG, "Error updating google contact :", err)
			return nil, err
		}

	} else {
		newAddresses = c.Addresses
		err = CreateGoogleContact(gc, dbCon)
		if err != nil {
			dbg.E(TAG, "Error creating google contact :", err)
			return nil, err
		}

	}

	for _, a := range newAddresses {
		c, addr, err := UpdateFromAddress(&a, c, sync, dbCon, geoCodingClient, 0,uId)
		if err != nil && err != ErrAddressNotFound {
			dbg.E(TAG, "Error in UpdateFromAddress : ", err)
			return nil, err
		}
		if err != ErrAddressNotFound {
			changedContacts = append(changedContacts, c)
		}
		gc.Addresses = append(gc.Addresses, addr)

	}

	if len(newAddresses) > 0 {
		_, err = UpdateGoogleContact(gc, dbCon)
		if err != nil {
			dbg.E(TAG, "Error updating google contact : ", err)
			return
		}
	}
	return
}

var ErrAddressNotFound error = errors.New("Address not found")

// UpdateFromAddress updates the given GoogleContactsItem by the given StructuredPostalAddress-object, doing GeoCoding etc.
func UpdateFromAddress(a *StructuredPostalAddress, c GoogleContactsItem, sync *Sync, dbCon *sql.DB, geoCodingClient *http.Client, prevId int64, uId int64) (newC *addressManager.Contact, newAddr *GoogleAddress, err error) {

	newAddr = &GoogleAddress{
		FormattedAddress: S.NString(a.FormattedAddress),
		Rel:              S.NString(a.Rel),
		Sync:             sync,
		TripType:         S.NInt64(GetTripTypeFromRel(a.Rel)),
		Id:               S.NInt64(prevId),
	}

	address, err := addressManager.GetAddressFromString(a.FormattedAddress, geoCodingClient, uId)
	if err != nil {
		dbg.W(TAG, "Error getting contact address : ", err)
		newAddr.RetryTime = S.NInt64(addressManager.CalcRetryTime(int64(newAddr.TryCount), err))
		dbg.W(TAG, "RetryTime : ", newAddr.RetryTime)
		if prevId > 0 {
			_, err = UpdateGoogleAddress(newAddr, dbCon)
		} else {
			err = CreateGoogleAddress(newAddr, dbCon)
		}
		if err != nil {
			dbg.E(TAG, "Error creating google address : ", err)
		}
		return nil, newAddr, ErrAddressNotFound
	}
	newC = &addressManager.Contact{
		Title:      S.NString(c.Title),
		Address:    address,
		TripType:   newAddr.TripType,
		SyncedWith: S.NString(sync.Name),
	}
	_, err = addressManager.CreateContact(newC, dbCon)
	if err != nil {
		dbg.E(TAG, "Error creating google synced contact!", err)
		return nil, newAddr, err
	}
	newAddr.Contact = newC
	if prevId > 0 {
		newAddr.RetryTime = -1
		_, err = UpdateGoogleAddress(newAddr, dbCon)
	} else {
		err = CreateGoogleAddress(newAddr, dbCon)
	}
	if err != nil {
		dbg.E(TAG, "Error creating google address : ", err)
	}

	return
}

// RetryGoogleAddresses retrys GeoCoding for addresses that could not be geocoded at the last try.
func RetryGoogleAddresses(uId int64,dbCon *sql.DB) (adrCount int, err error) {

	client := &http.Client{}
	dbg.I(TAG,"Start GetGoogleContactsByWhere")
	cs, err := GetGoogleContactsByWhere("GoogleAddresses.retrytime!=0 AND GoogleAddresses.retrytime<?", dbCon, time.Now().Unix())
	if err != nil {
		dbg.E(TAG, "Error getting GoogleAddresses open for retry", err)
		return
	}
	dbg.I(TAG,"End GetGoogleContactsByWhere")

	for _, c := range cs {
		dbg.I(TAG,"Start GetSyncById")

		c.Sync, err = GetSyncById(c.Sync.Id, dbCon)
		if err != nil {
			dbg.E(TAG, "Error at RetryGoogleAddresses/GetSyncById : ", err)
		}
		dbg.I(TAG,"End GetSyncById")

		for _, a := range c.Addresses {
			adrCount++
			sa := StructuredPostalAddress{
				Rel:              string(a.Rel),
				FormattedAddress: string(a.FormattedAddress),
			}
			gc := GoogleContactsItem{
				Edited:    time.Unix(int64(c.LastUpdate), 0),
				Title:     string(c.Name),
				Addresses: []StructuredPostalAddress{sa},
				Id:        string(c.Key),
			}
			dbg.I(TAG,"Start UpdateFromAddress")
			_, _, err = UpdateFromAddress(&sa, gc, c.Sync, dbCon, client, int64(a.Id),uId)
			if err != nil && err != ErrAddressNotFound {
				dbg.E(TAG, "Error in UpdateFromAddress : ", err)
				return
			}
			dbg.I(TAG,"End UpdateFromAddress")
			err = nil

		}
	}
	return
}

// GetTripTypeFromRel converts the "Rel"-field of a google contact to an appropriate tripType
func GetTripTypeFromRel(rel string) (tripType int64) {
	if strings.Contains(rel, "work") || strings.Contains(rel, "other") {
		return 3
	}
	return 1
}

// CreateGoogleRefreshToken Creates a refresh-token from a google authCode.
func CreateGoogleRefreshToken(authCode string, clientsecretFilecontent []byte, dbCon *sql.DB) (auth OAuth, err error) {

	client := http.Client{}
	config, err := google.ConfigFromJSON(clientsecretFilecontent, "https://www.googleapis.com/auth/contacts.readonly")
	if err != nil {
		dbg.E(TAG, "Error getting Google OAuth-config", err)
		return auth, err
	}
	data := url.Values{}
	data.Add("code", authCode)
	data.Add("client_id", config.ClientID)
	data.Add("client_secret", config.ClientSecret)
	data.Add("redirect_uri", "postmessage")
	data.Add("grant_type", "authorization_code")
	data.Add("access_type", "offline")

	req, err := http.NewRequest("POST", "https://www.googleapis.com/oauth2/v4/token", strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	res, err := client.Do(req)
	if err != nil {
		dbg.E(TAG, "Error getting refresh token : ", err)
		return auth, err
	}
	if res.StatusCode != 200 {
		r, err := ioutil.ReadAll(res.Body)
		if err != nil {
			dbg.E(TAG, "Error getting refresh token & in addition unable to read response body : %+v", res)
			return auth, err
		}
		dbg.E(TAG, "Error getting refresh token - response : %+v \r\n body : %s", res, r)
		return auth, err
	}
	r := &GoogleRefreshTokenJSONAnswer{}
	rb, err := ioutil.ReadAll(res.Body)
	if err != nil {
		dbg.E(TAG, "Unable to read refresh token response body : %+v", res)
		return auth, err
	}
	err = json.Unmarshal(rb, &r)
	if err != nil {
		dbg.E(TAG, "Unable to json-parse refresh token response body : %+v", res)
		return auth, err
	}
	auth.RefreshToken = S.NString(r.Refresh_token)
	if len(auth.RefreshToken) == 0 {
		dbg.E(TAG, "Empty refresh token in result : %+v", res)
		return auth, err
	}
	dbg.I(TAG, "Saving refresh token")
	dbRes, err := dbCon.Exec("INSERT INTO OAUTH (refreshToken) VALUES (?)", auth.RefreshToken)
	if err != nil {
		dbg.E(TAG, "Error saving new refreshToken : ", err)
		return auth, err
	}
	var id int64
	id, err = dbRes.LastInsertId()
	if err != nil {
		dbg.E(TAG, "Error getting LastInsertId : ", err)
		auth.Id = -1
		return auth, err
	}
	auth.Id = S.NInt64(id)
	return
}

// TODO: Make this CalDav-stuff work
/*req, err := http.NewRequest("PROPFIND", cardDavInitUri, nil)
if err != nil {
	dbg.E(TAG, "Error starting request",err)
}
res, err := client.Do(req)
if err != nil {
	dbg.E(TAG, "Error executing propfind-request : ", err)
	if dbg.Debugging {
		dbg.E(TAG, "req : %+v \r\n res : %+v", req, res)
	}
	return nil, nil, err
}
if res.StatusCode != 301 {
	dbg.E(TAG, "Propfind-Request did not return status 301 :(")
	if dbg.Debugging {
		dbg.E(TAG, "req : %+v \r\n res : %+v", req, res)
	}
	return nil, nil, errors.New("Propfind-Request did not return status 301")
}
loc, err := res.Location()
if err != nil {
	dbg.E(TAG, "Propfind-Request did not contain location :(")
	if dbg.Debugging {
		dbg.E(TAG, "req : %+v \r\n res : %+v", req, res)
	}
	return nil, nil, err
}
sc := &SyncCollection{
	D:         "DAV:",
	SyncLevel: 1,
	Prop:      "<d:displayname />",
}
x, err := xml.Marshal(sc)
if err != nil {
	dbg.E(TAG, "Error marshalling SyncCollection xml : ", err)
	return nil, nil, err
}
req, err = http.NewRequest("POST", loc.RequestURI(), bytes.NewReader(x))
if err != nil {
	dbg.E(TAG, "Error initialising addressbook request!", err)
	return nil, nil, err
}

res, err = client.Do(req)
if err != nil {
	dbg.E(TAG, "Error executing addressbook request!", err)
	if dbg.Debugging {
		dbg.E(TAG, "Request %+v returned \r\n\r\n res: %+v")
	}
	return nil, nil, err
}
if dbg.Debugging {
	dbg.WTF(TAG, "Request %+v returned \r\n\r\n res: %+v")
}*/
