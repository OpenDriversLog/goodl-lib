-- +migrate Up
DROP TABLE IF EXISTS OAuth;
DROP TABLE IF EXISTS CardDavConfig;
DROP TABLE IF EXISTS CalDavConfig;
DROP TABLE IF EXISTS HttpBasicAuth;
DROP TABLE IF EXISTS HttpDigestAuth;
DROP TABLE IF EXISTS Sync;
CREATE TABLE OAuth (
_oAuthId INTEGER PRIMARY KEY AUTOINCREMENT,
refreshToken TEXT NOT NULL DEFAULT "",
accessToken TEXT DEFAULT "",
expirationTime INTEGER DEFAULT 0
);
CREATE TABLE CardDavConfig(
_cardDavConfigId INTEGER PRIMARY KEY AUTOINCREMENT,
type TEXT NOT NULL DEFAULT "Custom",
rootUri TEXT DEFAULT "",
addressBookName TEXT DEFAULT "",
principalName TEXT DEFAULT ""
);
CREATE TABLE CalDavConfig(
_calDavConfigId INTEGER PRIMARY KEY AUTOINCREMENT,
type TEXT NOT NULL DEFAULT "Custom",
rootUri TEXT DEFAULT "",
calendarName TEXT DEFAULT "",
principalName TEXT DEFAULT ""
);
CREATE TABLE HttpBasicAuth(
_httpBasicAuthId INTEGER PRIMARY KEY AUTOINCREMENT,
usr string DEFAULT "",
passwd string DEFAULT ""
);
CREATE TABLE HttpDigestAuth(
_httpDigestAuthId INTEGER PRIMARY KEY AUTOINCREMENT,
usr string DEFAULT "",
passwd string DEFAULT ""
);
CREATE TABLE Sync (
_syncId INTEGER PRIMARY KEY AUTOINCREMENT,
name TEXT NOT NULL DEFAULT "",
type TEXT NOT NULL DEFAULT "",
priority INTEGER NOT NULL DEFAULT 0,
lastUpdate INTEGER NOT NULL DEFAULT 0,
created INTEGER NOT NULL DEFAULT 0,
updateFrequency INTEGER NOT NULL DEFAULT 0,
cardDavConfigId INTEGER,
calDavConfigId INTEGER,
oAuthId INTEGER,
httpBasicAuthId INTEGER,
httpDigestAuthId INTEGER,
FOREIGN KEY (cardDavConfigId) REFERENCES CardDavConfig(_cardDavConfigId),
FOREIGN KEY (oAuthId) REFERENCES OAuth(_oAuthId),
FOREIGN KEY (httpBasicAuthId) REFERENCES HttpBasicAuth(_httpBasicAuthId),
FOREIGN KEY (httpDigestAuthId) REFERENCES HttpDigestAuth(_httpDigestAuthId),
FOREIGN KEY (calDavConfigId) REFERENCES CalDavConfig(_calDavConfigId)

)