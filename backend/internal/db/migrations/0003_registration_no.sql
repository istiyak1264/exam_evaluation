-- Candidates are identified by student ID and registration number; the
-- registration number is what the sign-up form checks before linking.

ALTER TABLE students
  ADD COLUMN registration_no VARCHAR(64) NULL AFTER student_no,
  ADD UNIQUE KEY students_reg_uidx (registration_no);
