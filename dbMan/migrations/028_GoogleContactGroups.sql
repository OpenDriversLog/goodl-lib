-- +migrate Up
DROP TABLE IF EXISTS GoogleGroups;
DROP TABLE IF EXISTS GoogleContacts;
DROP TABLE IF EXISTS GoogleContacts_Groups;
DROP TABLE IF EXISTS GoogleContacts_Addresses;
DROP TABLE IF EXISTS GoogleAddresses;

CREATE TABLE GoogleGroups (
    _googleGroupId INTEGER PRIMARY KEY AUTOINCREMENT,
    syncId INTEGER NOT NULL,
    key TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT "",
    tripType INTEGER NOT NULL DEFAULT 3,
    lastUpdate INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (syncId) REFERENCES Sync(_syncId)
);

CREATE TABLE GoogleContacts (
    _googleContactId INTEGER PRIMARY KEY AUTOINCREMENT,
    syncId INTEGER NOT NULL,
    key TEXT NOT NULL,
    lastUpdate INTEGER NOT NULL DEFAULT 0,
    name TEXT DEFAULT "",
    FOREIGN KEY (syncId) REFERENCES Sync(_syncId)
);

CREATE TABLE GoogleContacts_Groups (
    googleContactId INTEGER NOT NULL,
    googleGroupId INTEGER NOT NULL,
    FOREIGN KEY (googleContactId) REFERENCES GoogleContacts(_googleContactId),
    FOREIGN KEY (googleGroupId) REFERENCES GoogleGroups(_googleGroupId)
);

CREATE TABLE GoogleAddresses (
    _googleAddressId INTEGER PRIMARY KEY AUTOINCREMENT,
    syncId INTEGER NOT NULL,
    contactId INTEGER,
    rel STRING DEFAULT "http://schemas.google.com/g/2005#home",
    formattedAddress STRING DEFAULT "",
    tripType INTEGER NOT NULL DEFAULT 3,
    retryTime INTEGER DEFAULT 0,
    tryCount INTEGER DEFAULT 0
);

CREATE TABLE GoogleContacts_Addresses (
    googleContactId INTEGER NOT NULL,
    googleAddressId INTEGER NOT NULL,
    FOREIGN KEY (googleContactId) REFERENCES GoogleContacts(_googleContactId),
    FOREIGN KEY (googleAddressId) REFERENCES GoogleAddresses(_googleAddressId)
);

ALTER TABLE Contacts ADD syncedWith TEXT NOT NULL DEFAULT "";