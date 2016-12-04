package models
// DataTypes for better working with SQLite.
import (
	"database/sql"

	"database/sql/driver"
)

// NInt64 represents a nullable int64 (null is represented by a Value of 0)
type NInt64 int64

// Scan implements the Scanner interface.
func (n *NInt64) Scan(val interface{}) (err error) {
	nn := sql.NullInt64{}
	err = nn.Scan(val)
	if !nn.Valid {
		*n = 0
	} else {
		*n = NInt64(nn.Int64)
	}
	return
}

// Value implements driver.Valuer interface
func (n NInt64) Value() (v driver.Value, err error) {
	return int64(n), nil
}

// NFloat64 represents a nullable float64 (null is represented by a Value of 0)
type NFloat64 float64

// Scan implements the Scanner interface.
func (n *NFloat64) Scan(val interface{}) (err error) {
	nn := sql.NullFloat64{}
	err = nn.Scan(val)
	if !nn.Valid {
		*n = 0
	} else {
		*n = NFloat64(nn.Float64)
	}
	return
}

// Value implements driver.Valuer interface
func (n NFloat64) Value() (v driver.Value, err error) {
	return float64(n), nil
}

// NString represents a nullable string (null is represented by a Value of "")
type NString string

// Scan implements the Scanner interface!
func (n *NString) Scan(val interface{}) (err error) {

	nn := sql.NullString{}
	err = nn.Scan(val)
	if !nn.Valid {
		*n = ""
	} else {
		*n = NString(nn.String)
	}
	return
}

// Value implements driver.Valuer interface

func (n NString) Value() (v driver.Value, err error) {
	return string(n), nil
}

// Location represents a trackRecords-table-entry without device
type Location struct {
	Id             sql.NullInt64
	TimeMillis     sql.NullInt64
	Latitude       sql.NullFloat64
	Longitude      sql.NullFloat64
	Altitude       sql.NullFloat64
	Accuracy       sql.NullFloat64
	Provider       sql.NullString
	Source         sql.NullInt64
	AccuracyRating sql.NullInt64
	Speed          sql.NullFloat64
}

// ServerLocation represents a trackRecords-table-entry with device
type ServerLocation struct {
	Location
	DeviceKey sql.NullString
}

// TrackPoint represents a TrackPoints-Table-entry.
type TrackPoint struct {
	TrackPointId sql.NullInt64
	TrackId      sql.NullInt64
	TimeMillis   sql.NullInt64
	Latitude     sql.NullFloat64
	Longitude    sql.NullFloat64
	Accuracy     sql.NullFloat64
	Speed        sql.NullFloat64
	MinZoomLevel sql.NullInt64
	MaxZoomLevel sql.NullInt64
}

// KeyPoint represents a KeyPoints-Table-entry.
type KeyPoint struct {
	KeyPointId        sql.NullInt64
	Latitude          sql.NullFloat64
	Longitude         sql.NullFloat64
	StartTime         sql.NullInt64
	EndTime           sql.NullInt64
	PreviousTrackId   sql.NullInt64
	NextTrackId       sql.NullInt64
	DeviceId          sql.NullInt64
	AddressId         sql.NullInt64
	PointOfInterestId sql.NullInt64
}
