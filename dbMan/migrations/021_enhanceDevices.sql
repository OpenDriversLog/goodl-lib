-- +migrate Up
ALTER TABLE devices ADD checked INTEGER NOT NULL DEFAULT 1;