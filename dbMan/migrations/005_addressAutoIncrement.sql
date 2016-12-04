-- +migrate Up

ALTER TABLE Addresses RENAME TO Addresses_Old;
CREATE TABLE Addresses(
    _addressId INTEGER PRIMARY KEY AUTOINCREMENT,
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

-- +migrate Down