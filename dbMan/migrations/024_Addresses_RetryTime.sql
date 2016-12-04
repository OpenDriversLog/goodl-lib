-- +migrate Up
ALTER TABLE Addresses ADD retrytime INTEGER DEFAULT 0;
ALTER TABLE Addresses ADD trycount INTEGER DEFAULT 0;

CREATE INDEX Addresses_retrytime ON Addresses(retrytime)

