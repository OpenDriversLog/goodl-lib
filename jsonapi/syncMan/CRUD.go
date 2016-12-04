package syncMan

import (
	"database/sql"
	"errors"
	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/addressManager"
	S "github.com/OpenDriversLog/goodl-lib/models/SQLite"
	. "github.com/OpenDriversLog/goodl-lib/tools"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"time"

	"fmt"
	"net/http"
)

// GetSyncs returns all entries from Sync-Table
func GetSyncs(dbCon *sql.DB) (syncs []*Sync, err error) {
	return GetSyncsByWhere("", dbCon)
}

// GetSyncById returns the Sync with the given ID.
func GetSyncById(id int64, dbCon *sql.DB) (sync *Sync, err error) {
	res, err := GetSyncsByWhere("_syncId=?", dbCon, id)
	if err != nil {
		dbg.E(TAG, "Error in GetSyncById : ", err)
		return
	}
	if len(res) != 1 {
		return nil, sql.ErrNoRows
	}
	sync = res[0]
	return
}

// GetSyncsByWhere returns all syncs matching the given where-query and parameters.
func GetSyncsByWhere(where string, dbCon *sql.DB, params ...interface{}) (syncs []*Sync, err error) {
	syncs = make([]*Sync, 0)
	q := `SELECT _syncId, name, Sync.type, priority, lastUpdate, created,
	 updateFrequency,
	  CD._cardDavConfigId,CD.type,CD.rootUri,CD.addressBookName,CD.principalName,CD.lastSyncKey,
	  CL._calDavConfigId,CL.type,CL.rootUri,CL.calendarName,CL.principalName,
	  OA._oAuthId,OA.refreshToken,OA.accessToken,OA.expirationTime,
	  HB._httpBasicAuthId,HB.usr,HB.passwd,
	  HD._httpDigestAuthId,HD.usr,HD.passwd
	  FROM Sync
	 LEFT JOIN CardDavConfig CD ON cardDavConfigId=_cardDavConfigId
	 LEFT JOIN CalDavConfig CL ON calDavConfigId=_calDavConfigId
	 LEFT JOIN oAuth OA on oAuthId=_oAuthId
	 LEFT JOIN httpBasicAuth HB on httpBasicAuthId=_httpBasicAuthId
	 LEFT JOIN httpDigestAuth HD on httpDigestAuthId=_httpBasicAuthId`
	if where != "" {
		q += " WHERE " + where
	}
	res, err := dbCon.Query(q, params...)
	if err != nil {
		if err != nil {
			dbg.E(TAG, "unable to get Syncs", err)
			return
		}
	}

	for res.Next() {
		sync := &Sync{
			OAuth:          &OAuth{},
			HttpBasicAuth:  &HttpBasicAuth{},
			HttpDigestAuth: &HttpDigestAuth{},
			CardDavConfig:  &CardDavConfig{},
			CalDavConfig:   &CalDavConfig{},
		}
		err = res.Scan(&sync.Id, &sync.Name, &sync.Type, &sync.Priority, &sync.LastUpdate, &sync.Created,
			&sync.UpdateFrequency, &sync.CardDavConfig.Id, &sync.CardDavConfig.Type, &sync.CardDavConfig.RootUri,
			&sync.CardDavConfig.AddressBookName, &sync.CardDavConfig.PrincipalName, &sync.CardDavConfig.LastSyncKey,
			&sync.CalDavConfig.Id, &sync.CalDavConfig.Type, &sync.CalDavConfig.RootUri,
			&sync.CalDavConfig.CalendarName, &sync.CalDavConfig.PrincipalName,
			&sync.OAuth.Id, &sync.OAuth.RefreshToken, &sync.OAuth.AccessToken, &sync.OAuth.ExpirationTime,
			&sync.HttpBasicAuth.Id, &sync.HttpBasicAuth.Usr, &sync.HttpBasicAuth.Password,
			&sync.HttpDigestAuth.Id, &sync.HttpDigestAuth.Usr, &sync.HttpDigestAuth.Password)
		if err != nil {
			dbg.E(TAG, "Unable to scan sync!", err)
			return
		}
		if sync.UpdateFrequency > 0 {
			sync.NextUpdate = sync.LastUpdate + sync.UpdateFrequency
		} else {
			sync.NextUpdate = -1
		}
		syncs = append(syncs, sync)
	}
	return
}

