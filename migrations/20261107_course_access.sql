-- Review and apply explicitly to the intended database. This migration does not
-- remove legacy simulated purchases; their migration requires a separate review.
CREATE TABLE course_access (
  id VARCHAR(36) NOT NULL PRIMARY KEY,
  user_id VARCHAR(36) NOT NULL,
  course_id VARCHAR(36) NOT NULL,
  status VARCHAR(20) NOT NULL,
  source VARCHAR(30) NOT NULL,
  created_at DATETIME(3) NOT NULL,
  UNIQUE KEY uq_course_access_user_course (user_id, course_id),
  CONSTRAINT fk_course_access_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
  CONSTRAINT fk_course_access_course FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE RESTRICT
) ENGINE=InnoDB;
