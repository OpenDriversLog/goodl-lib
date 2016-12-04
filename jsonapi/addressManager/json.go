package addressManager

import (
	"database/sql"

	"encoding/json"

	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/models"
	. "github.com/OpenDriversLog/goodl-lib/tools"
)

// JSONGetContactAddressCollection returns a collection of all contacts (if getAllContacts true) and all addresses (if getAllAddresses true)
func JSONGetContactAddressCollection(getAllContacts bool, getAllAddresses bool, dbCon *sql.DB) (res JSONAddressManAnswer, err error) {
	res = JSONAddressManAnswer{}
	if getAllContacts {
		res.Contacts, err = GetContactsWithGeoZones("", dbCon)
		if err != nil {
			dbg.E(TAG, "Error getting GetContactsWithGeoZones : ", err)
			err = nil
			res = GetBadJsonAddressManAnswer("Unknown error while getting contacts")
			return
		}
	}

	if getAllAddresses {
		res.Addresses, err = GetAddressesWithGeoFences("", dbCon)
		if err != nil {
			dbg.E(TAG, "Error getting GetAddressesWithGeoFences : ", err)
			err = nil
			res = GetBadJsonAddressManAnswer("Unknown error while getting addresses")
			return
		}
	}

	res.Success = true
	return
}

// GetBadJsonAddressManAnswer returns a bad JSONAddressManAnswer if an error has occured.
func GetBadJsonAddressManAnswer(message string) JSONAddressManAnswer {
	return JSONAddressManAnswer{
		JSONAnswer: models.GetBadJSONAnswer(message),
	}
}