// CreateSync creates a new sync
func CreateSync(sync *Sync, dbCon *sql.DB) (key int64, err error) {

	sync.Created = int64(time.Now().Unix())
	vals := []interface{}{sync.Name, sync.Created}
	valString := "?,?"

	insFields := "name,created"
	if sync.CardDavConfig != nil {
		err = CreateCardDavConfig(sync.CardDavConfig, dbCon)
		if err != nil {
			dbg.E(TAG, "Error creating cardDav-config : ", err)
			return -1, err
		}
		insFields += ",cardDavConfigId"
		valString += ",?"
		vals = append(vals, sync.CardDavConfig.Id)
	}
	if sync.CalDavConfig != nil {
		err = CreateCalDavConfig(sync.CalDavConfig, dbCon)
		if err != nil {
			dbg.E(TAG, "Error creating calDav-config : ", err)
			return -1, err
		}
		insFields += ",calDavConfigId"
		valString += ",?"
		vals = append(vals, sync.CalDavConfig.Id)
	}
	if sync.HttpBasicAuth != nil {
		err = CreateHttpBasicAuth(sync.HttpBasicAuth, dbCon)
		if err != nil {
			dbg.E(TAG, "Error creating HttpBasicAuth : ", err)
			return -1, err
		}
		insFields += ",httpBasicAuthId"
		valString += ",?"
		vals = append(vals, sync.HttpBasicAuth.Id)
	}
	if sync.HttpDigestAuth != nil {
		err = CreateHttpDigestAuth(sync.HttpDigestAuth, dbCon)
		if err != nil {
			dbg.E(TAG, "Error creating HttpDigestAuth : ", err)
			return -1, err
		}
		insFields += ",httpDigestAuthId"
		valString += ",?"
		vals = append(vals, sync.HttpDigestAuth.Id)
	}
	if sync.OAuth != nil { // oAuth was created before
		if sync.OAuth.Id < 1 {
			dbg.E(TAG, "OAuth needs to be initialised before passing into CreateSync")
			return -1, errors.New("OAuth needs to be initialised before passing into CreateSync")
		}
		insFields += ",oAuthId"
		valString += ",?"
		vals = append(vals, sync.OAuth.Id)
	}
	if sync.Type != "" {
		insFields += ",type"
		valString += ",?"
		vals = append(vals, sync.Type)
	}
	if sync.Priority != 0 {
		insFields += ",priority"
		valString += ",?"
		vals = append(vals, sync.Priority)
	}
	if sync.UpdateFrequency != 0 {
		if sync.UpdateFrequency < 3600 && sync.UpdateFrequency > 0 { // Minimum interval is 1 hour
			sync.UpdateFrequency = 3600
		}
		insFields += ",updateFrequency"
		valString += ",?"
		vals = append(vals, sync.UpdateFrequency)
	}
	q := "INSERT INTO Sync(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for CreateSyncs: %v ", err)
		return
	}

	key, err = res.LastInsertId()
	sync.Id = key
	return
}

