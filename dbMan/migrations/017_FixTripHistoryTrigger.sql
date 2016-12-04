-- +migrate Up
ALTER TABLE Trip_History ADD startContactIdOLD INTEGER;
ALTER TABLE Trip_History ADD startContactIdNEW INTEGER;
ALTER TABLE Trip_History ADD endContactIdOLD INTEGER;
ALTER TABLE Trip_History ADD endContactIdNEW INTEGER;
ALTER TABLE Trip_History ADD isReturnTripOLD INTEGER;
ALTER TABLE Trip_History ADD isReturnTripNEW INTEGER;
ALTER TABLE Trip_History ADD isReviewedOLD INTEGER;
ALTER TABLE Trip_History ADD isReviewedNEW INTEGER;
--  Create an update trigger to keep TripChangeHistory
DROP TRIGGER IF EXISTS update_tripHistory;
CREATE TRIGGER update_tripHistory AFTER UPDATE ON Trips BEGIN INSERT INTO Trip_History (tripId, changeDate, typeOLD, typeNEW, titleOLD, titleNEW, descOLD, descNEW, driverIdOLD, driverIdNEW, contactIdOLD, contactIdNEW,startContactIdOLD,startContactIdNEW,endContactIdOLD,endContactIdNEW,isReturnTripOLD,isReturnTripNEW,isReviewedOLD,isReviewedNEW) values (new._tripId, DATETIME('NOW'),old.type, new.type, old.title, new.title, old.desc, new.desc, old.driverId, new.driverId, old.contactId, new.contactId,old.startContactId,new.startContactId,old.endContactId,new.endContactId,old.isReturnTrip,new.isReturnTrip,old.Reviewed,new.Reviewed); END;
