-- +migrate Up

DROP TABLE Contacts;
CREATE TABLE  Contacts (
    _contactId INTEGER PRIMARY KEY AUTOINCREMENT,
    type INTEGER  NOT NULL DEFAULT 1,
    title STRING DEFAULT "",
    description STRING DEFAULT "",
    additional STRING DEFAULT "",
    addressId INTEGER,
    tripTypeId INTEGER NOT NULL,
    FOREIGN KEY(addressId) REFERENCES Addresses(_addressId),
    FOREIGN KEY(tripTypeId) REFERENCES TripTypes(_tripTypeId)
);

-- +migrate Down