// UpdateSync updates a sync
func UpdateSync(sync *Sync, dbCon *sql.DB) (rowCount int64, err error) {

	vals := []interface{}{}
	firstVal := true
	valString := ""

	if sync.Type != "" {
		AppendStringUpdateField("type", &sync.Type, &firstVal, &vals, &valString)
	}
	if sync.CardDavConfig != nil {
		if sync.CardDavConfig.Id < 1 {
			err = CreateCardDavConfig(sync.CardDavConfig, dbCon)
			if err != nil {
				dbg.E(TAG, "Error creating cardDav-config : ", err)
				return -1, err
			}
		}
		AppendNInt64UpdateField("cardDavConfigId", &sync.CardDavConfig.Id, &firstVal, &vals, &valString)
	} else {
		del := S.NInt64(-1337)
		AppendNInt64UpdateField("cardDavConfigId", &del, &firstVal, &vals, &valString)
	}
	if sync.CalDavConfig != nil {
		if sync.CalDavConfig.Id < 1 {
			err = CreateCalDavConfig(sync.CalDavConfig, dbCon)
			if err != nil {
				dbg.E(TAG, "Error creating calDav-config : ", err)
				return -1, err
			}
		}
		AppendNInt64UpdateField("calDavConfigId", &sync.CalDavConfig.Id, &firstVal, &vals, &valString)

	} else {
		del := S.NInt64(-1337)
		AppendNInt64UpdateField("calDavConfigId", &del, &firstVal, &vals, &valString)
	}
	if sync.HttpBasicAuth != nil {
		if sync.HttpBasicAuth.Id < 1 {
			err = CreateHttpBasicAuth(sync.HttpBasicAuth, dbCon)
			if err != nil {
				dbg.E(TAG, "Error creating HttpBasicAuth : ", err)
				return -1, err
			}
		}
		AppendNInt64UpdateField("httpBasicAuthId", &sync.HttpBasicAuth.Id, &firstVal, &vals, &valString)

	} else {
		del := S.NInt64(-1337)
		AppendNInt64UpdateField("httpBasicAuthId", &del, &firstVal, &vals, &valString)
	}
	if sync.HttpDigestAuth != nil {
		if sync.HttpDigestAuth.Id < 1 {
			err = CreateHttpDigestAuth(sync.HttpDigestAuth, dbCon)
			if err != nil {
				dbg.E(TAG, "Error creating HttpDigestAuth : ", err)
				return -1, err
			}
		}
		AppendNInt64UpdateField("httpDigestAuthId", &sync.HttpDigestAuth.Id, &firstVal, &vals, &valString)

	} else {
		del := S.NInt64(-1337)
		AppendNInt64UpdateField("httpDigestAuthId", &del, &firstVal, &vals, &valString)
	}
	if sync.OAuth != nil { // oAuth was created before
		if sync.OAuth.Id < 1 {
			dbg.E(TAG, "OAuth needs to be initialised before passing into CreateSync")
			return -1, errors.New("OAuth needs to be initialised before passing into CreateSync")
		}
		AppendNInt64UpdateField("oAuthId", &sync.OAuth.Id, &firstVal, &vals, &valString)

	} else {
		del := S.NInt64(-1337)
		AppendNInt64UpdateField("oAuthId", &del, &firstVal, &vals, &valString)
	}
	if sync.Priority != 0 {
		AppendInt64UpdateField("priority", &sync.Priority, &firstVal, &vals, &valString)
	}
	if sync.LastUpdate != 0 {
		AppendInt64UpdateField("lastUpdate", &sync.LastUpdate, &firstVal, &vals, &valString)
	}
	if sync.UpdateFrequency != 0 {
		if sync.UpdateFrequency < 3600 && sync.UpdateFrequency > 0 { // Minimum interval is 1 hour
			sync.UpdateFrequency = 3600
		}
		AppendInt64UpdateField("updateFrequency", &sync.UpdateFrequency, &firstVal, &vals, &valString)
	}
	if firstVal {
		err = ErrNoChanges
		return
	}
	q := "UPDATE Sync SET " + valString + " WHERE _syncId=?"
	vals = append(vals, sync.Id)
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for UpdateSync: %v ", err)

		return
	}
	rowCount, err = res.RowsAffected()
	if err != nil {
		dbg.E(TAG, "Error in UpdateSync RowsAffected", err)
		return
	}
	err = CleanUpAuths(dbCon)
	if err != nil {
		dbg.E(TAG, "Error in UpdateSync CleanUpAuths : ", err)
		return
	}
	return
}

