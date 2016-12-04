package syncMan

import (
	"database/sql"
	"encoding/json"
	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/models"
	S "github.com/OpenDriversLog/goodl-lib/models/SQLite"
	. "github.com/OpenDriversLog/goodl-lib/tools"
)

// GetBadJSONSyncManAnswer gets a bad JSONSyncManAnswer when an error occured
func GetBadJSONSyncManAnswer(msg string) JSONSyncManAnswer {
	return JSONSyncManAnswer{
		JSONAnswer: models.GetBadJSONAnswer(msg),
	}
}

// GetBadJSONTokenSyncManAnswer gets a bad JSONTokenSyncManAnswer when an error occured
func GetBadJSONTokenSyncManAnswer(msg string) JSONTokenSyncManAnswer {
	return JSONTokenSyncManAnswer{
		JSONAnswer: models.GetBadJSONAnswer(msg),
	}
}

// GetBadJSONRefreshSyncManAnswer gets a bad JSONRefreshSyncManAnswer when an error occured
func GetBadJSONRefreshSyncManAnswer(msg string) JSONRefreshSyncManAnswer {
	return JSONRefreshSyncManAnswer{
		JSONAnswer: models.GetBadJSONAnswer(msg),
	}
}

// JSONAutoRefresh starts automatic synchronisation update
func JSONAutoRefresh(clientSecretFilecontent []byte,uId int64, dbCon *sql.DB) (res JSONRefreshSyncManAnswer, err error) {
	err = AutoRefresh(clientSecretFilecontent,uId, dbCon)
	if err != nil {
		dbg.E(TAG, "Error calling refresh : ", err)
		res = GetBadJSONRefreshSyncManAnswer("Internal server error")
		err = nil
		return
	}
	res.Success = true
	return
}

// JSONRefreshSync refresh the given Sync
func JSONRefreshSync(syncJson string, clientSecretFilecontent []byte,uId int64, dbCon *sql.DB) (res JSONRefreshSyncManAnswer, err error) {

	sync := &Sync{}
	if syncJson == "" {
		res = GetBadJSONRefreshSyncManAnswer(NoDataGiven)
		return
	}
	err = json.Unmarshal([]byte(syncJson), sync)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in JSONCreateSync : ", syncJson, err)
		res = GetBadJSONRefreshSyncManAnswer("Invalid format")
		err = nil
		return
	}
	_, _, err = RefreshSync(clientSecretFilecontent, sync.Id,uId, dbCon)
	if err != nil {
		dbg.E(TAG, "Error in JSONRefreshSyncRefreshSync: ", err)
		err = nil
		res = GetBadJSONRefreshSyncManAnswer("Internal server error")
		return
	}

	res.Success = true
	return
	/*updatedContacts, err := Refresh(clientSecretFilecontent, dbCon)
	if err != nil {
		dbg.E(TAG, "Error calling refresh : ", err)
		res = GetBadJSONRefreshSyncManAnswer("Internal server error")
		err = nil
		return
	}
	res.UpdatedContacts = updatedContacts
	res.Success = true*/
	//return
}

// JSONGetSyncs returns all setup synchronisations.
func JSONGetSyncs(dbCon *sql.DB) (res JSONSyncManAnswer, err error) {
	res.Syncs, err = GetSyncs(dbCon)
	if err != nil {
		dbg.E(TAG, "Error getting GetSyncs : ", err)
		err = nil
		res = GetBadJSONSyncManAnswer("Unknown error while getting syncs")
		return
	}
	res.Success = true
	return
}

// JSONCreateRefreshToken Creates a RefreshToken from the given authCode.
func JSONCreateRefreshToken(clientSecretFilecontent []byte, authCode string, dbCon *sql.DB) (res JSONTokenSyncManAnswer, err error) {
	auth, err := CreateGoogleRefreshToken(authCode, clientSecretFilecontent, dbCon)
	if err != nil {
		dbg.E(TAG, "Error creating refresh token : ", err)
		res = GetBadJSONTokenSyncManAnswer("Internal server error")
		err = nil
		return
	} else {
		res.Success = true
		res.Id = int64(auth.Id)
		res.Type = "Google"
	}
	return
}

