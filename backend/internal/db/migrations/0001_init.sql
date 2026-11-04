-- Secure exam evaluation schema (MySQL 8).
-- Core invariant: an evaluator never sees student identity for a script;
-- scripts are addressed by an opaque cover-sheet code held by the hall team.

CREATE TABLE IF NOT EXISTS users (
  id            CHAR(36) PRIMARY KEY,
  email         VARCHAR(320) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  full_name     VARCHAR(160) NOT NULL,
  role          ENUM('ADMIN','HALL_MANAGER','EVALUATOR','STUDENT') NOT NULL,
  is_active     TINYINT(1) NOT NULL DEFAULT 1,
  created_at    DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at    DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  UNIQUE KEY users_email_uidx (email),
  KEY users_role_idx (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS academic_sessions (
  id         CHAR(36) PRIMARY KEY,
  name       VARCHAR(64) NOT NULL,
  code       VARCHAR(16) NOT NULL,
  is_active  TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  UNIQUE KEY sessions_name_uidx (name),
  UNIQUE KEY sessions_code_uidx (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS students (
  id         CHAR(36) PRIMARY KEY,
  student_no VARCHAR(64) NOT NULL,
  full_name  VARCHAR(160) NOT NULL,
  email      VARCHAR(320) NULL,
  session_id CHAR(36) NOT NULL,
  user_id    CHAR(36) NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  UNIQUE KEY students_no_uidx (student_no),
  UNIQUE KEY students_user_uidx (user_id),
  KEY students_name_idx (full_name),
  KEY students_session_idx (session_id),
  CONSTRAINT students_session_fk FOREIGN KEY (session_id) REFERENCES academic_sessions(id),
  CONSTRAINT students_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS courses (
  id            CHAR(36) PRIMARY KEY,
  code          VARCHAR(32) NOT NULL,
  title         VARCHAR(200) NOT NULL,
  session_id    CHAR(36) NOT NULL,
  max_marks     DECIMAL(5,2) NOT NULL DEFAULT 100.00,
  pass_marks    DECIMAL(5,2) NOT NULL DEFAULT 40.00,
  instructor_id CHAR(36) NULL,
  is_active     TINYINT(1) NOT NULL DEFAULT 1,
  created_at    DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  UNIQUE KEY courses_code_session_uidx (code, session_id),
  CONSTRAINT courses_session_fk FOREIGN KEY (session_id) REFERENCES academic_sessions(id),
  CONSTRAINT courses_instructor_fk FOREIGN KEY (instructor_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Blinded script: this row is the only link back to the student.
CREATE TABLE IF NOT EXISTS scripts (
  id            CHAR(36) PRIMARY KEY,
  script_code   VARCHAR(40) NOT NULL,
  course_id     CHAR(36) NOT NULL,
  student_id    CHAR(36) NOT NULL,
  status        ENUM('REGISTERED','ASSIGNED','EVAL_PENDING','THIRD_REVIEW','RESOLVED','WITHDRAWN')
                NOT NULL DEFAULT 'REGISTERED',
  pdf_key       VARCHAR(255) NOT NULL,
  pdf_sha256    CHAR(64) NOT NULL,
  pdf_bytes     BIGINT NOT NULL,
  page_count    INT NULL,
  registered_by CHAR(36) NULL,
  registered_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  resolved_at   DATETIME(6) NULL,
  UNIQUE KEY scripts_code_uidx (script_code),
  UNIQUE KEY scripts_course_student_uidx (course_id, student_id),
  KEY scripts_status_idx (status),
  KEY scripts_student_idx (student_id),
  CONSTRAINT scripts_course_fk FOREIGN KEY (course_id) REFERENCES courses(id),
  CONSTRAINT scripts_student_fk FOREIGN KEY (student_id) REFERENCES students(id),
  CONSTRAINT scripts_registrar_fk FOREIGN KEY (registered_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS script_assignments (
  id           CHAR(36) PRIMARY KEY,
  script_id    CHAR(36) NOT NULL,
  evaluator_id CHAR(36) NOT NULL,
  slot         ENUM('FIRST','SECOND','THIRD') NOT NULL,
  assigned_by  CHAR(36) NULL,
  assigned_at  DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  UNIQUE KEY assignments_script_slot_uidx (script_id, slot),
  KEY assignments_evaluator_idx (evaluator_id),
  CONSTRAINT assignments_script_fk FOREIGN KEY (script_id) REFERENCES scripts(id) ON DELETE CASCADE,
  CONSTRAINT assignments_evaluator_fk FOREIGN KEY (evaluator_id) REFERENCES users(id),
  CONSTRAINT assignments_by_fk FOREIGN KEY (assigned_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS marks (
  id           CHAR(36) PRIMARY KEY,
  script_id    CHAR(36) NOT NULL,
  slot         ENUM('FIRST','SECOND','THIRD') NOT NULL,
  evaluator_id CHAR(36) NOT NULL,
  score        DECIMAL(5,2) NOT NULL,
  comment      TEXT NULL,
  submitted_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  UNIQUE KEY marks_script_slot_uidx (script_id, slot),
  UNIQUE KEY marks_script_evaluator_uidx (script_id, evaluator_id),
  CONSTRAINT marks_script_fk FOREIGN KEY (script_id) REFERENCES scripts(id) ON DELETE CASCADE,
  CONSTRAINT marks_evaluator_fk FOREIGN KEY (evaluator_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS results (
  script_id   CHAR(36) PRIMARY KEY,
  final_marks DECIMAL(5,2) NOT NULL,
  method      VARCHAR(32) NOT NULL,
  gap         DECIMAL(5,2) NULL,
  resolved_by CHAR(36) NULL,
  resolved_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  CONSTRAINT results_script_fk FOREIGN KEY (script_id) REFERENCES scripts(id) ON DELETE CASCADE,
  CONSTRAINT results_by_fk FOREIGN KEY (resolved_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS grade_challenges (
  id            CHAR(36) PRIMARY KEY,
  script_id     CHAR(36) NOT NULL,
  student_id    CHAR(36) NOT NULL,
  reason        TEXT NOT NULL,
  status        ENUM('OPEN','UPHELD','DISMISSED') NOT NULL DEFAULT 'OPEN',
  decided_by    CHAR(36) NULL,
  decision_note TEXT NULL,
  created_at    DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  decided_at    DATETIME(6) NULL,
  KEY challenges_script_idx (script_id),
  CONSTRAINT challenges_script_fk FOREIGN KEY (script_id) REFERENCES scripts(id) ON DELETE CASCADE,
  CONSTRAINT challenges_student_fk FOREIGN KEY (student_id) REFERENCES students(id),
  CONSTRAINT challenges_by_fk FOREIGN KEY (decided_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Append-only audit trail.
CREATE TABLE IF NOT EXISTS audit_log (
  id         BIGINT AUTO_INCREMENT PRIMARY KEY,
  actor_id   CHAR(36) NULL,
  actor_role ENUM('ADMIN','HALL_MANAGER','EVALUATOR','STUDENT') NULL,
  action     VARCHAR(64) NOT NULL,
  entity     VARCHAR(40) NOT NULL,
  entity_id  CHAR(36) NULL,
  meta       JSON NOT NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  KEY audit_entity_idx (entity, entity_id),
  KEY audit_created_idx (created_at),
  CONSTRAINT audit_actor_fk FOREIGN KEY (actor_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id         CHAR(36) PRIMARY KEY,
  user_id    CHAR(36) NOT NULL,
  token_hash CHAR(64) NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  revoked_at DATETIME(6) NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  UNIQUE KEY refresh_hash_uidx (token_hash),
  KEY refresh_user_idx (user_id),
  CONSTRAINT refresh_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