// DeleteSync deletes the sync with the given ID.
func DeleteSync(id int64, clientsecretFilecontent []byte, dbCon *sql.DB) (rowCount int64, err error) {
	var res sql.Result

	// Revoke the token!
	// https://accounts.google.com/o/oauth2/revoke?token=
	config, err := google.ConfigFromJSON(clientsecretFilecontent, "https://www.googleapis.com/auth/calendar")
	if err != nil {
		dbg.E(TAG, "Error getting Google OAuth-config", err)
		return -1, err
	}
	sync, err := GetSyncById(id, dbCon)
	if err != nil {
		dbg.E(TAG, "Can't find sync by id : ", err)
		return -1, err
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
	ctx, _ := context.WithTimeout(context.Background(), 120*time.Second)
	ts := config.TokenSource(ctx, token)
	token, err = ts.Token()

	if err != nil {
		dbg.I(TAG, "Error getting token - probably already revoked.")
	} else {
		client := &http.Client{}
		req, err := http.NewRequest("GET", fmt.Sprintf("https://accounts.google.com/o/oauth2/revoke?token=%s", token.AccessToken), nil)
		if err != nil {
			dbg.E(TAG, "Error creating revoke request : ", err)
			return -1, err
		}
		res, err := client.Do(req)
		if err != nil || res.StatusCode != 200 {
			dbg.E(TAG, "Error revoking token - result : %+v, \r\n error : ", res, err)
			return -1, err
		}
	}
	res, err = dbCon.Exec("DELETE FROM Sync WHERE _syncId=?", id)
	if err != nil {
		dbg.E(TAG, "Error in DeleteSync: ", err)
		return
	} else {
		rowCount, err = res.RowsAffected()
		if err != nil {
			dbg.E(TAG, "Error in DeleteSync get RowsAffected : ", err)
			return
		}
	}
	err = CleanUpAuths(dbCon)
	if err != nil {
		dbg.E(TAG, "Error in DeleteSync CleanUpAuths : ", err)
		return
	}
	return
}


//CleanUpAuths Deletes unused authentications
func CleanUpAuths(dbCon *sql.DB) (err error) {
	res, err := dbCon.Exec(`
DELETE FROM CardDavConfig WHERE (SELECT COUNT(cardDavConfigId) FROM Sync where _cardDavConfigId=cardDavConfigId)=0;
DELETE FROM CalDavConfig WHERE (SELECT COUNT(calDavConfigId) FROM Sync where _calDavConfigId=calDavConfigId)=0;
DELETE FROM HttpBasicAuth WHERE (SELECT COUNT(httpBasicAuthId) FROM Sync where _httpBasicAuthId=httpBasicAuthId)=0;
DELETE FROM HttpDigestAuth WHERE (SELECT COUNT(httpDigestAuthId) FROM Sync where _httpDigestAuthId=httpDigestAuthId)=0;
DELETE FROM oAuth WHERE (SELECT COUNT(oAuthId) FROM Sync where _oAuthId=oAuthId)=0;
DELETE FROM GoogleGroups WHERE (SELECT COUNT(_syncId) FROM Sync where _syncId=syncId)=0;
DELETE FROM GoogleContacts WHERE (SELECT COUNT(_syncId) FROM Sync where _syncId=syncId)=0;
DELETE FROM GoogleContacts_Groups WHERE (SELECT COUNT(_googleGroupId) FROM GoogleGroups where _googleGroupId=googleGroupId)=0;
`)
	if err != nil {
		dbg.E(TAG, "Error cleaning up authentications : ", err)
		return err
	}
	rCount, err := res.RowsAffected()
	if err != nil {
		dbg.E(TAG, "Error getting cleanup rows affected : ", err)
		return err
	}
	if rCount > 0 {
		dbg.I(TAG, "Removed %d unused sync configurations/authentications")
	}

	return
}

// GetEmptySync returns an empty sync-object
func GetEmptySync() (sync *Sync, err error) {
	sync = &Sync{}
	return
}

// CreateCardDavConfig creates a new CardDavConfig in the database.
func CreateCardDavConfig(config *CardDavConfig, dbCon *sql.DB) (err error) {
	vals := []interface{}{config.Type}
	valString := "?"

	insFields := "type"

	if config.AddressBookName != "" {
		insFields += ",addressBookName"
		valString += ",?"
		vals = append(vals, config.AddressBookName)
	}
	if config.PrincipalName != "" {
		insFields += ",principalName"
		valString += ",?"
		vals = append(vals, config.PrincipalName)
	}
	if config.RootUri != "" {
		insFields += ",rootUri"
		valString += ",?"
		vals = append(vals, config.RootUri)
	}
	if config.LastSyncKey != "" {
		insFields += ",lastSyncKey"
		valString += ",?"
		vals = append(vals, config.LastSyncKey)
	}
	q := "INSERT INTO CardDavConfig(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for CreateCardDavConfig: %v ", err)
		return
	}
	var key int64
	key, err = res.LastInsertId()
	config.Id = S.NInt64(key)
	return
}

// UpdateCardDavConfig updates the given CardDavConfig.
func UpdateCardDavConfig(config *CardDavConfig, dbCon *sql.DB) (rowCount int64, err error) {
	vals := []interface{}{}
	firstVal := true
	valString := ""
	if config.AddressBookName != "" {
		AppendNStringUpdateField("addressBookName", &config.AddressBookName, &firstVal, &vals, &valString)
	}
	if config.PrincipalName != "" {
		AppendNStringUpdateField("principalName", &config.PrincipalName, &firstVal, &vals, &valString)
	}
	if config.RootUri != "" {
		AppendNStringUpdateField("rootUri", &config.RootUri, &firstVal, &vals, &valString)
	}
	if config.LastSyncKey != "" {
		AppendNStringUpdateField("lastSyncKey", &config.LastSyncKey, &firstVal, &vals, &valString)
	}
	if firstVal {
		err = ErrNoChanges
		return
	}
	q := "UPDATE CardDavConfig SET " + valString + " WHERE _cardDavConfigId=?"
	vals = append(vals, config.Id)
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for Updateconfig: %v ", err)
		return
	}
	rowCount, err = res.RowsAffected()
	if err != nil {
		dbg.E(TAG, "Error in Updateconfig RowsAffected", err)
		return
	}
	return
}

