-- +migrate Up
ALTER TABLE Addresses ADD fuel TEXT;

DROP VIEW IF EXISTS AddressesWithGeoZones;
CREATE VIEW AddressesWithGeoZones AS
SELECT CASE WHEN _geoFenceRegionId IS NOT NULL THEN 1 ELSE 0 END AS hasGeoFenceRegion, _addressId,street,postal,city,additional1,additional2,
HouseNumber,title,latitude,longitude,fuel, _geoFenceRegionId,outerMinLat,outerMinLon,outerMaxLat,outerMaxLon,color,rectangleId,
Rectangles.topLeftLat,Rectangles.topLeftLon,Rectangles.botRightLat,Rectangles.botRightLon FROM Addresses LEFT JOIN Address_GeoFenceRegion ON _addressId=addressId LEFT JOIN GeoFenceRegions on
	geoFenceRegionId = _geoFenceRegionId LEFT JOIN Rectangles ON rectangleId=_rectangleId;

ALTER TABLE TRIPS RENAME TO TRIPS_OLD;
CREATE TABLE `Trips` (
    _tripId INTEGER PRIMARY KEY,
    type INTEGER, -- private, homeway, worktour
    title STRING,
    desc STRING,
    driverId INTEGER,
    startContactId INTEGER,
    endContactId INTEGER,
    isReturnTrip INTEGER,
    contactId INTEGER,
    FOREIGN KEY (driverId) REFERENCES Drivers(_driverId),
    FOREIGN KEY (startContactId) REFERENCES Contacts(_contactId),
    FOREIGN KEY (endContactId) REFERENCES Contacts(_contactId),
    FOREIGN KEY (contactId) REFERENCES Contacts(_contactId)
);
INSERT INTO TRIPS (_tripId,type,title,desc,driverId,startContactId,endContactId,isReturnTrip,contactId)
SELECT _tripId,type,title,desc,driverId,NULL,NULL,NULL,contactId
FROM TRIPS_OLD;
DROP TABLE TRIPS_OLD;
ALTER TABLE GeoFenceRegions ADD priority INTEGER DEFAULT 0 NOT NULL;
CREATE VIEW Trips_Start_EndTrack AS
SELECT tripId,
(SELECT trackId FROM Tracks LEFT JOIN
KeyPoints ON startKeyPointId=_keyPointId
WHERE Tracks._trackId=Tracks_Trips.trackId
ORDER BY endTime ASC
LIMIT 1) AS startTrackId,
(
SELECT trackId FROM Tracks LEFT JOIN
KeyPoints ON endKeyPointId=_keyPointId
WHERE Tracks._trackId=Tracks_Trips.trackId
ORDER BY startTime DESC
LIMIT 1
) AS endTrackId
FROM Tracks_Trips GROUP BY TripId;

DROP VIEW IF EXISTS Trips_Start_EndTrack;
CREATE VIEW Trips_Start_EndTrack AS
SELECT tripId,
(SELECT trackId FROM Tracks LEFT JOIN
KeyPoints ON startKeyPointId=_keyPointId
WHERE Tracks._trackId=Tracks_Trips.trackId
ORDER BY endTime ASC
LIMIT 1) AS startTrackId,
(
SELECT trackId FROM Tracks LEFT JOIN
KeyPoints ON endKeyPointId=_keyPointId
WHERE Tracks._trackId=Tracks_Trips.trackId
ORDER BY startTime DESC
LIMIT 1
) AS endTrackId
FROM Tracks_Trips GROUP BY TripId;

DROP VIEW IF EXISTS Tracks_Trips_Start_End_Kp_Tracks_AllTrackIds_NoGroup;
CREATE VIEW Tracks_Trips_Start_End_Kp_Tracks_AllTrackIds_NoGroup AS
SELECT TSET.tripId, T.type AS tripType,T.title AS tripTitle,T.desc AS tripDesc,T.DriverId AS tripDriverId,T.startContactId AS tripStartContactId,
T.endContactId AS tripEndContactId,T.isReturnTrip,T.contactId AS tripContactId, SKP._keyPointId as sKeyPointId,
SKP.latitude AS sLatitude,SKP.longitude AS sLongitude,SKP.startTime AS sStartTime,
SKP.endTime AS sEndTime,SKP.addressId AS sAddressId,SKP.previousTrackId AS sPreviousTrackId,SKP.nextTrackId AS sNextTrackId,
 EKP._keyPointId AS eKeyPointId,
 EKP.latitude AS eLatitude,EKP.longitude AS eLongitude,EKP.startTime AS eStartTime,
 EKP.endTime AS eEndTime,EKP.addressId AS eAddressId,
 EKP.previousTrackId AS ePreviousTrackId,EKP.nextTrackId AS eNextTrackId,
 ST.deviceId AS sDeviceId,
 ET.deviceId AS eDeviceId,startTrackId,endTrackId,
