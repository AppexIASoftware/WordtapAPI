-- Apply manually only after verifying users.id has varchar(36) and its
-- character set/collation match the database defaults, and after a backup.
-- Intentionally fails if the table already exists; never replaces user data.
CREATE TABLE notification_schedule_modes (
  user_id varchar(36) NOT NULL PRIMARY KEY,
  mode varchar(10) NOT NULL,
  CONSTRAINT fk_notification_schedule_mode_user FOREIGN KEY (user_id)
    REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;
