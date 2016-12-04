-- +migrate Up
DROP TABLE IF EXISTS TutorialInfo;
CREATE TABLE IF NOT EXISTS TutorialInfo (
lastMilestone NOT NULL DEFAULT "",
disabled INTEGER NOT NULL DEFAULT 0
);
INSERT INTO TutorialInfo (lastMilestone, disabled) VALUES("",0);