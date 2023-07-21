ALTER TABLE users ADD COLUMN password TEXT;

CREATE TABLE IF NOT EXISTS access_tokens (
  id INTEGER primary key autoincrement,
  token text NOT NULL UNIQUE,
  name text NOT NULL,
  user_id INTEGER NOT NULL,
  expires_at DATETIME,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL,
  CONSTRAINT user_id_fk
  FOREIGN KEY (user_id) REFERENCES users(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);
