-- +migrate Up
CREATE TABLE IF NOT EXISTS `Drivers` (
    _driverId INTEGER,
    priority INTEGER,
    title STRING,
    desc STRING,
    addressId INTEGER,
    FOREIGN KEY (addressId) REFERENCES Addresses(_addressId)
);

CREATE TABLE IF NOT EXISTS `Contacts` (
    _contactId INTEGER,
    type INTEGER,
    title STRING,
    desc STRING,
    addressId INTEGER,
    FOREIGN KEY (addressId) REFERENCES Addresses(_addressId)
);

CREATE TABLE IF NOT EXISTS `Trips` (
    _tripId INTEGER PRIMARY KEY,
    type INTEGER, -- private, homeway, worktour
    title STRING,
    desc STRING,
    driverId STRING,
    contactId STRING,
    FOREIGN KEY (driverId) REFERENCES Drivers(_driverId),
    FOREIGN KEY (contactId) REFERENCES Contacts(_contactId)
);

CREATE TABLE IF NOT EXISTS `Tracks_Trips` (
    tripId INTEGER,
    trackId INTEGER,
    FOREIGN KEY (tripId) REFERENCES Trips(_tripId),
    FOREIGN KEY (trackId) REFERENCES Tracks(_trackId)
);

CREATE TABLE IF NOT EXISTS `Trip_History` (
    id INTEGER PRIMARY KEY,
    tripId INTEGER,
    changeDate DATE,
    -- changeUser userId,
    -- reason STRING,
    typeOLD INTEGER,
    typeNEW INTERGER,
    titleOLD STRING,
    titleNEW STRING,
    descOLD STRING,
    descNEW STRING,
    driverIdOLD INTEGER,
    driverIdNEW INTEGER,
    contactIdOLD INTEGER,
    contactIdNEW INTEGER
);

--  Create an update trigger to keep TripChangeHistory
CREATE TRIGGER IF NOT EXISTS update_tripHistory AFTER UPDATE ON Trips BEGIN INSERT INTO Trip_History (tripId, changeDate, typeOLD, typeNEW, titleOLD, titleNEW, descOLD, descNEW, driverIdOLD, driverIdNEW, contactIdOLD, contactIdNEW) values (new._tripId, DATETIME('NOW'),old.type, new.type, old.title, new.title, old.desc, new.desc, old.driverId, new.driverId, old.contactId, new.contactId ); END;

CREATE INDEX IF NOT EXISTS IDX_Trips_TripId ON Trips(_tripId);
CREATE INDEX IF NOT EXISTS IDX_Drivers_DriverId ON Drivers(_driverId);
CREATE INDEX IF NOT EXISTS IDX_Contacts_ContactId ON Contacts(_contactId);

-- +migrate Down
DROP INDEX IDX_Trips_TripId;
DROP INDEX IDX_Drivers_DriverId;
DROP INDEX IDX_Contacts_ContactId;
DROP TRIGGER update_tripHistory;

DROP TABLE Trip_History;
DROP TABLE Trips;
DROP TABLE Tracks_Trips;
DROP TABLE Drivers;
DROP TABLE Contacts;
