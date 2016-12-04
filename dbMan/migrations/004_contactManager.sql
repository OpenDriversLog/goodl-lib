-- +migrate Up

ALTER TABLE Addresses RENAME TO Addresses_Old;
CREATE TABLE Addresses(
    _addressId INTEGER PRIMARY KEY,
    street TEXT NOT NULL DEFAULT '',
    postal TEXT NOT NULL DEFAULT '',
    city TEXT NOT NULL DEFAULT '',
    additional1 TEXT NOT NULL DEFAULT '',
    additional2 TEXT NOT NULL DEFAULT '',
    latitude DOUBLE NOT NULL DEFAULT 0,
    longitude DOUBLE NOT NULL DEFAULT 0,
    HouseNumber TEXT NOT NULL DEFAULT ''
);
INSERT INTO Addresses (_addressId,street,postal,city,additional1,additional2,latitude,longitude)
SELECT _addressId,street,postal,city,'','',
coalesce((SELECT latitude FROM KeyPoints WHERE KeyPoints.addressId=Addresses_Old._addressId LIMIT 1),0),
coalesce((SELECT longitude FROM KeyPoints WHERE KeyPoints.addressId=Addresses_Old._addressId LIMIT 1),0)
FROM Addresses_Old;
DROP TABLE Addresses_Old;

ALTER TABLE KeyPoints RENAME TO KeyPoints_Old;
CREATE TABLE IF NOT EXISTS `KeyPoints` (
	_keyPointId INTEGER PRIMARY KEY,
    latitude DOUBLE NOT NULL,
    longitude DOUBLE NOT NULL,
	startTime INTEGER NOT NULL,
	endTime INTEGER NOT NULL,
	previousTrackId INTEGER,
	nextTrackId INTEGER,
	deviceId INTEGER NOT NULL,
    addressId INTEGER,
    FOREIGN KEY (previousTrackId) REFERENCES Tracks(_trackId),
    FOREIGN KEY (nextTrackId) REFERENCES Tracks(_trackId),
    FOREIGN KEY (deviceId) REFERENCES DEVICES(id),
    FOREIGN KEY (addressId) REFERENCES Addresses(_addressId)
);
INSERT INTO KeyPoints(_keyPointId,latitude,longitude,startTime,endTime,previousTrackId,nextTrackId,deviceId,addressId)
SELECT _keyPointId,latitude,longitude,startTime,endTime,previousTrackId,nextTrackId,deviceId,addressId
FROM KeyPoints_Old;
DROP TABLE KeyPoints_Old;

CREATE TABLE IF NOT EXISTS Rectangles(
    _rectangleId INTEGER PRIMARY KEY ,
    topLeftLat DOUBLE NOT NULL,
    topLeftLon DOUBLE NOT NULL,
    botRightLat DOUBLE NOT NULL,
    botRightLon DOUBLE NOT NULL
);

CREATE TABLE  IF NOT EXISTS Circles(
    _circleId INTEGER PRIMARY KEY ,
    radius DOUBLE NOT NULL,
    centerLatitude DOUBLE NOT NULL,
    centerLongitude DOUBLE NOT NULL
);

DROP TABLE GeoFenceRegions;
CREATE TABLE GeoFenceRegions(
    _geoFenceRegionId INTEGER PRIMARY KEY,
    outerMinLat DOUBLE  NOT NULL,
    outerMinLon DOUBLE NOT NULL,
    outerMaxLat DOUBLE NOT NULL,
    outerMaxLon DOUBLE NOT NULL,
    rectangleId INTEGER,
    circleId INTEGER,
    FOREIGN KEY(rectangleId) REFERENCES Rectangles(_rectangleId),
    FOREIGN KEY(circleId) REFERENCES Circles(_circleId)
);

CREATE TABLE  IF NOT EXISTS Address_GeoFenceRegion (
    addressId INTEGER NOT NULL,
    geoFenceRegionId INTEGER NOT NULL,
    FOREIGN KEY(addressId) REFERENCES Addresses(_addressId),
    FOREIGN KEY(geoFenceRegionId) REFERENCES geoFenceRegion(_geoFenceRegionId)
);

CREATE TABLE  IF NOT EXISTS TripTypes (
    _tripTypeId INTEGER PRIMARY KEY AUTOINCREMENT,
    description STRING NOT NULL,
    viewCategory INTEGER NOT NULL
);
INSERT INTO TripTypes(description, viewCategory) VALUES('Privat',2);
INSERT INTO TripTypes(description, viewCategory) VALUES('Arbeitsweg',1);
INSERT INTO TripTypes(description, viewCategory) VALUES('Dienstlich',0);

CREATE TABLE PointOfInterestTypes(
    _pointOfInterestTypeId INTEGER PRIMARY KEY AUTOINCREMENT,
    description STRING NOT NULL,
    tripTypeId INTEGER,
    FOREIGN KEY(tripTypeId) REFERENCES TripTypes(_tripTypeId)
);


INSERT INTO PointOfInterestTypes(description,tripTypeId) VALUES('Supermarkt',1);
INSERT INTO PointOfInterestTypes(description,tripTypeId) VALUES('Baumarkt',1);


DROP TABLE Contacts;
CREATE TABLE  Contacts (
    _contactId INTEGER PRIMARY KEY AUTOINCREMENT,
    type INTEGER  NOT NULL DEFAULT 1,
    title STRING DEFAULT "",
    description STRING DEFAULT "",
    additional STRING DEFAULT "",
    addressId INTEGER,
    tripTypeId INTEGER,
    FOREIGN KEY(addressId) REFERENCES Addresses(_addressId),
    FOREIGN KEY(tripTypeId) REFERENCES TripTypes(_tripTypeId)
);

DROP TABLE pointsOfInterest;
CREATE TABLE  IF NOT EXISTS PointsOfInterest(
    _pointOfInterestId INTEGER PRIMARY KEY,
    description TEXT DEFAULT "",
    pointOfInterestType INTEGER,
    tripTypeId INTEGER NOT NULL
);

CREATE VIEW AddressesWithGeoZones AS
SELECT CASE WHEN _geoFenceRegionId IS NOT NULL THEN 1 ELSE 0 END AS hasGeoFenceRegion, _addressId,street,postal,city,additional1,additional2,
latitude,longitude, _geoFenceRegionId,outerMinLat,outerMinLon,outerMaxLat,outerMaxLon,rectangleId,
Rectangles.topLeftLat,Rectangles.topLeftLon,Rectangles.botRightLat,Rectangles.botRightLon FROM Addresses LEFT JOIN Address_GeoFenceRegion ON _addressId=addressId LEFT JOIN GeoFenceRegions on
	geoFenceRegionId = _geoFenceRegionId LEFT JOIN Rectangles ON rectangleId=_rectangleId;



-- +migrate Down
