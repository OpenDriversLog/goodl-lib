-- +migrate Up

CREATE INDEX IF NOT EXISTS IDX_TR_TimeMillis ON TrackRecords(timeMillis);
CREATE INDEX IF NOT EXISTS IDX_TR_DeviceKey ON TrackRecords(deviceId);
CREATE INDEX IF NOT EXISTS IDX_Address_AddressId ON Addresses(_addressId);
CREATE INDEX IF NOT EXISTS IDX_KeyPoints_AddressId ON KeyPoints(addressId);
CREATE INDEX IF NOT EXISTS IDX_KP_PreviousTrackId ON KeyPoints(previousTrackId);
CREATE INDEX IF NOT EXISTS IDX_KP_NextTrackId ON KeyPoints(nextTrackId);
CREATE INDEX IF NOT EXISTS IDX_T_StartKeyPointId ON Tracks(startKeyPointId);
CREATE INDEX IF NOT EXISTS IDX_T_EndKeyPointId ON Tracks(endKeyPointId);
CREATE INDEX IF NOT EXISTS IDX_Trips_TripId ON Trips(_tripId);
CREATE INDEX IF NOT EXISTS IDX_Drivers_DriverId ON Drivers(_driverId);
CREATE INDEX IF NOT EXISTS IDX_Contacts_ContactId ON Contacts(_contactId);
CREATE INDEX IF NOT EXISTS IDX_Tracks_Trips_History_trackIdOLD ON Tracks_Trips_History(trackIdOLD);
