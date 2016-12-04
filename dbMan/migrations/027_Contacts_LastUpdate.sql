-- +migrate Up
ALTER TABLE Contacts ADD lastUpdate INTEGER DEFAULT 0;
ALTER TABLE CardDavConfig ADD syncToken STRING DEFAULT "";
ALTER TABLE CalDavConfig ADD syncToken STRING DEFAULT "";

