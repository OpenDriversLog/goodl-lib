-- +migrate Up
CREATE TRIGGER AutoFillTripStartEndContact AFTER INSERT ON KeyPoints_GeoFenceRegions BEGIN
UPDATE Trips
SET startContactId=
(
SELECT proposedSContactId FROM Trips_FullBlown TF WHERE Trips._tripId=TF.tripId  AND proposedSContactId IS NOT NULL LIMIT 1
)
 WHERE _tripId IN (
 SELECT TE.tripID from Trips_Start_EndTrack TE LEFT JOIN Tracks ON startTrackId=_trackId
WHERE startKeyPointId  = NEW.keyPointId) AND startContactId IS null  OR startContactId<1; UPDATE Trips
		SET endContactId=
		(
		SELECT proposedEContactId FROM Trips_FullBlown TF WHERE Trips._tripId=TF.tripId  AND proposedEContactId IS NOT NULL LIMIT 1
		)
		WHERE _tripId IN (
			SELECT TE.tripID from Trips_Start_EndTrack TE LEFT JOIN Tracks ON endTrackId=_trackId
		WHERE endKeypointId = NEW.keyPointId) AND endContactId IS null OR endContactId<1;END;
