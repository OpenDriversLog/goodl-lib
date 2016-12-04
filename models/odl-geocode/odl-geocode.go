package odl_geocode
/*
Models for odl-geocoder - make sure to keep at sync with odl-geocoder/models.go
Having this redundancy because importing odl-geocoder in goodl-lib is quite annoying.
 */

// GeoResp represents a GeoCoder-response
type GeoResp struct {
	ReqId                string
	Lat                  float64
	Lng                  float64
	Address              Address
	MaxRequestsPerDay    int
	MaxRequestsPerUser   int
	CurDailyRequestsUsed int
	CurUserRequestsUsed  int
	Error                string
	Provider string
}

// Address represents the address given in a GeoCoder-Response.
type Address struct {
	Lat         float64
	Lng         float64
	Street      string
	Postal      string
	City        string
	Additional1 string
	Additional2 string
	HouseNumber string
	Title       string
	Fuel        string
	Accuracy    string
	Country     string
}