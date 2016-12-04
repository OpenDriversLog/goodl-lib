/**
*
* Provides methods for helping running the lib inside an Android- or iOS-application.
* This is more like a proof of concept than anything else.
* Data syncing is not completed,
* so this needs to be tuned quite big until it can be used in production. This is so difficult because I wasn't able to find
* an Go-SQLite-Driver for ARM-architecture.
*
*
 */
package datapolish

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/Compufreak345/dbg"
	"io"
	"strings"
	"sync"
	"time"
)

// These values need to be changed if we compile for Android
// const useLocalDb = false
const useLocalDb = true

// Android : _id , Other : id
// const colId = "_id"
const colId = "_id"

// Android : deviceId, Other : deviceKey <- no more deviceKey after DB upgrade
const colDeviceKey = "deviceId"
const cTag = "datapolish/crossplattform.go"

const isIos = false

var SelectParms string
var UpdateParms string
var SelectQueryRes string
var SelectQuery string
var UpdateQueries string

var mutexWaitingForRes sync.Mutex
var csvRd *csv.Reader
var mutexWaitingCancel sync.Mutex
var Cancel = false
var IsInCancelleableProcess = false

// CancelProcess cancels current running GetPolishedPointsInTimeFrame (as soon as it tries to get something from the database or a query result)
func CancelProcess() {
	mutexWaitingCancel.Lock()
	defer mutexWaitingCancel.Unlock()
	fmt.Println("CancelProcess called.")
	Cancel = true
	for Cancel && IsInCancelleableProcess {
		fmt.Println("Waiting for cancel...")
		// cancelation still in progress
		time.Sleep(9 * time.Millisecond)
	}

}

const cpTag = "glib/t/crossplatform.go"

// OpenDbCon tries to open database connection AND validates
// dont forget to defer in your calling func!
func OpenDbCon(dbPath string) (database *sql.DB, err error) {

	if useLocalDb {
		database, err = sql.Open("SQLITE", dbPath)
		// defer database.Close()
		if err != nil {
			return nil, err
		}
	}

	// TODO: CS use crossplatform database stuffs
	if err != nil {
		dbg.E(cpTag, "Failed to create DB handle at lib/tools/OpenDbCon() : ", err)
		return
	}
	if err = database.Ping(); err != nil {
		dbg.E(cpTag, "Failed to keep connection alive at lib/tools/OpenDbCon() : ", err)
		return
	}

	// dbg.D(cpTag, "Databaseconnection established", dbPath)

	return
}

// getDbLine reads a line of buffered query result - returns io.EOF if end of file
func getDbLine(rows *sql.Rows, args ...interface{}) (err error) {
	defer func() { // Error handling, if this panics (should not happen)
		if errr := recover(); errr != nil {
			err = errors.New(fmt.Sprintf("%v", errr))
			closeSelectQuery()
		}
	}()
	if useLocalDb {
		if !rows.Next() {
			err = io.EOF
			return
		}

		return rows.Scan(args...)
	}
	if Cancel { // interrupt whole process from Android / iOS
		closeSelectQuery()
		return errors.New("We got canceled.")
	}
	if csvRd != nil {
		res, err := csvRd.Read()
		if err != nil {
			closeSelectQuery()
			return err
		}
		resLength := len(res)
		for i, v := range args {
			if i >= resLength {
				closeSelectQuery()
				return csv.ErrFieldCount
			}
			err = convertAssign(v, res[i])
			if err != nil {
				closeSelectQuery()
				return err
			}
		}
	} else {
		closeSelectQuery()
		err = io.EOF
	}
	return
}

// closeSelectQuery finishs a select query by resetting the query variables to their defaults and unlocking the mutexWaitingForRes.
func closeSelectQuery() {
	defer func() { // Error handling, if unlocking unlocked mutex
		if errr := recover(); errr != nil {
			if fmt.Sprintf("%v", errr) == "sync: unlock of unlocked mutex" {
				// Everything OK
			} else {
				panic(errors.New(fmt.Sprintf("%v", errr)))
			}

		}
	}()
	// We finished with this query - make place for the next one
	if !useLocalDb {

		csvRd = nil
		SelectQueryRes = ""
		SelectParms = ""
		SelectQuery = ""
		Cancel = false
		mutexWaitingForRes.Unlock()
	}
}

// func executeSelectQuery executes a select-query
func executeSelectQuery(q string, db *sql.DB, parms ...interface{}) (rows *sql.Rows, err error) {

	if useLocalDb {
		// Website :
		return db.Query(q, parms...)
	} else {
		if isIos {
			// Ios :

			return
		}

		// Android :
		// Block until this query is completely read & finished.
		mutexWaitingForRes.Lock()
		defer func() { // Error handling, if this getviewdata panics (should not happen)
			if errr := recover(); errr != nil {
				err = errors.New(fmt.Sprintf("Panic in executeSelectQuery : %v", errr))
				closeSelectQuery()
			}
		}()
		if len(parms) != 0 {

			var buf bytes.Buffer
			var firstParm = true
			for _, parm := range parms {
				if firstParm {
					firstParm = false
				} else {
					buf.WriteString(",")
				}
				buf.WriteString(fmt.Sprintf("%v", parm))

			}
			SelectParms = buf.String()
			dbg.D(cTag, "Set SelectParms to ", SelectParms)
		}
		csvRd = nil
		SelectQueryRes = "Waiting"
		SelectQuery = q

		var sleepCnt = 0
		// TODO : Put some kind of timeout here
		for SelectQueryRes == "Waiting" { // block until finished - publicodl will watch Query and update and so
			if Cancel { // interrupt whole process from Android / iOS
				closeSelectQuery()
				err = errors.New("We got canceled.")
				return
			}
			sleepCnt++
			time.Sleep(11 * time.Millisecond) // This sleep is necessary, otherwise it blocks every other go call!
			if sleepCnt > 9000 {
				// timeout after 99 seconds.
				closeSelectQuery()
				err = errors.New("Query timed out")
				return
			}
		}
		if !(len(SelectQueryRes) > 6 && SelectQueryRes[:7] == "Error :") {
			// Everything ok. Nice.
			csvRd = csv.NewReader(strings.NewReader(SelectQueryRes))
		} else if SelectQueryRes == "Error : No rows found" {
			csvRd = nil
		} else {
			err = errors.New(SelectQueryRes)
			closeSelectQuery()
		}
	}

	return
}
