// Package notificationManager is responsible for creating & getting notifications ("reminders") to be sent by E-Mail or shown in the GUI (TBD).
package notificationManager

import (
	"database/sql"
	"github.com/Compufreak345/dbg"
	. "github.com/OpenDriversLog/goodl-lib/tools"

	tripMan "github.com/OpenDriversLog/goodl-lib/jsonapi/tripMan/models"
	"github.com/OpenDriversLog/goodl-lib/translate"
)

const TAG = "goodl-lib/jsonApi/notificationManager"
const SelectColumns = "_notificationDataId,notificationType,expirationTime,wasSent,wasUiDisabled,subject,message,shortMessage,tripID"

// GetNotificationsByWhere returns all notifications matching the given where-string with the given parameters.
func GetNotificationsByWhere(withBasicTripData bool,where string, dbCon *sql.DB, params ...interface{}) (notifications []*Notification, err error) {
	notifications = make([]*Notification, 0)
	q := "SELECT " + SelectColumns + " FROM NotificationData"

	if(withBasicTripData) {
		q = "SELECT " + SelectColumns + `,_tripId,type,title,desc,driverId,startContactId,
		endContactId,isReturnTrip,contactId,reviewed,timeOverDue FROM NotificationData LEFT JOIN TRIPS on tripId=_tripID`
	}
	if where != "" {
		q += " WHERE " + where
	}

	res, err := dbCon.Query(q,params...)
	if err != nil {
		if err != nil {
			dbg.E(TAG, "unable to get Notifications", err)
			return
		}
	}

	for res.Next() {
		notification := &Notification{Trip: &tripMan.Trip{}}


		err = res.Scan(getScanCols(notification,withBasicTripData)...)
		if err != nil {
			dbg.E(TAG, "Unable to scan notification!", err)
			return
		}
		notifications = append(notifications, notification)
	}
	return
}

// GetActiveNotifications returns all active notifications.
func GetActiveNotifications(withBasicTripData bool,dbCon *sql.DB) (notifications []*Notification, err error) {
	return GetNotificationsByWhere(withBasicTripData,"wasSent=0 OR wasUiDisabled=0",dbCon)
}

// getScanCols  gets the pointers to the columns to scan for a default Notification-query.
func getScanCols(notification *Notification,withBasicTripData bool) ([]interface{}) {
	if notification==nil {
		notification = &Notification{Trip:&tripMan.Trip{}}
	}
	if notification.Trip==nil {
		notification.Trip = &tripMan.Trip{}
	}
	//_notificationDataId,notificationType,expirationTime,wasSent,
	// wasUiDisabled,subject,message,shortMessage,tripID,_tripId,type,title,desc,driverId,startContactId,
	//endContactId,isReturnTrip,contactId,reviewed,timeOverDue FROM NotificationData LEFT JOIN TRIPS on tripId=_tripID
	scanCols := []interface{}{&notification.Id,&notification.Type,&notification.ExpirationTime,&notification.WasSent,&notification.WasUiDisabled,
		&notification.Subject,&notification.Message,&notification.ShortMessage,&notification.TripId}
	if(withBasicTripData) {
		scanCols = append(scanCols,&notification.Trip.Id,&notification.Trip.Type,&notification.Trip.Title,&notification.Trip.Description,
			&notification.Trip.DriverId,&notification.Trip.StartContactId,
			&notification.Trip.EndContactId,&notification.Trip.IsReturnTrip,&notification.Trip.ContactId,
			&notification.Trip.Reviewed,&notification.Trip.TimeOverDue)
	}
	return scanCols
}
// GetNotificationById returns the notification with the given ID
func GetNotificationById(id int64,withBasicTripData bool, dbCon *sql.DB) (notification *Notification, err error) {
	notification = &Notification{Trip: &tripMan.Trip{}}
	q := "SELECT " + SelectColumns + " FROM NotificationData LEFT JOIN Addresses ON _addressId=addressId WHERE _notificationDataId=?"

	err = dbCon.QueryRow(q, id).Scan(getScanCols(notification,withBasicTripData)...)
	if err != nil {
		dbg.E(TAG, "unable to get Notification", err)
		return
	}
	return
}

