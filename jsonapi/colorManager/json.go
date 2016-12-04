package colorManager

import (
	"database/sql"
	"encoding/json"
	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/models"
	. "github.com/OpenDriversLog/goodl-lib/models/SQLite"
	. "github.com/OpenDriversLog/goodl-lib/tools"
)

type JSONColorsAnswer struct {
	models.JSONAnswer
	Colors []*Color
}

// JSONGetColors returns all colors
func JSONGetColors(dbCon *sql.DB) (res JSONColorsAnswer, err error) {
	res = JSONColorsAnswer{}
	res.Colors, err = GetColors(dbCon)
	if err != nil {
		dbg.E(TAG, "Error getting GetColors : ", err)
		err = nil
		res = GetBadJsonColorsManAnswer("Unknown error while getting colors")
		return
	}
	res.Success = true
	return
}

// GetBadJsonColorsManAnswer returns a bad JSONColorsAnswer if an error occured.
func GetBadJsonColorsManAnswer(message string) JSONColorsAnswer {
	return JSONColorsAnswer{
		JSONAnswer: models.GetBadJSONAnswer(message),
	}
}

// JSONCreateColor creates a new color from the given JSON-string
func JSONCreateColor(colorJson string, dbCon *sql.DB) (res models.JSONInsertAnswer, err error) {
	c := &Color{}
	if colorJson == "" {
		res = models.GetBadJSONInsertAnswer(NoDataGiven)
		return
	}
	err = json.Unmarshal([]byte(colorJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in JSONCreateColor : ", colorJson, err)
		res = models.GetBadJSONInsertAnswer("Invalid format")
		err = nil
		return
	}
	var key int64
	key, err = CreateColor(c, dbCon)
	if err != nil {
		dbg.E(TAG, "Error in JSONCreateColor CreateColor: ", err)
		err = nil
		res = models.GetBadJSONInsertAnswer("Internal server error")
		return
	}
	res.LastKey = key
	c.Id = NInt64(res.LastKey)
	res.Success = true
	return

}

// JSONDeleteColor deletes the color given in the JSON-string
func JSONDeleteColor(colorJson string, dbCon *sql.DB) (res models.JSONDeleteAnswer, err error) {
	c := &Color{}
	if colorJson == "" {
		res = models.GetBadJSONDeleteAnswer(NoDataGiven, -1)
		return
	}
	err = json.Unmarshal([]byte(colorJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in DeleteColorJSON : ", colorJson, err)
		res = models.GetBadJSONDeleteAnswer("Invalid format", -1)
		err = nil
		return
	}
	var rowCount int64
	rowCount, err = DeleteColor(int64(c.Id), dbCon)
	if err != nil {
		dbg.E(TAG, "Error in DeleteColorJSON DeleteColor: ", err)

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

// JSONGetEmptyColor returns JSONSelectAnswer with empty color-object
func JSONGetEmptyColor() (res models.JSONSelectAnswer, err error) {
	emptyColor, err := GetEmptyColor()
	if err != nil {
		dbg.E(TAG, "Error in GetEmptyContact: ", err)
	}
	res = models.GetGoodJSONSelectAnswer(emptyColor)
	return
}

// JSONUpdateColor updates the given color.
func JSONUpdateColor(colorJson string, dbCon *sql.DB) (res models.JSONUpdateAnswer, err error) {
	c := &Color{}
	if colorJson == "" {
		res = models.GetBadJSONUpdateAnswer(NoDataGiven, -1)
		return
	}
	err = json.Unmarshal([]byte(colorJson), c)
	if err != nil {
		dbg.W(TAG, "Could not read JSON %v in UpdateColorJSON : ", colorJson, err)
		res = models.GetBadJSONUpdateAnswer("Invalid format", -1)
		err = nil
		return
	}
	var rowCount int64
	rowCount, err = UpdateColor(c, dbCon)
	if err != nil {
		if err == ErrNoChanges {
			err = nil
			res = models.GetBadJSONUpdateAnswer(NoDataGiven, int64(c.Id))
			return
		}
		dbg.E(TAG, "Error in JSONUpdateColor UpdateColor: ", err)
		err = nil
		res = models.GetBadJSONUpdateAnswer("Internal server error", int64(c.Id))
		return
	}
	res.RowCount = rowCount
	res.Id = int64(c.Id)
	res.Success = true
	return

}
