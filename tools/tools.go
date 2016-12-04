// Package tools provides several helper-functions used in goodl & goodl-lib.
package tools

import (
	"archive/zip"
	"crypto/rand"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"database/sql"
	"fmt"
	"time"

	"github.com/Compufreak345/dbg"
	"github.com/mattn/go-sqlite3"
	S "github.com/OpenDriversLog/goodl-lib/models/SQLite"
)

const tTag = dbg.Tag("goodl-lib/tools.go")

var ErrNoChanges = errors.New("No columns to update")

// ReadAllDirContents lists all files in a given directory, prepending "prefix" if it is a file.
func ReadAllDirContents(prefix string, dirPath string) (allFiles []string) {

	allFiles = make([]string, 0)
	dir, _ := ioutil.ReadDir(dirPath)
	for _, f := range dir {
		if f.IsDir() {
			allFiles = append(allFiles, ReadAllDirContents(prefix+f.Name()+"/", dirPath+"/"+f.Name())...)
		} else {
			allFiles = append(allFiles, prefix+f.Name())
		}
	}
	return
}

// interface Scannable represents an item that can be scanned into pointers, e.g. an SQL-row.
type Scannable interface {
	Scan(...interface{}) error
}

// SQLIgnoreField is used to skip a field while scanning a SQL-row.
type SQLIgnoreField struct {
}

// Scan implements the Scanner interface.
func (*SQLIgnoreField) Scan(interface{}) error {
	return nil
}

// SkippyScanRow is used to scan a *sql.Row(s) while skipping some fields (e.g. to be scanned in another method)
func SkippyScanRow(row Scannable, fieldsBefore int, fieldsAfter int, data ...interface{}) (err error) {
	dataObjects := make([]interface{}, fieldsBefore+len(data)+fieldsAfter)

	mCount := len(data)
	for i := 0; i < fieldsBefore; i++ {
		dataObjects[i] = &SQLIgnoreField{}
	}
	for i := 0; i < mCount; i++ {
		dataObjects[i+fieldsBefore] = data[i]
	}
	for i := 0; i < fieldsAfter; i++ {
		dataObjects[i+mCount+fieldsBefore] = &SQLIgnoreField{}
	}
	err = row.Scan(dataObjects...)

	if err != nil {
		dbg.E(tTag, "SkippyScanRow produced an error %s \n with args fieldBefore: %s , fieldsAfter: %s , data : %s ", err, fieldsBefore, fieldsAfter, data)
	}
	return
}

// Unzip unzips a file. http://stackoverflow.com/a/24792688
func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetCleanFilePath Checks if a file is in the given dir or a subdir of the given dir. Should prevent injection like "../../etc/profile"
// returns a definitely in the directory filePath and an error if it was not succesful
// modified version of http://golang.org/src/net/http/fs.go?s=719:734#L23 (just skipping opening the file)
func GetCleanFilePath(dirRelativeFPath string, dirPath string) (secureFPath string, err error) {

	if filepath.Separator != '/' && strings.IndexRune(dirRelativeFPath, filepath.Separator) >= 0 ||
		strings.Contains(dirRelativeFPath, "\x00") {
		return "", errors.New("http: invalid character in file path")
	}
	dir := string(dirPath)
	if dir == "" {
		dir = "."
	}
	secureFPath = filepath.Join(dirPath, filepath.FromSlash(path.Clean("/"+dirRelativeFPath)))
	return
}

// GetDateForFileName gets a date string to be used in a file name, e.g. "2006_01_02_15_04_05"
func GetDateForFileName(timeMillis int64, timeConfig *TimeConfig) string {
	time := GetTimeFromMillis(timeMillis)
	return time.Format(timeConfig.FileTimeFormatString)
}

// GetDateOnlyForText gets a german string-representation of a data, e.g. "02.01.2006 15:04" or "15:04", depending on the "withDate"-parameter.
func GetDateOnlyForText(timeMillis int64, withDate bool) string {
	//TODO Check TimeZone foo
	t := GetTimeFromMillis(timeMillis)
	if withDate {
		return t.Format("02.01.2006 15:04")
	} else {
		return t.Format("15:04")
	}

}

// GetDateOnlyForText_NoTime gets a german string-representation of the given timestamp, without time, e.g. "02.01.2006"
func GetDateOnlyForText_NoTime(timeMillis int64) string {
	//TODO Check TimeZone foo
	t := GetTimeFromMillis(timeMillis)
	return t.Format("02.01.2006")


}

// GetTimeFromMillis converts a Millisecond-Unix-Timestamp to a golang time.Time-object.
func GetTimeFromMillis(timeMillis int64) time.Time {
	return time.Unix(0, timeMillis*1000*1000)
}

// CopyFile is a save way to copy files, as of http://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang
// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. If that fail, copy the file contents from src to dst.
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

const saltSize = 16

