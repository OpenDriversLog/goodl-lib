-- +migrate Up

CREATE TABLE IF NOT EXISTS `Tracks_Trips_History` (
    id INTEGER PRIMARY KEY,
    changeDate DATE,
    tripIdOLD INTEGER,
    tripIdNEW INTERGER,
    trackIdOLD STRING,
    trackIdNEW STRING,
    sqlAction STRING,
    timeEnter DATE
);

CREATE INDEX IF NOT EXISTS IDX_Tracks_Trips_History_trackIdOLD ON Tracks_Trips_History(trackIdOLD);
--  Create an update trigger to keep Trip_TracksHistory

--  Create an update trigger
CREATE TRIGGER update_triptrackshistory AFTER UPDATE  ON Tracks_Trips
BEGIN

  INSERT INTO Tracks_Trips_History  (tripIdNEW,tripIdOLD,trackIdOLD,trackIdNEW,
                        sqlAction,timeEnter)

          values (old.tripId,new.tripId,old.trackId,new.trackId,
           'UPDATE',
                  DATETIME('NOW') ); END;
--
--  Also create an insert trigger
--    NOTE  AFTER keyword ------v
CREATE TRIGGER insert_triptrackshistory AFTER INSERT ON Tracks_Trips
BEGIN
INSERT INTO Tracks_Trips_History  (tripIdNEW,trackIdNEW,
                      sqlAction,timeEnter)

          values (new.tripId,new.trackId,
                  'INSERT',DATETIME('NOW') ); END;

--  Also create a DELETE trigger
CREATE TRIGGER delete_triptrackshistory DELETE ON Tracks_Trips
BEGIN

INSERT INTO Tracks_Trips_History  (tripIdOLD,trackIdOLD,
                      sqlAction,timeEnter)

          values (old.tripId,old.trackId,
                  'DELETE',DATETIME('NOW') ); END;

