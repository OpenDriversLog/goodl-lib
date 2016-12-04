-- +migrate Up
CREATE TABLE IF NOT EXISTS `devices` (
	id	INTEGER,
	desc	STRING NOT NULL,
	PRIMARY KEY(id)
);

CREATE TABLE IF NOT EXISTS `TrackRecords` (
	_id	INTEGER,
	deviceId	INTEGER NOT NULL,
	timeMillis	INTEGER,
	latitude	DOUBLE,
	longitude	DOUBLE,
	altitude	DOUBLE,
	accuracy	FLOAT,
	provider	STRING,
	source	INTEGER,
	accuracyRating	INTEGER,
	speed	FLOAT,
	PRIMARY KEY(_id),
    FOREIGN KEY (deviceId) REFERENCES devices(id)
);

CREATE INDEX IF NOT EXISTS IDX_TR_TimeMillis ON TrackRecords(timeMillis);
CREATE INDEX IF NOT EXISTS IDX_TR_DeviceKey ON TrackRecords(deviceId);

-- +migrate Down
DROP INDEX IDX_TR_TimeMillis; -- SQLite specific
DROP INDEX IDX_TR_DeviceKey; -- SQLite specific
DROP TABLE devices;
DROP TABLE TrackRecords;
