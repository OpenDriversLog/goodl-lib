-- +migrate Up
DROP VIEW IF EXISTS Trips_Start_EndTrack;
CREATE VIEW Trips_Start_EndTrack AS
SELECT _tripId AS tripId,
(SELECT trackId FROM Tracks_Trips LEFT JOIN
Tracks ON Tracks_Trips.trackId=Tracks._trackId LEFT JOIN
KeyPoints ON startKeyPointId=_keyPointId
WHERE Tracks_Trips.tripId=Trips._tripId
ORDER BY endTime ASC
LIMIT 1) AS startTrackId,
(
SELECT trackId FROM Tracks_Trips LEFT JOIN
Tracks ON Tracks_Trips.trackId=Tracks._trackId LEFT JOIN
KeyPoints ON endKeyPointId=_keyPointId
WHERE Tracks_Trips.tripId=Trips._tripId
ORDER BY startTime DESC
LIMIT 1
) AS endTrackId
FROM Trips;
