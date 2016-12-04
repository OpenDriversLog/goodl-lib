// manages DBconnections, validation & migrations for trackrecords
package helpers

import (
	"errors"

	"database/sql"

	"github.com/OpenDriversLog/goodl-lib/models/SQLite"
)

// USAGE:
// create new object via NewUpdateHelper
// add them fields w/ Apppend<Type>Field()
// execute update via ExecUpdate(tablename string, where-selector [ending in =? or IN (?)], whereVal)

const TAG = "glib/dbMan/helpers/helpers.go"

var ErrNoChanges = errors.New("No columns to update")
var ErrNoTableName = errors.New("No table name given for update")

// UpdateHelper provides functions for more easy update-queries with parameters.
type UpdateHelper struct {
	values    []interface{}
	firstVal  bool
	valString string
	dbCon     *sql.DB
}

// NewUpdateHelper initializes a new UpdateHelper
func NewUpdateHelper(dbCon *sql.DB) (u *UpdateHelper) {
	return &UpdateHelper{
		values:    []interface{}{},
		firstVal:  true,
		valString: "",
		dbCon:     dbCon,
	}
}

// ExecUpdate executes the update, whereSelector should be like columnName=? or columnName IN (?)
func (u *UpdateHelper) ExecUpdate(tableName string, whereSelector string, whereVals ...interface{}) (res sql.Result, err error) {
	// TODO: make it error robust, add =? or IN (?) if not given in whereSelector argument

	if u.firstVal {
		err = ErrNoChanges
		return
	}
	if tableName == "" {
		err = ErrNoTableName
		return
	}

	q := "UPDATE " + tableName + " SET " + u.valString + " WHERE " + whereSelector
	u.values = append(u.values, whereVals...)
	// dbg.W(TAG, "update to exec", q, u.values)
	res, err = u.dbCon.Exec(q, u.values...)

	return
}

// AppendString appends a string to the update
func (u *UpdateHelper) AppendString(fieldName string, fieldVal *string) {
	if *fieldVal == "-" {
		*fieldVal = ""
	}
	u.append2Fields(fieldName, fieldVal)
}

// AppendInt appends an int to the update
func (u UpdateHelper) AppendInt(fieldName string, fieldVal *int) {
	if *fieldVal == -1337 {
		u.append2Fields(fieldName, nil)
	} else if *fieldVal == -1 {
		u.append2Fields(fieldName, 0)
	} else {
		u.append2Fields(fieldName, fieldVal)
	}
}

// AppendInt64 appends an int64 to the update
func (u *UpdateHelper) AppendInt64(fieldName string, fieldVal *int64) {
	if *fieldVal == -1337 {
		u.append2Fields(fieldName, nil)
	} else if *fieldVal == -1 {
		u.append2Fields(fieldName, 0)
	} else {
		u.append2Fields(fieldName, fieldVal)
	}
}

// AppendNInt64 appends an NInt64 to the update
func (u *UpdateHelper) AppendNInt64(fieldName string, fieldVal *models.NInt64) {
	v := int64(*fieldVal)
	u.AppendInt64(fieldName, &v)
}

// AppendNFloat appends a NFloat64 to the update.
func (u *UpdateHelper) AppendNFloat(fieldName string, fieldVal *models.NFloat64) {
	v := float64(*fieldVal)
	u.AppendFloat(fieldName, &v)
}

// AppendNString appends a NString to the update.
func (u *UpdateHelper) AppendNString(fieldName string, fieldVal *models.NString) {
	v := string(*fieldVal)
	u.AppendString(fieldName, &v)
}

// AppendFloat appends a float64 to the update.
func (u *UpdateHelper) AppendFloat(fieldName string, fieldVal *float64) {
	if *fieldVal == -1337.0 {
		u.append2Fields(fieldName, nil)
	} else if *fieldVal == -1.0 {
		u.append2Fields(fieldName, 0)
	} else {
		u.append2Fields(fieldName, fieldVal)
	}
}

// append2Vields appends an undefined type (interface{}) to the update.
func (u *UpdateHelper) append2Fields(fieldName string, fieldVal interface{}) {
	if u.firstVal {
		u.firstVal = false
	} else {
		u.valString += ","
	}
	u.valString += fieldName + "=?"
	u.values = append(u.values, fieldVal)
}