// JSONCreateSync creates a new sync.
func JSONCreateSync(syncJson string, oAuthId int64, dbCon *sql.DB) (res models.JSONInsertAnswer, err error) {
	sync := &Sync{}
	if syncJson == "" {
		res = models.GetBadJSONInsertAnswer(NoDataGiven)
		return
	}
	err = json.Unmarshal([]byte(syncJson), sync)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in JSONCreateSync : ", syncJson, err)
		res = models.GetBadJSONInsertAnswer("Invalid format")
		err = nil
		return
	}
	if oAuthId > 0 {
		sync.OAuth = &OAuth{Id: S.NInt64(oAuthId)}
	}
	var key int64
	key, err = CreateSync(sync, dbCon)
	if err != nil {
		dbg.E(TAG, "Error in JSONCreateSync CreateSync: ", err)
		err = nil
		res = models.GetBadJSONInsertAnswer("Internal server error")
		return
	}
	res.LastKey = key
	sync.Id = res.LastKey
	res.Success = true
	return

}

// JSONDeleteSync deletes the given sync by its ID.
func JSONDeleteSync(syncJson string, clientSecretFilecontent []byte, dbCon *sql.DB) (res models.JSONDeleteAnswer, err error) {
	c := &Sync{}
	if syncJson == "" {
		res = models.GetBadJSONDeleteAnswer(NoDataGiven, -1)
		return
	}
	err = json.Unmarshal([]byte(syncJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in DeleteSyncJSON : ", syncJson, err)
		res = models.GetBadJSONDeleteAnswer("Invalid format", -1)
		err = nil
		return
	}
	var rowCount int64
	rowCount, err = DeleteSync(int64(c.Id), clientSecretFilecontent, dbCon)
	if err != nil {
		dbg.E(TAG, "Error in DeleteSyncJSON DeleteSync: ", err)

		errMsg := "Internal server error"
		err = nil
		res = models.GetBadJSONDeleteAnswer(errMsg, int64(c.Id))
		return
	}
	res.RowCount = rowCount
	res.Id = int64(c.Id)
	res.Success = true
	return

}

const NoDataGiven = "Please fill at least one entry."

// JSONGetEmptySync returns JSONSelectAnswer with empty sync-object
func JSONGetEmptySync() (res models.JSONSelectAnswer, err error) {
	emptySync, err := GetEmptySync()
	if err != nil {
		dbg.E(TAG, "Error in GetEmptyContact: ", err)
	}
	res = models.GetGoodJSONSelectAnswer(emptySync)
	return
}

// JSONUpdateSync updates the given sync.
func JSONUpdateSync(syncJson string, oAuthId int64, dbCon *sql.DB) (res models.JSONUpdateAnswer, err error) {
	c := &Sync{}
	if syncJson == "" {
		res = models.GetBadJSONUpdateAnswer(NoDataGiven, -1)
		return
	}
	err = json.Unmarshal([]byte(syncJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in UpdateSyncJSON : ", syncJson, err)
		res = models.GetBadJSONUpdateAnswer("Invalid format", -1)
		err = nil
		return
	}
	if oAuthId > 0 {
		c.OAuth = &OAuth{Id: S.NInt64(oAuthId)}
	}
	var rowCount int64
	rowCount, err = UpdateSync(c, dbCon)
	if err != nil {
		if err == ErrNoChanges {
			err = nil
			res = models.GetBadJSONUpdateAnswer(NoDataGiven, int64(c.Id))
			return
		}
		dbg.E(TAG, "Error in JSONUpdateSync UpdateSync: ", err)
		err = nil
		res = models.GetBadJSONUpdateAnswer("Internal server error", int64(c.Id))
		return
	}
	res.RowCount = rowCount
	res.Id = int64(c.Id)
	res.Success = true
	return

}
