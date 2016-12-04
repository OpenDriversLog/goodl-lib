-- +migrate Up
CREATE INDEX IF NOT EXISTS IDX_KeyPoints_LatLng ON KeyPoints(latitude,longitude);
CREATE INDEX IF NOT EXISTS IDX_GeoFenceRegions_LatLng ON GeoFenceRegions(OuterMaxLat,OuterMaxLon,OuterMinLat,OuterMinLon);