-- +migrate Up

CREATE TABLE IF NOT EXISTS KeyPoints_GeoFenceRegions (
keyPointId INTEGER,
geoFenceRegionId INTEGER,
UNIQUE (keyPointId, geoFenceRegionId)
);

DELETE FROM KeyPoints_GeoFenceRegions;
INSERT OR IGNORE INTO KeyPoints_GeoFenceRegions(keyPointId,geoFenceRegionId)


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
-- The following is necessary to only include regions with existing contacts
-- we could remove this if we get performance issues :
LEFT JOIN Address_GeoFenceRegion AG ON G._geoFenceRegionId = AG.geoFenceRegionId
LEFT JOIN Addresses AA ON AG.addressID=AA._addressId
WHERE K._keyPointId IS NOT NULL AND AA._addressID IS NOT NULL;
CREATE VIEW IF NOT EXISTS KeyPoint_GeoFenceRegion_Contact AS
SELECT K.*,C.*,A.* FROM KeyPoints K LEFT JOIN
 KeyPoints_GeoFenceRegions KG ON KG.keyPointId = K._keyPointId
		 LEFT JOIN Address_GeoFenceRegion AG ON KG.geoFenceRegionId=AG.geoFenceRegionId
		LEFT JOIN Addresses A ON AG.addressId=A._addressId
		LEFT JOIN Contacts C ON C.addressId = A._addressId
		WHERE  C.addressId IS NOT NULL;
CREATE VIEW IF NOT EXISTS NoKeyPoint_GeoFenceRegion_Contact AS
        SELECT KG.*,C.*,A.* FROM KeyPoints_GeoFenceRegions KG
        		 LEFT JOIN Address_GeoFenceRegion AG ON KG.geoFenceRegionId=AG.geoFenceRegionId
        		LEFT JOIN Addresses A ON AG.addressId=A._addressId
        		LEFT JOIN Contacts C ON C.addressId = A._addressId
        		WHERE  C.addressId IS NOT NULL;

-- +migrate Down