// JSONCreateContact creates a contact by the given Json-String
func JSONCreateContact(contactJson string, dbCon *sql.DB) (res JSONInsertAddressManAnswer, err error) {
	c := &Contact{}
	if contactJson == "" {
		dbg.W(TAG, "No contact json given in JSONCreateContact")
		res = GetBadJSONAddressManInsertAnswer(NoDataGiven)
		return
	}
	err = json.Unmarshal([]byte(contactJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in JSONCreateContact : ", contactJson, err)
		res = GetBadJSONAddressManInsertAnswer("Invalid format")
		err = nil
		return
	}
	var key int64
	key, err = CreateContact(c, dbCon)
	if err != nil {
		dbg.E(TAG, "Error in JSONCreateContact CreateContact: ", err)
		err = nil
		res = GetBadJSONAddressManInsertAnswer("Internal server error")
		return
	}
	res.LastKey = key
	var cs []*Contact
	cs, err = GetContactsWithGeoZones("_contactId=?", dbCon, res.LastKey)
	if err != nil {
		dbg.E(TAG, "Error getting new contact!")
		err = nil
		res = GetBadJSONAddressManInsertAnswer("Internal server error")
		return
	}
	if len(cs) != 1 {
		dbg.E(TAG, "Error getting new contact - got %d contacts!", len(cs))
		err = nil
		res = GetBadJSONAddressManInsertAnswer("Internal server error")
		return
	}
	c = cs[0]
	if c.Address != nil && c.Address.GeoZones != nil && len(c.Address.GeoZones) != 0 {
		res.GeoZones = c.Address.GeoZones
		res.MatchingTripsForEndContact, err = GetTripIdsWithContact(int64(c.Id), true, dbCon)
		if err != nil {
			dbg.E(TAG, "Error in JSONCreateContact GetTripIdsWithContact: ", err)
			err = nil
			res = GetBadJSONAddressManInsertAnswer("Internal server error")
			return
		}
		res.MatchingTripsForStartContact, err = GetTripIdsWithContact(int64(c.Id), false, dbCon)
		if err != nil {
			dbg.E(TAG, "Error in JSONCreateContact GetTripIdsWithContact: ", err)
			err = nil
			res = GetBadJSONAddressManInsertAnswer("Internal server error")
			return
		}
	}
	res.Success = true
	return

}

// GetBadJSONAddressManInsertAnswer returns a bad JSONInsertAddressManAnswer if an error has occured.
func GetBadJSONAddressManInsertAnswer(msg string) JSONInsertAddressManAnswer {
	return JSONInsertAddressManAnswer{
		JSONInsertAnswer: models.GetBadJSONInsertAnswer(msg),
	}
}

// JSONDeleteContact deletes the given contact.
func JSONDeleteContact(contactJson string, dbCon *sql.DB) (res models.JSONDeleteAnswer, err error) {
	c := &Contact{}
	if contactJson == "" {
		res = models.GetBadJSONDeleteAnswer(NoDataGiven, -1)
		return
	}
	err = json.Unmarshal([]byte(contactJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in DeleteContactJSON : ", contactJson, err)
		res = models.GetBadJSONDeleteAnswer("Invalid format", -1)
		err = nil
		return
	}
	var rowCount int64
	rowCount, err = DeleteContact(int64(c.Id), dbCon)
	if err != nil {
		dbg.E(TAG, "Error in DeleteContactJSJON DeleteContact: ", err)
		err = nil
		res = models.GetBadJSONDeleteAnswer("Internal server error", int64(c.Id))
		return
	}
	res.RowCount = rowCount
	res.Id = int64(c.Id)
	res.Success = true
	return

}

const NoDataGiven = "Please fill at least one entry."

// JSONSelectContact gets a contact, INCLUDING its Address and GeoZones
func JSONSelectContact(contactJSON string, dbCon *sql.DB) (res models.JSONSelectAnswer, err error) {
	filter := &Contact{}
	var c *Contact
	if contactJSON == "" {
		res = models.GetBadJSONSelectAnswer(NoDataGiven)
		return
	}
	err = json.Unmarshal([]byte(contactJSON), &filter)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in JSONSelectContact : ", contactJSON, err)
		res = models.GetBadJSONSelectAnswer("Invalid format")
		err = nil
		return
	}
	c, err = GetContactWithGeoZone(int64(filter.Id), dbCon)
	if err != nil {
		dbg.E(TAG, "Error in GetContact JSONSelectContact: ", err)
		if err == sql.ErrNoRows {
			res = models.GetBadJSONSelectAnswer("Not found")
		} else {
			res = models.GetBadJSONSelectAnswer("Internal server error")
		}
		err = nil
		return
	}
	res = models.GetGoodJSONSelectAnswer(c)
	return
}

// JSONGetEmptyContact returns JSONSelectAnswer with empty Contact Object
func JSONGetEmptyContact() (res models.JSONSelectAnswer, err error) {
	emptyContact, err := GetEmptyContact()
	if err != nil {
		dbg.E(TAG, "Error in GetEmptyContact: ", err)
	}
	res = models.GetGoodJSONSelectAnswer(emptyContact)
	return
}

// JSONUpdateContact updates the given contact
func JSONUpdateContact(contactJson string, dbCon *sql.DB) (res JSONUpdateAddressManAnswer, err error) {
	c := &Contact{}
	if contactJson == "" {
		res = JSONUpdateAddressManAnswer{JSONUpdateAnswer: models.GetBadJSONUpdateAnswer(NoDataGiven, -1)}
		return
	}
	err = json.Unmarshal([]byte(contactJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in JSONUpdateContact : ", contactJson, err)
		res = JSONUpdateAddressManAnswer{JSONUpdateAnswer: models.GetBadJSONUpdateAnswer("Invalid format", -1)}
		err = nil
		return
	}
	initialId := c.Id
	var rowCount int64
	var oldUpdatedContact *Contact
	rowCount,oldUpdatedContact, err = UpdateContact(c, dbCon)
	if err != nil {
		if err == ErrNoChanges {
			err = nil
			res = JSONUpdateAddressManAnswer{JSONUpdateAnswer: models.GetBadJSONUpdateAnswer(NoDataGiven, int64(c.Id))}
			return
		}
		dbg.E(TAG, "Error in JSONUpdateContact UpdateContact: ", err)
		err = nil
		res = JSONUpdateAddressManAnswer{JSONUpdateAnswer: models.GetBadJSONUpdateAnswer("Internal server error", -1)}
		return
	}
	res.RowCount = rowCount
	res.Id = int64(c.Id)
	if c.Address != nil && c.Address.GeoZones != nil && len(c.Address.GeoZones) != 0 {
		res.GeoZones = c.Address.GeoZones
		res.MatchingTripsForEndContact, err = GetTripIdsWithContact(int64(c.Id), true, dbCon)
		if initialId != c.Id {
			res.NewContact = c
			res.Id = int64(oldUpdatedContact.Id)
			res.UpdatedDto = oldUpdatedContact;
		}
		if err != nil {
			dbg.E(TAG, "Error in JSONUpdateContact GetTripIdsWithContact: ", err)
			err = nil
			res = JSONUpdateAddressManAnswer{JSONUpdateAnswer: models.GetBadJSONUpdateAnswer("Internal server error", int64(c.Id))}
			return
		}
		res.MatchingTripsForStartContact, err = GetTripIdsWithContact(int64(c.Id), false, dbCon)
		if err != nil {
			dbg.E(TAG, "Error in JSONUpdateContact GetTripIdsWithContact: ", err)
			err = nil
			res = JSONUpdateAddressManAnswer{JSONUpdateAnswer: models.GetBadJSONUpdateAnswer("Internal server error", int64(c.Id))}
			return
		}
	}
	res.Success = true
	return

}
