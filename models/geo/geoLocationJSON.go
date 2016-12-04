package models


type IGeoGeometry interface{}

// GeoObject represents a GeoJSON-object.
type GeoObject struct {
	Type       string                 `json:"type"`
	Crs        string                 `json:"crs,,omitempty"`
	Bbox       []float64              `json:"bbox,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// GeoCoord represents coordinates for a GeoJSON-object.
type GeoCoord [2]float64

// GeoFeature represents a GeoJSON-feature.
type GeoFeature struct {
	GeoObject `json`
	Id        interface{}  `json:"id,omitempty"`
	Geometry  IGeoGeometry `json:"geometry"`
}

// GeoFeatureCollection represents a FeatureCollection for GeoJSON.
type GeoFeatureCollection struct {
	GeoObject
	Features []*GeoFeature `json:"features"`
}

// GeoRoot is the root of a GeoJSON-object.
type GeoRoot struct {
	FeatureCollections map[string]*GeoFeatureCollection
}

// GeoGeometryCollection represents a GeoJSON-GeometryCollection
type GeoGeometryCollection struct {
	GeoObject
	Geometries []GeoObject
}

// NewGeoRoot initialises an empty GeoJSON-Root
func NewGeoRoot() *GeoRoot {
	return &GeoRoot{
		FeatureCollections: make(map[string]*GeoFeatureCollection),
	}
}

// AppendGeometry appends a Geometry to a GeoJSON-GeoFeature-Collection.
func AppendGeometry(fs []*GeoFeature, gs ...IGeoGeometry) []*GeoFeature {
	for _, g := range gs {
		f := NewGeoFeature()
		f.Geometry = g
		fs = append(fs, f)
	}
	return fs
}

// GeoGeometry represents a GeoJSON-Geometry.
type GeoGeometry struct {
	GeoObject
	Coordinates []interface{} `json:"coordinates"`
}

// GeoPointGeometry represents a GeoJSON-GeoPointGeometry.
type GeoPointGeometry struct {
	GeoGeometry
	Coordinates GeoCoord `json:"coordinates"`
}

// GeoMultiPointGeometry represents a GeoJSON-MultiPointGeometry
type GeoMultiPointGeometry struct {
	GeoGeometry
	Coordinates []GeoCoord `json:"coordinates"`
}

// GeoMultiLineGeometry represents a GeoJSON-MultiLineGeometry
type GeoMultiLineGeometry struct {
	GeoGeometry
	Coordinates [][]GeoCoord `json:"coordinates"`
}

// NewGeoFeatureCollection initialises an empty GeoJSON-FeatureCollection.
func NewGeoFeatureCollection() *GeoFeatureCollection {
	g := GeoFeatureCollection{
		Features: make([]*GeoFeature, 0),
	}
	g.Type = "FeatureCollection"
	return &g
}

// NewGeoFeature initialises a new GeoJSON-Feature.
func NewGeoFeature() *GeoFeature {
	g := GeoFeature{}
	g.Properties = make(map[string]interface{})
	g.Type = "Feature"
	return &g
}

// NewGeoGeometryCollection initialises a new GeoJSON-GeometryCollection
func NewGeoGeometryCollection() *GeoGeometryCollection {
	g := GeoGeometryCollection{}
	g.Type = "GeometryCollection"
	g.Geometries = make([]GeoObject, 0)
	return &g
}
// NewGeoCoord initialises a new GeoJSON-coordinate
func NewGeoCoord(lng float64, lat float64) GeoCoord {
	return GeoCoord([2]float64{lng, lat})
}

// NewGeoPointGeometry initialises a new GeoJSON-PointGeometry
func NewGeoPointGeometry() *GeoPointGeometry {
	g := GeoPointGeometry{}
	g.Type = "Point"
	return &g
}

// NewGeoMultiPointGeometry initialises a new GeoJSON-MultiPointGeometry
func NewGeoMultiPointGeometry() *GeoMultiPointGeometry {
	g := GeoMultiPointGeometry{
		Coordinates: make([]GeoCoord, 0),
	}
	g.Type = "MultiPoint"
	return &g
}

// NewGeoLineStringGeometry  initialises a new GeoJSON-MultiPointGeometry as LineString
func NewGeoLineStringGeometry() *GeoMultiPointGeometry {
	g := GeoMultiPointGeometry{
		Coordinates: make([]GeoCoord, 0),
	}
	g.Type = "LineString"
	return &g
}

// NewGeoMultiLineStringGeometry  initialises a new GeoJSON-MultiLineGeometry as MultiLineString
func NewGeoMultiLineStringGeometry() *GeoMultiLineGeometry {
	g := GeoMultiLineGeometry{
		Coordinates: make([][]GeoCoord, 0),
	}
	g.Type = "MultiLineString"
	return &g
}

// NewGeoCircle creates one GeoPointGeometry with the property radius set to the radius. Needs to be handled
// by the presenters pointToLayer or something.
func NewGeoCircle(radius float32) *GeoPointGeometry {
	g := NewGeoPointGeometry()
	g.Properties = make(map[string]interface{})
	g.Properties["radius"] = radius
	return g
}

// NewGeoMultiCircle creates one GeoMultiPointGeometry with the property radius set to the radius. Needs to be handled
// by the presenters pointToLayer or something.
func NewGeoMultiCircle(radius float32) *GeoMultiPointGeometry {
	g := NewGeoMultiPointGeometry()
	g.Properties = make(map[string]interface{})
	g.Properties["radius"] = radius
	return g
}

// tripTypes represents a mapping of int-representations of tripTypes to their string-representation
var tripTypes = map[int]string{
	1: "business",
	2: "workway",
	3: "private",
}
