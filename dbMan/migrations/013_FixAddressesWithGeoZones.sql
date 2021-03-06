-- +migrate Up

DROP VIEW IF EXISTS AddressesWithGeoZones;
CREATE VIEW AddressesWithGeoZones AS
SELECT CASE WHEN _geoFenceRegionId IS NOT NULL THEN 1 ELSE 0 END AS hasGeoFenceRegion, _addressId,street,postal,city,additional1,additional2,
HouseNumber,title,latitude,longitude,fuel, _geoFenceRegionId,outerMinLat,outerMinLon,outerMaxLat,outerMaxLon,color,rectangleId,
Rectangles.topLeftLat,Rectangles.topLeftLon,Rectangles.botRightLat,Rectangles.botRightLon FROM Addresses LEFT JOIN Address_GeoFenceRegion ON _addressId=addressId LEFT JOIN GeoFenceRegions on
	geoFenceRegionId = _geoFenceRegionId LEFT JOIN Rectangles ON rectangleId=_rectangleId;