// CreateNotification creates a new notification.
func CreateNotification(notification *Notification, dbCon *sql.DB) (key int64, err error) {

	vals := []interface{}{notification.Subject}
	valString := "?"

	insFields := "subject"
	if notification.Message != "" {
		insFields += ",message"
		valString += ",?"
		vals = append(vals, notification.Message)
	}
	if notification.ShortMessage != "" {
		insFields += ",shortMessage"
		valString += ",?"
		vals = append(vals, notification.ShortMessage)
	}

	if notification.TripId==0 && notification.Trip != nil {
		notification.TripId = notification.Trip.Id
	}
	if notification.TripId != 0 {
		insFields += ",tripId"
		valString += ",?"
		vals = append(vals, notification.TripId)
	}
	if notification.ExpirationTime != 0 {
		insFields += ",expirationTime"
		valString += ",?"
		vals = append(vals, notification.ExpirationTime)
	}
	if notification.WasSent != 0 {
		insFields += ",wasSent"
		valString += ",?"
		vals = append(vals, notification.WasSent)
	}
	if notification.WasUiDisabled != 0 {
		insFields += ",wasUiDisabled"
		valString += ",?"
		vals = append(vals, notification.WasUiDisabled)
	}
	if notification.Type != 0 {
		insFields += ",notificationType"
		valString += ",?"
		vals = append(vals, notification.Type)
	}

	q := "INSERT INTO NotificationData(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for CreateNotifications: %v ", err)
		return
	}
	key, err = res.LastInsertId()
	return
}

// UpdateNotification updates a notification
func UpdateNotification(c *Notification, dbCon *sql.DB) (rowCount int64, err error) {

	vals := []interface{}{}
	firstVal := true
	valString := ""

	if c.Type != 0 {
		AppendIntUpdateField("notificationType", &c.Type, &firstVal, &vals, &valString)
	}
	if c.WasSent != 0 {
		AppendIntUpdateField("wasSent", &c.WasSent, &firstVal, &vals, &valString)
	}
	if c.WasUiDisabled != 0 {
		AppendIntUpdateField("wasUiDisabled", &c.WasUiDisabled, &firstVal, &vals, &valString)
	}
	if c.ExpirationTime != 0 {
		AppendInt64UpdateField("expirationTime", &c.ExpirationTime, &firstVal, &vals, &valString)
	}
	if c.Subject != "" {
		AppendStringUpdateField("subject", &c.Subject, &firstVal, &vals, &valString)
	}
	if c.Message != "" {
		AppendStringUpdateField("message", &c.Message, &firstVal, &vals, &valString)
	}
	if c.ShortMessage != "" {
		AppendStringUpdateField("shortMessage", &c.ShortMessage, &firstVal, &vals, &valString)
	}
	if c.TripId==0 && c.Trip != nil {
		c.TripId = c.Trip.Id
	}
	if c.TripId != 0 {
		AppendInt64UpdateField("tripId", &c.TripId, &firstVal, &vals, &valString)
	}


	if firstVal {
		err = ErrNoChanges
		return
	}
	q := "UPDATE NotificationData SET " + valString + " WHERE _notificationDataId=?"
	vals = append(vals, c.Id)
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for UpdateNotification: %v ", err)

		return
	}
	rowCount, err = res.RowsAffected()

	return
}

// DeleteNotification deletes the notification with the given ID.
func DeleteNotification(id int64, dbCon *sql.DB) (rowCount int64, err error) {
	var res sql.Result

	res, err = dbCon.Exec("DELETE FROM NotificationData WHERE _notificationDataId=?", id)
	if err != nil {
		dbg.E(TAG, "Error in DeleteNotification : ", err)
	} else {
		rowCount, err = res.RowsAffected()
		if err != nil {
			dbg.E(TAG, "Error in DeleteNotification get RowsAffected : ", err)
		}
	}

	return
}

// GetEmptyNotification returns an empty notification-object
func GetEmptyNotification() (notification *Notification, err error) {
	notification = &Notification{}
	return
}

