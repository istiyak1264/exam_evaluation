-- Deleting an account must not be blocked by the audit trail; the trail row
-- stays and loses its actor reference.

ALTER TABLE audit_log DROP FOREIGN KEY audit_actor_fk;
ALTER TABLE audit_log ADD CONSTRAINT audit_actor_fk
  FOREIGN KEY (actor_id) REFERENCES users (id) ON DELETE SET NULL;