// CreateCalDavConfig creates a new CalDavConfig in the database.
func CreateCalDavConfig(config *CalDavConfig, dbCon *sql.DB) (err error) {
	vals := []interface{}{config.Type}
	valString := "?"

	insFields := "type"

	if config.CalendarName != "" {
		insFields += ",calendarName"
		valString += ",?"
		vals = append(vals, config.CalendarName)
	}
	if config.PrincipalName != "" {
		insFields += ",principalName"
		valString += ",?"
		vals = append(vals, config.PrincipalName)
	}
	if config.RootUri != "" {
		insFields += ",rootUri"
		valString += ",?"
		vals = append(vals, config.RootUri)
	}
	q := "INSERT INTO CalDavConfig(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for CreateCalDavConfig: %v ", err)
		return
	}
	var key int64
	key, err = res.LastInsertId()
	config.Id = S.NInt64(key)
	return
}

// CreateHttpBasicAuth creates a new HttpBasicAuth in the database.
func CreateHttpBasicAuth(auth *HttpBasicAuth, dbCon *sql.DB) (err error) {
	vals := []interface{}{auth.Usr}
	valString := "?"

	insFields := "usr"

	if auth.Password != "" {
		insFields += ",password"
		valString += ",?"
		vals = append(vals, auth.Password)
	}
	q := "INSERT INTO HttpBasicAuth(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for CreateHttpBasicAuth: %v ", err)
		return
	}
	var key int64
	key, err = res.LastInsertId()
	auth.Id = S.NInt64(key)
	return
}

// CreateHTTPDigestAuth creates a new HttpDigestAuth in the database.
func CreateHttpDigestAuth(auth *HttpDigestAuth, dbCon *sql.DB) (err error) {
	vals := []interface{}{auth.Usr}
	valString := "?"

	insFields := "usr"

	if auth.Password != "" {
		insFields += ",password"
		valString += ",?"
		vals = append(vals, auth.Password)
	}
	q := "INSERT INTO HttpDigestAuth(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for CreateHttpDigestAuth: %v ", err)
		return
	}
	var key int64
	key, err = res.LastInsertId()
	auth.Id = S.NInt64(key)
	return
}

// GetGoogleGroupsByWhere returns the GoogleGroups found by the given where-clause and parameters.
func GetGoogleGroupsByWhere(where string, dbCon *sql.DB, params ...interface{}) (groups []*GoogleGroup, err error) {
	groups = make([]*GoogleGroup, 0)
	q := `SELECT _googleGroupId, key,name,tripType,lastUpdate,syncId
	FROM GoogleGroups`
	if where != "" {
		q += " WHERE " + where
	}
	res, err := dbCon.Query(q, params...)
	if err != nil {
		if err != nil {
			dbg.E(TAG, "unable to get Syncs", err)
			return
		}
	}

	for res.Next() {
		group := &GoogleGroup{
			Sync: &Sync{},
		}
		err = res.Scan(&group.Id, &group.Key, &group.Name, &group.TripType, &group.LastUpdate, &group.Sync.Id)
		if err != nil {
			dbg.E(TAG, "Unable to scan GoogleGroup!", err)
			return
		}
		groups = append(groups, group)
	}
	return
}