// GenerateSalt generates a random salt. https://www.socketloop.com/tutorials/golang-securing-password-with-salt
func GenerateSalt() []byte {
	buf := make([]byte, saltSize, saltSize)
	_, err := io.ReadFull(rand.Reader, buf)

	if err != nil {
		dbg.E(tTag, "random read failed: %v", err)
		panic(err)
	}

	return buf
}

// RegisterSqlite registers sqlite3 under name, if it isnt already
func RegisterSqlite(name string) (err error) {
	need_register := true
	for _, b := range sql.Drivers() {
		if b == name {
			need_register = false
		}
	}
	if need_register {
		sql.Register(name, &sqlite3.SQLiteDriver{})
	}
	return
}

// TimeConfig defines how to format time for user output.
type TimeConfig struct {

	// Long time format, contains month and year
	LongTimeFormatString string
	// Short time format, contains only time
	ShortTimeFormatString string

	// Time format for file names
	FileTimeFormatString string
	TimeLocation         *time.Location
}

// GetDefaultTimeConfig returns the default (german) time configuration.
func GetDefaultTimeConfig() *TimeConfig {
	return &TimeConfig{
		LongTimeFormatString:  "02.01.2006 15:04",
		ShortTimeFormatString: "15:04",
		FileTimeFormatString:  "2006_01_02_15_04_05",
		TimeLocation:          time.UTC,
	}
}

// AppendStringUpdateField appends a string-field to an update-query
func AppendStringUpdateField(fieldName string, fieldVal *string, firstVal *bool, vals *[]interface{}, uFields *string) {
	if *fieldVal == "-" {
		*fieldVal = ""
	}
	Append2UpdateFields(fieldName, *fieldVal, firstVal, vals, uFields)
}

// AppendNStringUpdateField appends a NString-field to an update-query
func AppendNStringUpdateField(fieldName string, fieldVal *S.NString, firstVal *bool, vals *[]interface{}, uFields *string) {
	if *fieldVal == "-" {
		*fieldVal = ""
	}
	Append2UpdateFields(fieldName, fieldVal, firstVal, vals, uFields)
}

// AppendIntUpdateField appends a int-field to an update-query
func AppendIntUpdateField(fieldName string, fieldVal *int, firstVal *bool, vals *[]interface{}, uFields *string) {
	if *fieldVal == -1337 {
		Append2UpdateFields(fieldName, nil, firstVal, vals, uFields)
	} else if *fieldVal == -1 {
		Append2UpdateFields(fieldName, 0, firstVal, vals, uFields)
	} else {
		Append2UpdateFields(fieldName, fieldVal, firstVal, vals, uFields)
	}

}

// AppendInt64UpdateField appends a int64-field to an update-query
func AppendInt64UpdateField(fieldName string, fieldVal *int64, firstVal *bool, vals *[]interface{}, uFields *string) {
	if *fieldVal == -1337 {
		Append2UpdateFields(fieldName, nil, firstVal, vals, uFields)
	} else if *fieldVal == -1 {
		Append2UpdateFields(fieldName, 0, firstVal, vals, uFields)
	} else {
		Append2UpdateFields(fieldName, fieldVal, firstVal, vals, uFields)
	}

}

// AppendNInt64UpdateField appends a NInt64-field to an update-query
func AppendNInt64UpdateField(fieldName string, fieldVal *S.NInt64, firstVal *bool, vals *[]interface{}, uFields *string) {
	if *fieldVal == -1337 {
		Append2UpdateFields(fieldName, nil, firstVal, vals, uFields)
	} else if *fieldVal == -1 {
		Append2UpdateFields(fieldName, 0, firstVal, vals, uFields)
	} else {
		Append2UpdateFields(fieldName, fieldVal, firstVal, vals, uFields)
	}

}

// AppendFloatUpdateField appends a float64-field to an update-query
func AppendFloatUpdateField(fieldName string, fieldVal *float64, firstVal *bool, vals *[]interface{}, uFields *string) {
	if *fieldVal == -1337.0 {
		Append2UpdateFields(fieldName, nil, firstVal, vals, uFields)
	} else if *fieldVal == -1.0 {
		Append2UpdateFields(fieldName, 0, firstVal, vals, uFields)
	} else {
		Append2UpdateFields(fieldName, fieldVal, firstVal, vals, uFields)
	}

}

// Append2UpdateFields appends a field of an undefined type (interface{}) to an update-query
func Append2UpdateFields(fieldName string, fieldVal interface{}, firstVal *bool, vals *[]interface{}, uFields *string) {
	if *firstVal {
		*firstVal = false
	} else {
		*uFields += ","
	}
	*uFields += fieldName + "=?"
	*vals = append(*vals, fieldVal)
}

// AppendStringsByComma appends val2 to val1, if val 1 and val 2 not empty adds comma.
func AppendStringsByComma(val1 string, val2 string) string {
	if val2 == "" {
		return val1
	}
	if val1 != "" {
		val1 += ","
	} else {
		return val2
	}
	val1 += val2
	return val1
}
