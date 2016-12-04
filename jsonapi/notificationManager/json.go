package notificationManager

import (
	"database/sql"
	"encoding/json"
	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/models"
	. "github.com/OpenDriversLog/goodl-lib/tools"
)

type JSONNotificationsAnswer struct {
	models.JSONAnswer
	Notifications []*Notification
}

// JSONGetNotifications returns all currently active notifications.
func JSONGetNotifications(withBasicTripData bool,dbCon *sql.DB) (res JSONNotificationsAnswer, err error) {
	res = JSONNotificationsAnswer{}
	res.Notifications, err = GetActiveNotifications(withBasicTripData,dbCon)
	if err != nil {
		dbg.E(TAG, "Error getting GetNotifications : ", err)
		err = nil
		res = GetBadJsonNotificationsManAnswer("Unknown error while getting notifications")
		return
	}
	res.Success = true
	return
}

// GetBadJsonNotificationsManAnswer returns a bad JSONNotificationsAnswer in case of an error.
func GetBadJsonNotificationsManAnswer(message string) JSONNotificationsAnswer {
	return JSONNotificationsAnswer{
		JSONAnswer: models.GetBadJSONAnswer(message),
	}
}

// JSONCreateNotification creates a new notification
func JSONCreateNotification(notificationJson string, dbCon *sql.DB) (res models.JSONInsertAnswer, err error) {
	c := &Notification{}
	if notificationJson == "" {
		res = models.GetBadJSONInsertAnswer(NoDataGiven)
		return
	}
	err = json.Unmarshal([]byte(notificationJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in JSONCreateNotification : ", notificationJson, err)
		res = models.GetBadJSONInsertAnswer("Invalid format")
		err = nil
		return
	}
	var key int64
	key, err = CreateNotification(c, dbCon)
	if err != nil {
		dbg.E(TAG, "Error in JSONCreateNotification CreateNotification: ", err)
		err = nil
		res = models.GetBadJSONInsertAnswer("Internal server error")
		return
	}
	res.LastKey = key
	c.Id = res.LastKey
	res.Success = true
	return

}

// JSONDeleteNotification deletes the given notification by its ID
func JSONDeleteNotification(notificationJson string, dbCon *sql.DB) (res models.JSONDeleteAnswer, err error) {
	c := &Notification{}
	if notificationJson == "" {
		res = models.GetBadJSONDeleteAnswer(NoDataGiven, -1)
		return
	}
	err = json.Unmarshal([]byte(notificationJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in DeleteNotificationJSON : ", notificationJson, err)
		res = models.GetBadJSONDeleteAnswer("Invalid format", -1)
		err = nil
		return
	}
	var rowCount int64
	rowCount, err = DeleteNotification(int64(c.Id), dbCon)
	if err != nil {
		dbg.E(TAG, "Error in DeleteNotificationJSON DeleteNotification: ", err)
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

//JSONGetEmptyNotification  returns good JSONSelectAnswer with empty notification-object
func JSONGetEmptyNotification() (res models.JSONSelectAnswer, err error) {
	emptyNotification, err := GetEmptyNotification()
	if err != nil {
		dbg.E(TAG, "Error in GetEmptyContact: ", err)
	}
	res = models.GetGoodJSONSelectAnswer(emptyNotification)
	return
}

// JSONUpdateNotification updates the given notification.´
func JSONUpdateNotification(notificationJson string, dbCon *sql.DB) (res models.JSONUpdateAnswer, err error) {
	c := &Notification{}
	if notificationJson == "" {
		res = models.GetBadJSONUpdateAnswer(NoDataGiven, -1)
		return
	}
	err = json.Unmarshal([]byte(notificationJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in UpdateNotificationJSON : ", notificationJson, err)
		res = models.GetBadJSONUpdateAnswer("Invalid format", -1)
		err = nil
		return
	}
	var rowCount int64
	rowCount, err = UpdateNotification(c, dbCon)
	if err != nil {
		if err == ErrNoChanges {
			err = nil
			res = models.GetBadJSONUpdateAnswer(NoDataGiven, int64(c.Id))
			return
		}
		dbg.E(TAG, "Error in JSONUpdateNotification UpdateNotification: ", err)
		err = nil
		res = models.GetBadJSONUpdateAnswer("Internal server error", int64(c.Id))
		return
	}
	res.RowCount = rowCount
	res.Id = int64(c.Id)
	res.Success = true
	return

}