-- +migrate Up
ALTER TABLE CONTACTS ADD disabled INTEGER DEFAULT 0;