TT.trackId
FROM Trips_Start_EndTrack TSET LEFT JOIN
TRIPS T ON T._tripID = TSET.tripId LEFT JOIN
TRACKS ST ON startTrackId=ST._trackId LEFT JOIN
TRACKS ET on endTrackId=ET._trackId
LEFT JOIN KeyPoints SKP ON ST.startKeyPointId=SKP._keyPointId
LEFT JOIN KeyPoints EKP ON ET.endKeyPointId=EKP._keyPointId
LEFT JOIN Tracks_Trips TT ON TSET.tripId=TT.tripId;
DROP VIEW IF EXISTS Tracks_Trips_Start_End_Kp_Tracks_AllTrackIds;
CREATE VIEW Tracks_Trips_Start_End_Kp_Tracks_AllTrackIds AS
SELECT tripId,tripType,tripTitle,tripDesc,tripDriverId,tripStartContactId,
tripEndContactId,isReturnTrip,tripContactId,sKeyPointId,
sLatitude,sLongitude,sStartTime,
sEndTime,sAddressId,sPreviousTrackId,sNextTrackId,
eKeyPointId,eLatitude,eLongitude,eStartTime,eEndTime,eAddressId,
 ePreviousTrackId,eNextTrackId,sDeviceId,eDeviceId,startTrackId,endTrackId
, GROUP_CONCAT(DISTINCT trackId) AS trackIds
FROM Tracks_Trips_Start_End_Kp_Tracks_AllTrackIds_NoGroup
GROUP BY tripId;
DROP VIEW IF EXISTS Trips_FullBlown;
CREATE VIEW Trips_FullBlown AS
SELECT tripId,tripType,tripTitle,tripDesc,tripDriverId,tripStartContactId,
tripEndContactId,isReturnTrip,tripContactId,sKeyPointId,
sLatitude,sLongitude,sStartTime,
sEndTime,sAddressId,
SA.street AS sStreet,
SA.postal AS sPostal,
SA.city AS sCity,
SA.additional1 AS sAdditional1,
SA.additional2 AS sAdditional2,
SA.latitude AS sAddLatitude,
SA.longitude AS sAddLongitude,
SA.HouseNumber AS sHouseNumber,
SA.title AS sAddTitle,
sPreviousTrackId,sNextTrackId,
eKeyPointId,eLatitude,eLongitude,eStartTime,eEndTime,eAddressId,
EA.street AS eStreet,
EA.postal AS ePostal,
EA.city AS eCity,
EA.additional1 AS eAdditional1,
EA.additional2 AS eAdditional2,
EA.latitude AS eAddLatitude,
EA.longitude AS eAddLongitude,
EA.HouseNumber AS eHouseNumber,
EA.title AS eAddTitle,
 ePreviousTrackId,eNextTrackId,sDeviceId,eDeviceId,startTrackId,endTrackId
