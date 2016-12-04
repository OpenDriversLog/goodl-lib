-- +migrate Up
CREATE TABLE IF NOT EXISTS `Addresses` (
    _addressId INTEGER PRIMARY KEY,
    street STRING,
    postal STRING,
    city STRING,
    additional1 STRING,
    additionsl2 STRING
);

CREATE TABLE IF NOT EXISTS `KeyPoints` (
	_keyPointId INTEGER PRIMARY KEY,
    latitude DOUBLE,
    longitude DOUBLE,
	startTime INTEGER,
	endTime INTEGER,
	previousTrackId INTEGER,
	nextTrackId INTEGER,
	deviceId INTEGER NOT NULL,
    addressId INTEGER,
    pointOfInterestId INTEGER,
    FOREIGN KEY (previousTrackId) REFERENCES tracks(_trackId),
    FOREIGN KEY (nextTrackId) REFERENCES tracks(_trackId),
    FOREIGN KEY (deviceId) REFERENCES devices(id),
    FOREIGN KEY (addressId) REFERENCES addresses(_addressId)
);

CREATE INDEX IF NOT EXISTS IDX_Address_AddressId ON Addresses(_addressId);
CREATE INDEX IF NOT EXISTS IDX_KeyPoints_AddressId ON KeyPoints(addressId);

CREATE INDEX IF NOT EXISTS IDX_KP_PreviousTrackId ON KeyPoints(previousTrackId);
CREATE INDEX IF NOT EXISTS IDX_KP_NextTrackId ON KeyPoints(nextTrackId);

CREATE TABLE IF NOT EXISTS `Tracks` (
    _trackId INTEGER PRIMARY KEY,
    deviceId INTEGER NOT NULL,
    --startTime INTEGER, -- starts@ startKeyPointId.endTime
    --endTime INTEGER, -- ends@ endKeyPointId.startTime
    startKeyPointId INTEGER,
    endKeyPointId INTEGER,
    distance DOUBLE,
    FOREIGN KEY (startKeyPointId) REFERENCES keyPoints(_keyPointId),
    FOREIGN KEY (endKeyPointId) REFERENCES keyPoints(_keyPointId),
    FOREIGN KEY (deviceId) REFERENCES devices(id)
);

CREATE INDEX IF NOT EXISTS IDX_T_StartKeyPointId ON Tracks(startKeyPointId);
CREATE INDEX IF NOT EXISTS IDX_T_EndKeyPointId ON Tracks(endKeyPointId);

CREATE TABLE IF NOT EXISTS `trackPoints` (
    _trackPointId INTEGER PRIMARY KEY,
    trackId INTEGER NOT NULL, -- which track it belongs to
    timeMillis INTEGER,
    latitude DOUBLE,
    longitude DOUBLE,
    accuracy DOUBLE,
    -- source INTEGER, -- tracks depend on devices
    -- accuracyRating INTEGER,
    speed DOUBLE,
    minZoomLevel INTEGER, -- when to display on map
    maxZoomLevel INTEGER,
    FOREIGN KEY (trackId) REFERENCES Tracks(_trackId)
);

CREATE TABLE IF NOT EXISTS `pointsOfInterest` (
	_pointOfInterestId INTEGER PRIMARY KEY,
	latitude INTEGER,
	longitude INTEGER,
	accuracy INTEGER,
	street STRING,
	streetNumber INTEGER,
	postal INTEGER,
    city STRING,
    description STRING
);

CREATE TABLE IF NOT EXISTS `geoFenceRegions` (
    _geoFenceRegionId INTEGER PRIMARY KEY,
    title STRING,
    description STRING,
    geometryType INTEGER,
    geometryId INTEGER,
    outerMinLat DOUBLE,
    outerMinLon DOUBLE,
    outerMaxLat DOUBLE,
    outerMaxLon DOUBLE
);

-- +migrate Down
DROP INDEX IDX_KP_PreviousTrackId;
DROP INDEX IDX_KP_NextTrackId;
DROP INDEX IDX_T_StartKeyPointId;
DROP INDEX IDX_T_EndKeyPointId;
DROP INDEX IDX_Address_AddressId;
DROP INDEX IDX_KeyPoints_AddressId;
DROP TABLE TrackPoints;
DROP TABLE Tracks;
DROP TABLE PointsOfInterest;
DROP TABLE KeyPoints;
DROP TABLE Addresses;
