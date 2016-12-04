-- +migrate Up
UPDATE Colors SET color2="#8E24AA" WHERE color2="8E24AA";
UPDATE Colors SET color1 = "#90a4ae",color2="#607d8b",color3="#455a64" WHERE _colorId=7;