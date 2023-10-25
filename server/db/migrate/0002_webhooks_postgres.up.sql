CREATE TABLE IF NOT EXISTS webhooks (
  id SERIAL PRIMARY KEY,
  repo_id INTEGER NOT NULL,
  url TEXT NOT NULL,
  secret TEXT NOT NULL,
  content_type INTEGER NOT NULL,
  active BOOLEAN NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL,
  UNIQUE (repo_id, url),
  CONSTRAINT repo_id_fk
  FOREIGN KEY(repo_id) REFERENCES repos(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS webhook_events (
  id SERIAL PRIMARY KEY,
  webhook_id INTEGER NOT NULL,
  event INTEGER NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (webhook_id, event),
  CONSTRAINT webhook_id_fk
  FOREIGN KEY(webhook_id) REFERENCES webhooks(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
  id TEXT PRIMARY KEY,
  webhook_id INTEGER NOT NULL,
  event INTEGER NOT NULL,
  request_url TEXT NOT NULL,
  request_method TEXT NOT NULL,
  request_error TEXT,
  request_headers TEXT NOT NULL,
  request_body TEXT NOT NULL,
  response_status INTEGER NOT NULL,
  response_headers TEXT NOT NULL,
  response_body TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT webhook_id_fk
  FOREIGN KEY(webhook_id) REFERENCES webhooks(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);
