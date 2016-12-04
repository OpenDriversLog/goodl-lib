-- +migrate Up

-- removing unique constraint speeds it up LIKE HELL

DROP TABLE KeyPoints_GeoFenceRegions;
CREATE TABLE IF NOT EXISTS KeyPoints_GeoFenceRegions (
keyPointId INTEGER,
geoFenceRegionId INTEGER,
FOREIGN KEY (keyPointId) REFERENCES KeyPoints(_keyPointId),
FOREIGN KEY (geoFenceRegionId) REFERENCES GeoFenceRegions(_geoFenceRegionId)
);

DELETE FROM KeyPoints_GeoFenceRegions;
INSERT INTO KeyPoints_GeoFenceRegions(keyPointId,geoFenceRegionId)

SELECT K._keyPointId,G._geoFenceRegionId FROM
GeoFenceRegions G
LEFT JOIN KeyPoints K
ON (
K.latitude>G.OuterMinLat AND
K.latitude<G.OuterMaxLat AND
K.longitude>G.OuterMinLon AND
K.longitude<G.OuterMaxLon
)
LEFT JOIN Addresses A  ON K.addressId=A._addressId
WHERE K._keyPointId IS NOT NULL;