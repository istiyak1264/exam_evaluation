-- An academic session is now identified by its name alone; the short code
-- ("A26") was a second thing to type and nothing keyed off it.
ALTER TABLE academic_sessions DROP INDEX sessions_code_uidx;
ALTER TABLE academic_sessions DROP COLUMN code;
