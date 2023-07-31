ALTER TABLE repos RENAME TO repos_old;

CREATE TABLE repos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  project_name TEXT NOT NULL,
  description TEXT NOT NULL,
  private BOOLEAN NOT NULL,
  mirror BOOLEAN NOT NULL,
  hidden BOOLEAN NOT NULL,
  user_id INTEGER NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL,
  CONSTRAINT user_id_fk
  FOREIGN KEY(user_id) REFERENCES users(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

INSERT INTO repos (id, name, project_name, description, private, mirror, hidden, user_id, created_at, updated_at)
SELECT id, name, project_name, description, private, mirror, hidden, (
  SELECT id FROM users WHERE admin = true ORDER BY id LIMIT 1
), created_at, updated_at
FROM repos_old;