// CalcNextNotification returns the next notification to be sent, preferably using the buffered "LastNotification")
func CalcNextNotification(LastNotification *Notification,ActiveNotifications *[]*Notification,dbCon *sql.DB) (NextNotification *Notification,err error) {
	if LastNotification != nil && LastNotification.WasSent<1 {
		return LastNotification, nil
	}
	return GetNextNotification(ActiveNotifications,dbCon)
}

// GetNextNotification returns the next notification to be sent
func GetNextNotification(ActiveNotifications *[]*Notification, dbCon *sql.DB) (NextNotification *Notification, err error) {

	if ActiveNotifications == nil {
		*ActiveNotifications,err = GetActiveNotifications(true,dbCon)
		if err != nil {
			dbg.E(TAG,"Error getting notifications : ", err)
			return
		}
	}
	for _,n := range *ActiveNotifications {
		if n.WasSent<1 {
			if NextNotification==nil || NextNotification.ExpirationTime>n.ExpirationTime {
				NextNotification = n
			}
		}
	}
	return
}

// CheckForNotificationUpdate checks if any notifications need to be created, updated or deleted by the changes in UpdatedTrip
func CheckForNotificationUpdate(UpdatedTrip *tripMan.Trip,ActiveNotifications *[]*Notification,T *translate.Translater,dbCon *sql.DB) (notificationsUpdated bool, err error){
	deleteIdxs := make([]int,0)
	var nearestNotification = &Notification{ExpirationTime:0x7FFFFFFFFFFFFFFF, Id:-1}
	nearestNotificationIdx := 0
	if ActiveNotifications == nil {
		var nots []*Notification
		nots,err = GetActiveNotifications(true,dbCon)
		if err != nil {
			dbg.E(TAG,"Error getting notifications : ", err)
			return
		}
		ActiveNotifications = &nots
	}
	for i, n := range *ActiveNotifications {
		if n.Type !=1 {
			continue
		}
		skip := false
		if UpdatedTrip.Id == n.TripId {
			if UpdatedTrip.Reviewed>0 {
				skip = true
				notificationsUpdated = true
				_,err = DeleteNotification(n.Id, dbCon)
				if err != nil {
					dbg.E(TAG,"Error deleting notification %d : ", n.Id, err)
					return
				}
				deleteIdxs = append(deleteIdxs, i)
			} else if UpdatedTrip.TimeOverDue != n.ExpirationTime+3*24*60*60*1000 {
				n.ExpirationTime = UpdatedTrip.TimeOverDue - 3*24*60*60*1000
				_,err = UpdateNotification(n, dbCon)
				if err != nil {
					dbg.E(TAG,"Error updating notification %d : ", n.Id, err)
					return
				}
			}
		}
		if !skip && n.ExpirationTime<nearestNotification.ExpirationTime {
			nearestNotification = n
			nearestNotificationIdx = i
		}
	}
	expTime := UpdatedTrip.TimeOverDue - 3*24*60*60*1000
	if nearestNotification.ExpirationTime > expTime {

		notificationsUpdated = true
		if nearestNotification.Id!=-1 {
			DeleteNotification(nearestNotification.Id, dbCon)
			deleteIdxs = append(deleteIdxs, nearestNotificationIdx)
		}
		newNotification := &Notification{
			ExpirationTime:expTime,
			Type:1,
			TripId: UpdatedTrip.Id,
			Trip: UpdatedTrip,
			Subject : T.T("TripReminderSubject"),
			Message : T.T("TripReminderMessage"),
			ShortMessage: T.T("TripReminderShortMessage"),
		}
		var id int64
		id, err = CreateNotification(newNotification,dbCon)
		if err != nil {
			dbg.E(TAG,"Error creating notification : ", err)
			return
		}
		dbg.WTF(TAG,"New notification exp time : %d", newNotification.ExpirationTime)
		newNotification.Id = id
		*ActiveNotifications = append(*ActiveNotifications,newNotification)
	}
	deleted := 0
	for _,i := range deleteIdxs {
		c := i - deleted
		// https://github.com/golang/go/wiki/SliceTricks
		*ActiveNotifications, (*ActiveNotifications)[len(*ActiveNotifications)-1] = append((*ActiveNotifications)[:c], (*ActiveNotifications)[c+1:]...), nil
		deleted++;
	}

	return
}