// GetGoogleContactsByWhere returns the GoogleContacts found by the given where-clause and parameters.
func GetGoogleContactsByWhere(where string, dbCon *sql.DB, params ...interface{}) (contacts []*GoogleContact, err error) {
	contacts = make([]*GoogleContact, 0)
	q := `SELECT _googleContactId, key,name,lastUpdate,GoogleContacts.syncId, GoogleAddresses.contactId, GoogleAddresses.formattedAddress,GoogleAddresses.rel, retryTime, tryCount,tripType,GoogleAddresses.syncId,GoogleAddresses._googleAddressId
	FROM GoogleContacts LEFT JOIN GoogleContacts_Addresses ON _googleContactId=GoogleContacts_Addresses.googlecontactId
	LEFT JOIN GoogleAddresses ON GoogleContacts_Addresses.googleAddressId=_googleAddressId`
	contactsById := make(map[int64]*GoogleContact)
	if where != "" {
		q += " WHERE " + where
	}
	res, err := dbCon.Query(q, params...)
	if err != nil {
		if err != nil {
			dbg.E(TAG, "unable to get GoogleContacts", err)
			return
		}
	}

	for res.Next() {

		contact := &GoogleContact{
			Sync: &Sync{},
		}
		var addrSyncId S.NInt64
		addr := &GoogleAddress{Contact: &addressManager.Contact{}}
		err = res.Scan(&contact.Id, &contact.Key, &contact.Name, &contact.LastUpdate, &contact.Sync.Id, &addr.Contact.Id, &addr.FormattedAddress, &addr.Rel, &addr.RetryTime, &addr.TryCount, &addr.TripType, &addrSyncId, &addr.Id)
		pc := contactsById[int64(contact.Id)]
		if pc != nil {
			contact = pc
		}
		if addr.Id > 0 {
			addr.Sync = &Sync{Id: int64(addrSyncId)}
			contact.Addresses = append(contact.Addresses, addr)
		}
		if err != nil {
			dbg.E(TAG, "Unable to scan GoogleGroup!", err)
			return
		}
		if pc == nil {
			contacts = append(contacts, contact)
			contactsById[int64(contact.Id)] = contact
		}
	}
	return
}
// CreateGoogleGroup creates a new GoogleGroup in the database.
func CreateGoogleGroup(group *GoogleGroup, dbCon *sql.DB) (err error) {
	vals := []interface{}{group.Key}
	valString := "?"
	insFields := "key"

	if group.Name != "" {
		insFields += ",name"
		valString += ",?"
		vals = append(vals, group.Name)
	}
	if group.Sync != nil {
		insFields += ",syncId"
		valString += ",?"
		vals = append(vals, group.Sync.Id)
	}
	if group.TripType != 0 {
		insFields += ",tripType"
		valString += ",?"
		vals = append(vals, group.TripType)
	}
	q := "INSERT INTO GoogleGroups(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for CreateGoogleGroup: %v ", err)
		return
	}
	var key int64
	key, err = res.LastInsertId()
	group.Id = S.NInt64(key)
	return
}

// CreateGoogleContact creates a new GoogleContact in the database.
func CreateGoogleContact(contact *GoogleContact, dbCon *sql.DB) (err error) {
	vals := []interface{}{contact.Key}
	valString := "?"

	insFields := "key"

	if contact.Name != "" {
		insFields += ",name"
		valString += ",?"
		vals = append(vals, contact.Name)
	}

	if contact.Sync != nil {
		insFields += ",syncId"
		valString += ",?"
		vals = append(vals, contact.Sync.Id)
	}
	if contact.LastUpdate != 0 {
		insFields += ",lastUpdate"
		valString += ",?"
		vals = append(vals, contact.LastUpdate)
	}

	q := "INSERT INTO GoogleContacts(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for CreateGooglecontact: %v ", err)
		return
	}
	q = ""
	var key int64
	key, err = res.LastInsertId()
	if contact.Groups != nil && len(contact.Groups) != 0 {
		q = ""
		vals = make([]interface{}, 0)
		for _, g := range contact.Groups {
			q += "INSERT INTO GoogleContacts_Groups (googleContactId,googleGroupId) VALUES(?,?);"
			vals = append(vals, contact.Id, g.Id)
		}
	}
	if contact.Addresses != nil && len(contact.Addresses) > 0 {
		vals = make([]interface{}, 0)
		for _, a := range contact.Addresses {
			if a.Id < 1 {
				err = CreateGoogleAddress(a, dbCon)
				if err != nil {
					dbg.E(TAG, "Error creating google address : ", err)
					return
				}
			}
			q += "INSERT INTO GoogleContacts_Addresses (googleContactId,googleAddressId) VALUES(?,?);"
			vals = append(vals, contact.Id, a.Id)

		}
	}
	if q != "" {
		_, err = dbCon.Exec(q, vals...)
		if err != nil {
			dbg.E(TAG, "Error creating group/address contact mapping with query %s : ", q, err)
		}
	}
	contact.Id = S.NInt64(key)
	return
}

