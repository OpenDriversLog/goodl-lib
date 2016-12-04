package models
/*
DTOs for reading KML-files.
 */
import (
	"encoding/xml"
)

// KML represents the basic structure of a KML-file
type KML struct {
	XMLName  xml.Name `xml:"kml"`
	KMLxmlns string   `xml:"xmlns,attr"`
	//KML      string   `xml:",innerxml"`
	Document KMLDocument
}

// KMLDocument represents the  contents of a KML-file
type KMLDocument struct {
	//XMLName xml.Name `xml:"Document"`
	//DocumentInner string   `xml:",innerxml"`
	Name        string
	P           string       `xml:"p,attr"`
	Open        int          `xml:"open"`
	Description string       `xml:"description"`
	StyleMap    KMLStyleMap  `xml:"StyleMap"`
	Placemark   KMLPlacemark `xml:"Placemark"`
}

// KMLStyleMap represents the StyleMap-node of an KML-file.
type KMLStyleMap struct {
	XMLName xml.Name `xml:"StyleMap"`
}

// KMLPlacemark represents the Placemark-node of an KML-file.
type KMLPlacemark struct {
	XMLName xml.Name `xml:"Placemark"`
	//PlacemarkInner string   `xml:",innerxml"`
	Name        string       `xml:"name"`
	Description string       `xml:"description"`
	GxTrack     []KMLgxTrack `xml:"Track"`
}

// KMLgxTrack represents a track in a KML-file
type KMLgxTrack struct {
	//KMLgxTrackInner string `xml:",innerxml"`
	//XMLName      xml.Name `xml:"gx:Track"`
	AltitudeMode string   `xml:"altitudeMode"`
	When         []string `xml:"when"`
	Coord        []string `xml:"coord"`
}
