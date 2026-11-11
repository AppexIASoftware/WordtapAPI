-- Apply manually to an isolated non-production database after reviewing backup and schema.
CREATE TABLE simulated_course_purchases (
  id VARCHAR(36) NOT NULL PRIMARY KEY,
  user_id VARCHAR(36) NOT NULL,
  course_id VARCHAR(36) NOT NULL,
  amount_cents INT NOT NULL,
  currency CHAR(3) NOT NULL,
  created_at DATETIME(3) NOT NULL,
  UNIQUE KEY uq_simulated_purchase_user_course (user_id, course_id),
  CONSTRAINT fk_simulated_purchase_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
  CONSTRAINT fk_simulated_purchase_course FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE RESTRICT
) ENGINE=InnoDB;

CREATE TABLE simulated_course_purchase_keys (
  user_id VARCHAR(36) NOT NULL,
  idempotency_key VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  purchase_id VARCHAR(36) NOT NULL,
  PRIMARY KEY (user_id, idempotency_key),
  CONSTRAINT fk_simulated_purchase_key_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
  CONSTRAINT fk_simulated_purchase_key_purchase FOREIGN KEY (purchase_id) REFERENCES simulated_course_purchases(id) ON DELETE RESTRICT
) ENGINE=InnoDB;