// CreateGoogleAddress creates a new GoogleAddress in the database.
func CreateGoogleAddress(address *GoogleAddress, dbCon *sql.DB) (err error) {
	vals := []interface{}{address.FormattedAddress}
	valString := "?"

	insFields := "formattedAddress"

	if address.Rel != "" {
		insFields += ",rel"
		valString += ",?"
		vals = append(vals, address.Rel)
	}
	if address.Sync != nil {
		insFields += ",syncId"
		valString += ",?"
		vals = append(vals, address.Sync.Id)
	}
	if address.Contact != nil {
		insFields += ",contactId"
		valString += ",?"
		vals = append(vals, address.Contact.Id)
	}
	if address.RetryTime != 0 {
		insFields += ",retryTime"
		valString += ",?"
		vals = append(vals, address.RetryTime)
	}
	if address.TripType != 0 {
		insFields += ",tripType"
		valString += ",?"
		vals = append(vals, address.TripType)
	}
	if address.TryCount != 0 {
		insFields += ",tryCount"
		valString += ",?"
		vals = append(vals, address.TryCount)
	}

	q := "INSERT INTO GoogleAddresses(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for CreateGoogleAddress: %v ", err)
		return
	}
	var key int64
	key, err = res.LastInsertId()
	if err != nil {
		dbg.E(TAG, "Error getting LastInsertId for new address : ", err)
		return
	}
	address.Id = S.NInt64(key)
	return
}

// UpdateGoogleAddress updates the given GoogleAddress
func UpdateGoogleAddress(address *GoogleAddress, dbCon *sql.DB) (rowCount int64, err error) {
	vals := []interface{}{}
	firstVal := true
	valString := ""
	if address.Rel != "" {
		AppendNStringUpdateField("rel", &address.Rel, &firstVal, &vals, &valString)
	}
	if address.Sync != nil {
		AppendInt64UpdateField("syncId", &address.Sync.Id, &firstVal, &vals, &valString)
	}
	if address.Contact != nil {
		AppendNInt64UpdateField("contactId", &address.Contact.Id, &firstVal, &vals, &valString)
	}
	if address.TripType != 0 {
		AppendNInt64UpdateField("tripType", &address.TripType, &firstVal, &vals, &valString)
	}
	if address.RetryTime != 0 {
		AppendNInt64UpdateField("retryTime", &address.RetryTime, &firstVal, &vals, &valString)
	}
	if address.TryCount != 0 {
		AppendNInt64UpdateField("tryCount", &address.TryCount, &firstVal, &vals, &valString)
	}
	if firstVal {
		err = ErrNoChanges
		return
	}
	q := "UPDATE GoogleAddresses SET " + valString + " WHERE _googleAddressId=?"
	vals = append(vals, address.Id)
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for UpdateAddress: %v ", err)
		return
	}
	rowCount, err = res.RowsAffected()
	if err != nil {
		dbg.E(TAG, "Error in UpdateAddress RowsAffected", err)
		return
	}
	return
}

// UpdateGoogleGroup updates the given GoogleGroup
func UpdateGoogleGroup(group *GoogleGroup, dbCon *sql.DB) (rowCount int64, err error) {
	vals := []interface{}{}
	firstVal := true
	valString := ""
	if group.Name != "" {
		AppendNStringUpdateField("name", &group.Name, &firstVal, &vals, &valString)
	}
	if group.Sync != nil {
		AppendInt64UpdateField("syncId", &group.Sync.Id, &firstVal, &vals, &valString)
	}
	if group.TripType != 0 {
		AppendNInt64UpdateField("tripType", &group.TripType, &firstVal, &vals, &valString)
	}
	if firstVal {
		err = ErrNoChanges
		return
	}
	q := "UPDATE GoogleGroups SET " + valString + " WHERE _googleGroupId=?"
	vals = append(vals, group.Id)
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for UpdateGroup: %v ", err)
		return
	}
	rowCount, err = res.RowsAffected()
	if err != nil {
		dbg.E(TAG, "Error in UpdateGroup RowsAffected", err)
		return
	}
	return
}

