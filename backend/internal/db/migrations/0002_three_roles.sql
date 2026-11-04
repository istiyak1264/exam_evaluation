-- Three roles only: ADMIN, EVALUATOR, STUDENT.
-- Existing hall managers are promoted to admin — they already did the
-- registration and appointment work that admin now covers.

UPDATE audit_log SET actor_role = 'ADMIN' WHERE actor_role = 'HALL_MANAGER';
UPDATE users SET role = 'ADMIN' WHERE role = 'HALL_MANAGER';

ALTER TABLE users
  MODIFY role ENUM('ADMIN','EVALUATOR','STUDENT') NOT NULL;

ALTER TABLE audit_log
  MODIFY actor_role ENUM('ADMIN','EVALUATOR','STUDENT') NULL;