,trackIds,
SGC._contactId AS proposedSContactId,
SGC.type AS proposedSContactType,
SGC.title AS proposedSContactTitle,
SGC.description AS proposedSContactDescription,
SGC.additional AS proposedSContactAdditional,
SGC.addressId AS proposedSContactAddressId,
SGC.tripTypeId AS proposedSContactTripTypeId,
SGCA.street AS proposedSContactStreet,
SGCA.postal AS proposedSContactPostal,
SGCA.city AS proposedSContactCity,
SGCA.additional1 AS proposedSContactAdditional1,
SGCA.additional2 AS proposedSContactAdditional2,
SGCA.latitude AS proposedSContactLatitude,
SGCA.longitude AS proposedSContactLongitude,
SGCA.HouseNumber AS proposedSContactHouseNumber,
SGCA.title AS proposedSContactAddTitle,
EGC._contactId AS proposedEContactId,
EGC.type AS proposedEContactType,
EGC.title AS proposedEContactTitle,
EGC.description AS proposedEContactDescription,
EGC.additional AS proposedEContactAdditional,
EGC.addressId AS proposedEContactAddressId,
EGCA.street AS proposedEContactStreet,
EGCA.postal AS proposedEContactPostal,
EGCA.city AS proposedEContactCity,
EGCA.additional1 AS proposedEContactAdditional1,
EGCA.additional2 AS proposedEContactAdditional2,
EGCA.latitude AS proposedEContactLatitude,
EGCA.longitude AS proposedEContactLongitude,
EGCA.HouseNumber AS proposedEContactHouseNumber,
EGCA.title AS proposedEContactAddTitle,
EGC.tripTypeId AS proposedEContactTripTypeId,
STC._contactId AS sContactId,
STC.type AS sContactType,
STC.title AS sContactTitle,
STC.description AS sContactDescription,
STC.additional AS sContactAdditional,
STC.addressId AS sContactAddressId,
STCA.street AS sContactStreet,
STCA.postal AS sContactPostal,
STCA.city AS sContactCity,
STCA.additional1 AS sContactAdditional1,
STCA.additional2 AS sContactAdditional2,
STCA.latitude AS sContactLatitude,
STCA.longitude AS sContactLongitude,
STCA.HouseNumber AS sContactHouseNumber,
STCA.title AS sContactAddTitle,
STC.tripTypeId AS sContactTripTypeId,
ETC._contactId AS eContactId,
ETC.type AS eContactAType,
ETC.title AS eContactATitle,
ETC.description AS eContactDescription,
ETC.additional AS eContactAdditional,
ETC.addressId AS eContactAddressId,
ETCA.street AS eContactStreet,
ETCA.postal AS eContactPostal,
ETCA.city AS eContactCity,
ETCA.additional1 AS eContactAdditional1,
ETCA.additional2 AS eContactAdditional2,
ETCA.latitude AS eContactLatitude,
ETCA.longitude AS eContactLongitude,
ETCA.HouseNumber AS eContactHouseNumber,
ETCA.title AS eContactAddTitle,
ETC.tripTypeId AS eContactTripTypeId,
TC.type AS tripContactType,
TC.title AS tripContactTitle,
TC.description AS tripContactDescription,
TC.additional AS tripContactAdditional,
TC.addressId AS tripContactAddressId,
TCA.street AS tripContactStreet,
TCA.postal AS tripContactPostal,
TCA.city AS tripContactCity,
TCA.additional1 AS tripContactAdditional1,
TCA.additional2 AS tripContactAdditional2,
TCA.latitude AS tripContactLatitude,
TCA.longitude AS tripContactLongitude,
TCA.HouseNumber AS tripContactHouseNumber,
TCA.title AS tripContactAddTitle,
TC.tripTypeId AS tripContactTripTypeId
FROM Tracks_Trips_Start_End_Kp_Tracks_AllTrackIds
LEFT JOIN KeyPoints_GeoFenceRegions SKGF ON SKGF.keyPointId=sKeyPointId
LEFT JOIN KeyPoints_GeoFenceRegions EKGF ON EKGF.keyPointId=eKeyPointId
LEFT JOIN Address_GeoFenceRegion EGF ON EKGF.geoFenceRegionId=EGF.geoFenceRegionId
LEFT JOIN Address_GeoFenceRegion SGF ON SKGF.geoFenceRegionId=SGF.geoFenceRegionId
LEFT JOIN Contacts SGC ON SGC.addressId=SGF.addressId
Left JOIN Contacts EGC ON EGC.addressId=EGF.addressId
LEFT JOIN Contacts STC ON STC._contactId=tripStartContactId
LEFT JOIN Contacts ETC ON ETC._contactId=tripEndContactId
LEFT JOIN Contacts TC ON TC._contactId=tripContactId
LEFT JOIN Addresses EA ON EA._addressId=eAddressId
LEFT JOIN Addresses SA ON SA._addressId=sAddressId
LEFT JOIN Addresses ETCA ON ETCA._addressId=ETC.addressId
LEFT JOIN Addresses STCA ON STCA._addressId=STC.addressId
LEFT JOIN Addresses TCA ON TCA._addressId=TC.addressId
LEFT JOIN Addresses EGCA ON EGCA._addressId=EGC.addressId
LEFT JOIN Addresses SGCA ON SGCA._addressId=SGC.addressId;

