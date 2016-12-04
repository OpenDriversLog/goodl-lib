-- +migrate Up
DROP TABLE IF EXISTS Drivers;
DROP TABLE IF EXISTS Colors;
CREATE TABLE Drivers (
_driverId INTEGER PRIMARY KEY AUTOINCREMENT,
priority INTEGER NOT NULL DEFAULT 3,
addressId INTEGER,
name TEXT,
additional TEXT,
FOREIGN KEY (addressId) REFERENCES Addresses(_addressId)
);
DROP TABLE IF EXISTS Cars;
CREATE TABLE Cars (
_carId INTEGER PRIMARY KEY AUTOINCREMENT,
type TEXT,
ownerId INTEGER,
plate TEXT,
firstMileage INTEGER DEFAULT 0,
mileage INTEGER  DEFAULT 0,
firstUseDate INTEGER DEFAULT 0,
FOREIGN KEY (ownerId) REFERENCES Drivers(_driverId)
);

CREATE TABLE Colors(
_colorId INTEGER primary key,
color1 TEXT,
color2 TEXT,
color3 TEXT);
INSERT INTO Colors(color1,color2,color3) VALUES ('#FFC107', '#FFA000', '#FF6F00');
    INSERT INTO Colors(color1,color2,color3) VALUES ('#E91E63', '#C2185B', '#880E4F');
    INSERT INTO Colors(color1,color2,color3) VALUES ('#9C27B0', '8E24AA', '#6A1B9A');
    INSERT INTO Colors(color1,color2,color3) VALUES ('#3F51B5', '#303F9F', '#1A237E');
    INSERT INTO Colors(color1,color2,color3) VALUES ('#4CAF50', '#388E3C', '#1B5E20');
    INSERT INTO Colors(color1,color2,color3) VALUES ('#795548', '#5D4037', '#3E2723');
    INSERT INTO Colors(color1,color2,color3) VALUES ('#3F51B5', '#303F9F', '#1A237E');

ALTER TABLE devices RENAME TO devices_old;
CREATE TABLE Devices(
    _deviceId INTEGER PRIMARY KEY AUTOINCREMENT,
    desc TEXT,
    colorId INTEGER,
    carId INTEGER,
     FOREIGN KEY (carId) REFERENCES cars(_carId),
     FOREIGN KEY (colorId) REFERENCES colors(_colorId)
);
INSERT INTO Devices (_deviceId,desc, carId,colorId)
SELECT id,desc,1,COALESCE(
(SELECT _colorId FROM Colors WHERE (
(SELECT COUNT(*) FROM Devices WHERE Devices.colorId=colors._colorId) = 0
) ORDER BY RANDOM() LIMIT 1),
(SELECT _colorId FROM Colors ORDER BY RANDOM() LIMIT 1)
)
FROM devices_old;
DROP TABLE devices_old;

ALTER TABLE KeyPoints RENAME TO KeyPoints_old;
CREATE TABLE KeyPoints (_keyPointId INTEGER PRIMARY KEY, latitude DOUBLE NOT NULL, longitude DOUBLE NOT NULL, startTime INTEGER NOT NULL, endTime INTEGER NOT NULL, previousTrackId INTEGER, nextTrackId INTEGER, deviceId INTEGER NOT NULL,
 addressId INTEGER, mileage INTEGER NOT NULL DEFAULT 0,
 carId INTEGER,  FOREIGN KEY (addressId) REFERENCES Addresses (_addressId),  FOREIGN KEY (deviceId) REFERENCES DEVICES (_deviceId),  FOREIGN KEY (nextTrackId) REFERENCES Tracks (_trackId),
   FOREIGN KEY (previousTrackId) REFERENCES Tracks (_trackId),
 FOREIGN KEY (carId) REFERENCES Cars (_carId));
INSERT INTO KeyPoints (_keyPointId, latitude, longitude, startTime, endTime, previousTrackId, nextTrackId, deviceId, addressId,carId)
SELECT _keyPointId, latitude, longitude, startTime, endTime, previousTrackId, nextTrackId, deviceId, addressId,1 FROM KeyPoints_old;
DROP TABLE KeyPoints_old;


ALTER TABLE Tracks RENAME TO Tracks_old;
CREATE TABLE Tracks (_trackId INTEGER PRIMARY KEY, deviceId INTEGER NOT NULL, startKeyPointId INTEGER, endKeyPointId INTEGER, distance DOUBLE,  carId INTEGER,   FOREIGN KEY (deviceId) REFERENCES devices (_deviceId) ON UPDATE NO ACTION ON DELETE NO ACTION,  FOREIGN KEY (endKeyPointId) REFERENCES keyPoints (_keyPointId) ON UPDATE NO ACTION ON DELETE NO ACTION,  FOREIGN KEY (startKeyPointId) REFERENCES keyPoints (_keyPointId) ON UPDATE NO ACTION ON DELETE NO ACTION,FOREIGN KEY (carId) REFERENCES Cars (_carId));
INSERT INTO Tracks (_trackId, deviceId, startKeyPointId, endKeyPointId, distance,carId)
SELECT _trackId, deviceId, startKeyPointId, endKeyPointId, distance,1 FROM Tracks_old;
DROP TABLE Tracks_old;

ALTER TABLE TrackRecords ADD mileage INTEGER NOT NULL DEFAULT 0;
ALTER TABLE TrackPoints ADD mileage INTEGER NOT NULL DEFAULT 0;
