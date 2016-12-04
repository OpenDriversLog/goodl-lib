// Package models contains several models used by multiple classes.
package models

import (
	geo "github.com/kellydunn/golang-geo"
	"github.com/OpenDriversLog/goodl-lib/models/SQLite"
)

// LocationConfig is the configuration definition for calculating tracks.
type LocationConfig struct {
	// MinMoveDist configures the radius (in meters) to consider as "same position".
	// Is used for determining wether a route stopped or started, together with MinMoveTime
	MinMoveDist int

	// MinMoveTime configures the amount of time (in millseconds) that the user needs to
	// stay at the same position to consider as route end or start point.
	MinMoveTime int

	// AccuraryThreshold configures the Accuracy where we stop worrying about the point
	AccuracyThreshold int

	// minimal distances for points to display at the given zoom stage
	// http://wiki.openstreetmap.org/wiki/DE:Zoom_levels
	// TODO: implement usable config... how to set a const size?
	// map[int]int
}

// GeoAnalyzeData is currently not used :>
type GeoAnalyzeData struct {
	GeoRoot   *GeoRoot
	Devices   map[string]int
	MaxLat    float64
	MinLat    float64
	MaxLng    float64
	MinLng    float64
	StartTime int64
	EndTime   int64
}

// EnhancedMapData is currently not used :>
type EnhancedMapData struct {
	*GeoAnalyzeData
	MinTime      int64
	MaxTime      int64
	KeyPointData []*KeyPointInfo
}

// EGeoPoint extends golang-geo.Point with Accuracy and Timestamp
type EGeoPoint struct {
	*geo.Point
	Accuracy float64
	Time     int64
}

// KeyPointInfo represents a Geo-Point with address and a timespan
type KeyPointInfo struct {
	KeyPointId         int64
	Lat                float64
	Lng                float64
	Street             string
	Postal             string
	City               string
	HouseNumber			string
	MinTime            int64
	MaxTime            int64
	MatchingContactids models.NString
	GeoCoder string
}

// Track represents a track from a StartKeyPoint to an EndKeyPoint
type Track struct {
	TrackId           int64 `json:"id"`
	DeviceId          int64 `json:"deviceId"`
	StartTime         int64
	EndTime           int64
	StartKeyPointId   int64
	EndKeyPointId     int64
	StartKeyPointInfo *KeyPointInfo
	EndKeyPointInfo   *KeyPointInfo
	Distance          float64
}

// TrackPointJson represents a Trackpoint for JSON
type TrackPointJson struct {
	TrackId    int64   `json:"trackId"`
	TimeMillis int64   `json:"time"`
	Speed      float64 `json:"speed"`
	Accuracy   float64 `json:"accuracy"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	ZoomLevel  int64   `json:"zoomlevel"`
}