// UpdateGoogleContact updates the given GoogleContact.
func UpdateGoogleContact(contact *GoogleContact, dbCon *sql.DB) (rowCount int64, err error) {
	vals := []interface{}{}
	firstVal := true
	valString := ""
	prevContact, err := GetGoogleContactById(int64(contact.Id), dbCon)
	if err != nil {
		dbg.E(TAG, "Error getting prevContact with id %d : ", contact.Id, err)
		return -1, err
	}
	if contact.Name != "" {
		AppendNStringUpdateField("name", &contact.Name, &firstVal, &vals, &valString)
	}
	if contact.Addresses != nil {
		q := ""
		vals2 := make([]interface{}, 0)

		for _, a := range contact.Addresses {
			var pa *GoogleAddress
			if a.Id > 0 {
				_, err = UpdateGoogleAddress(a, dbCon)
				if err != nil && err != ErrNoChanges {
					dbg.E(TAG, "Error updating google address : ", err)
					return -1, err
				}
				for _, oa := range prevContact.Addresses {
					if int64(oa.Id) == int64(a.Id) {
						pa = oa
						break
					}
				}
			}
			if pa == nil {
				if a.Id < 1 {
					err = CreateGoogleAddress(a, dbCon)
					if err != nil {
						dbg.E(TAG, "Error creating google address : ", err)
						return -1, err
					}
				}
				firstVal = false
				q += "INSERT INTO GoogleContacts_Addresses (googleContactId,googleAddressId) VALUES(?,?);"
				vals2 = append(vals2, contact.Id, a.Id)
			}
		}
		for _, oa := range prevContact.Addresses {
			found := false
			for _, a := range contact.Addresses {
				if int64(oa.Id) == int64(a.Id) {
					found = true
					break
				}
			}
			if !found {
				firstVal = false
				q += "DELETE FROM GoogleContacts_Addresses WHERE googleContactId=? AND googleAddressId=?;"
				vals2 = append(vals2, contact.Id, oa.Id)
			}
		}
		if q != "" {
			_, err = dbCon.Exec(q, vals2...)
			if err != nil {
				dbg.E(TAG, "Error updating GoogleContacts_Addresses : ", err)
				return -1, err
			}
		}
	}
	if contact.Sync != nil {
		AppendInt64UpdateField("syncId", &contact.Sync.Id, &firstVal, &vals, &valString)
	}
	if contact.LastUpdate != 0 {
		AppendNInt64UpdateField("lastUpdate", &contact.LastUpdate, &firstVal, &vals, &valString)
	}

	if contact.Groups != nil && len(contact.Groups) != 0 {

		q := ""
		vals2 := make([]interface{}, 0)

		for _, g := range contact.Groups {
			found := false
			for _, og := range prevContact.Groups {
				if int64(og.Id) == int64(g.Id) {
					found = true
					break
				}
			}
			if !found {
				firstVal = false
				q += "INSERT INTO GoogleContacts_Groups (googleContactId,googleGroupId) VALUES(?,?);"
				vals2 = append(vals2, contact.Id, g.Id)
			}
		}
		for _, og := range prevContact.Groups {
			found := false
			for _, g := range contact.Groups {
				if int64(og.Id) == int64(g.Id) {
					found = true
					break
				}
			}
			if !found {
				firstVal = false
				q += "DELETE FROM GoogleContacts_Groups WHERE googleContactId=? AND googleGroupId=?;"
				vals2 = append(vals2, contact.Id, og.Id)
			}
		}
		if q != "" {
			_, err = dbCon.Exec(q, vals2...)
			if err != nil {
				dbg.E(TAG, "Error updating GoogleContacts_Groups : ", err)
				return -1, err
			}
		}
	}
	if firstVal {
		err = ErrNoChanges
		return
	}
	q := "UPDATE GoogleContacts SET " + valString + " WHERE _googleContactId=?"
	vals = append(vals, contact.Id)
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for UpdateContact: %v ", err)

		return
	}
	rowCount, err = res.RowsAffected()
	if err != nil {
		dbg.E(TAG, "Error in UpdateContact RowsAffected", err)
		return
	}

	return
}

// GetGoogleContactById returns the GoogleContact with the given ID.
func GetGoogleContactById(id int64, dbCon *sql.DB) (contact *GoogleContact, err error) {
	res, err := GetGoogleContactsByWhere("_googleContactId=?", dbCon, id)
	if err != nil {
		dbg.E(TAG, "Error in GetGoogleContactById : ", err)
		return
	}
	if len(res) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(res) > 1 {
		dbg.WTF(TAG, "How can we have multiple contacts with same id?", len(res))
		return nil, errors.New("Multiple contacts with same id!")
	}
	contact = res[0]
	return
}

// GetGoogleGroupById returns the GoogleGroup with the given ID.
func GetGoogleGroupById(id int64, dbCon *sql.DB) (group *GoogleGroup, err error) {
	res, err := GetGoogleGroupsByWhere("_googleGroupId=?", dbCon, id)
	if err != nil {
		dbg.E(TAG, "Error in GetGoogleGroupById : ", err)
		return
	}
	if len(res) != 1 {
		return nil, sql.ErrNoRows
	}
	group = res[0]
	return
}
