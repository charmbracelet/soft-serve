CREATE TABLE IF NOT EXISTS handles (
  id SERIAL PRIMARY KEY,
  handle TEXT NOT NULL UNIQUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS organizations (
  id SERIAL PRIMARY KEY,
  name TEXT,
  contact_email TEXT NOT NULL,
  handle_id INTEGER NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL,
  CONSTRAINT handle_id_fk
  FOREIGN KEY(handle_id) REFERENCES handles(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS organization_members (
  id SERIAL PRIMARY KEY,
  org_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL,
  UNIQUE (org_id, user_id),
  CONSTRAINT org_id_fk
  FOREIGN KEY(org_id) REFERENCES organizations(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE,
  CONSTRAINT user_id_fk
  FOREIGN KEY(user_id) REFERENCES users(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS teams (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  org_id INTEGER NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL,
  UNIQUE (name, org_id),
  CONSTRAINT org_id_fk
  FOREIGN KEY(org_id) REFERENCES organizations(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS team_members (
  id SERIAL PRIMARY KEY,
  team_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL,
  UNIQUE (team_id, user_id),
  CONSTRAINT team_id_fk
  FOREIGN KEY(team_id) REFERENCES teams(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE,
  CONSTRAINT user_id_fk
  FOREIGN KEY(user_id) REFERENCES users(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS user_emails (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  email TEXT NOT NULL UNIQUE,
  is_primary BOOLEAN NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL,
  CONSTRAINT user_id_fk
  FOREIGN KEY(user_id) REFERENCES users(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

-- Add name to users table
ALTER TABLE users ADD COLUMN name TEXT;

-- Add handle_id to users table
ALTER TABLE users ADD COLUMN handle_id INTEGER;
ALTER TABLE users ADD CONSTRAINT handle_id_fk
  FOREIGN KEY(handle_id) REFERENCES handles(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE;

-- Migrate user username to handles
INSERT INTO handles (handle, updated_at) SELECT username, updated_at FROM users;

-- Update handle_id for users
UPDATE users SET handle_id = handles.id FROM handles WHERE handles.handle = users.username;

-- Make handle_id not null and unique
ALTER TABLE users ALTER COLUMN handle_id SET NOT NULL;
ALTER TABLE users ADD CONSTRAINT handle_id_unique UNIQUE (handle_id);

-- Drop username from users
ALTER TABLE users DROP COLUMN username;

-- Add org_id to repos table
ALTER TABLE repos ADD COLUMN org_id INTEGER;
ALTER TABLE repos ADD CONSTRAINT org_id_fk
  FOREIGN KEY(org_id) REFERENCES organizations(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE;

-- Alter user_id nullness in repos table
ALTER TABLE repos ALTER COLUMN user_id DROP NOT NULL;

-- Check that both user_id and org_id can't be null
ALTER TABLE repos ADD CONSTRAINT user_id_org_id_not_null CHECK (user_id IS NULL <> org_id IS NULL);

-- Add team_id to collabs table
ALTER TABLE collabs ADD COLUMN team_id INTEGER;
ALTER TABLE collabs ADD CONSTRAINT team_id_fk
  FOREIGN KEY(team_id) REFERENCES teams(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE;

-- Alter user_id nullness in collabs table
ALTER TABLE collabs ALTER COLUMN user_id DROP NOT NULL;

-- Check that both user_id and team_id can't be null
ALTER TABLE collabs ADD CONSTRAINT user_id_team_id_not_null CHECK (user_id IS NULL <> team_id IS NULL);

-- Alter unique constraint on collabs table
ALTER TABLE collabs DROP CONSTRAINT collabs_user_id_repo_id_key;
ALTER TABLE collabs ADD CONSTRAINT collabs_user_id_repo_id_team_id_key UNIQUE (user_id, repo_id, team_id);
