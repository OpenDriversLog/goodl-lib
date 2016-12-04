// Package colorManager is responsible for CRUD colors.
package colorManager

import (
	"database/sql"
	"errors"
	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/dbMan/helpers"
)

const TAG = "goodl-lib/jsonApi/colorManager"

// GetColors returns all colors
func GetColors(dbCon *sql.DB) (colors []*Color, err error) {
	colors = make([]*Color, 0)
	q := "SELECT _colorId, color1, color2, color3 FROM Colors"

	res, err := dbCon.Query(q)
	if err != nil {
		if err != nil {
			dbg.E(TAG, "unable to get Colors", err)
			return
		}
	}

	for res.Next() {
		color := &Color{}
		err = res.Scan(&color.Id, &color.Color1, &color.Color2, &color.Color3)
		if err != nil {
			dbg.E(TAG, "Unable to scan color!", err)
			return
		}
		colors = append(colors, color)
	}
	return
}

// GetColor returns color with given id
func GetColor(id int64, dbCon *sql.DB) (color *Color, err error) {
	color = &Color{}
	q := "SELECT _colorId, color1, color2, color3 FROM Colors WHERE _colorId=?"

	err = dbCon.QueryRow(q, id).Scan(&color.Id, &color.Color1, &color.Color2, &color.Color3)
	if err != nil {
		if err != nil {
			dbg.E(TAG, "unable to get Color by ID %d : ", id, err)
			return
		}
	}

	return
}

// CreateColor creates the given color.
func CreateColor(color *Color, dbCon *sql.DB) (key int64, err error) {

	vals := []interface{}{color.Color1}
	valString := "?"

	insFields := "color1"
	if color.Color2 != "" {
		insFields += ",color2"
		valString += ",?"
		vals = append(vals, color.Color2)
	}
	if color.Color3 != "" {
		insFields += ",color3"
		valString += ",?"
		vals = append(vals, color.Color3)
	}
	q := "INSERT INTO Colors(" + insFields + ") VALUES(" + valString + ")"
	var res sql.Result
	res, err = dbCon.Exec(q, vals...)
	if err != nil {
		dbg.E(TAG, "Error in dbCon.Exec for CreateColors: %v ", err)
		return
	}
	key, err = res.LastInsertId()
	return
}

// UpdateColor updates a color
func UpdateColor(d *Color, dbCon *sql.DB) (rowCount int64, err error) {

	update := helpers.NewUpdateHelper(dbCon)

	if d.Color2 != "" {
		update.AppendNString("color2", &d.Color2)
	}
	if d.Color3 != "" {
		update.AppendNString("color3", &d.Color3)
	}

	// at least update one field to don't get errors ;)
	update.AppendNString("color1", &d.Color1)

	res, err := update.ExecUpdate("Colors", "_colorId=?", d.Id)
	if err != nil {
		dbg.E(TAG, "UpdateColor: error execUpdate", err)
	} else if rowCount, err = res.RowsAffected(); err != nil || rowCount == 0 {
		err = errors.New("did update nothing")
	}
	return
}

// DeleteColor deletes the color with the given ID.
func DeleteColor(id int64, dbCon *sql.DB) (rowCount int64, err error) {
	var res sql.Result

	res, err = dbCon.Exec("DELETE FROM Colors WHERE _colorId=?", id)
	if err != nil {
		dbg.E(TAG, "Error in DeleteColor : ", err)
	} else {
		rowCount, err = res.RowsAffected()
		if err != nil {
			dbg.E(TAG, "Error in DeleteColor get RowsAffected : ", err)
		}
	}

	return
}

// GetEmptyColor returns empty color Object
func GetEmptyColor() (color *Color, err error) {
	color = &Color{}
	return
}
