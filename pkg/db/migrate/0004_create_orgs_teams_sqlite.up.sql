CREATE TABLE IF NOT EXISTS handles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  handle TEXT NOT NULL UNIQUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS organizations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
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
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  access_level INTEGER NOT NULL,
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
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  org_id INTEGER NOT NULL,
  access_level INTEGER NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL,
  UNIQUE (name, org_id),
  CONSTRAINT org_id_fk
  FOREIGN KEY(org_id) REFERENCES organizations(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS team_members (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
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
  id INTEGER PRIMARY KEY AUTOINCREMENT,
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

ALTER TABLE users RENAME TO _users_old;

CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT,
  handle_id INTEGER NOT NULL UNIQUE,
  admin BOOLEAN NOT NULL,
  password TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL,
  CONSTRAINT handle_id_fk
  FOREIGN KEY(handle_id) REFERENCES handles(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

-- Migrate user username to handles
INSERT INTO handles (handle, updated_at) SELECT username, updated_at FROM _users_old;

-- Migrate users
INSERT INTO users (id, handle_id, admin, password, created_at, updated_at) SELECT id, (
  SELECT id FROM handles WHERE handle = _users_old.username
), admin, password, created_at, updated_at FROM _users_old;

-- Drop old table
DROP TABLE _users_old;

ALTER TABLE repos RENAME TO _repos_old;

CREATE TABLE IF NOT EXISTS repos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  project_name TEXT NOT NULL,
  description TEXT NOT NULL,
  private BOOLEAN NOT NULL,
  mirror BOOLEAN NOT NULL,
  hidden BOOLEAN NOT NULL,
  user_id INTEGER,
  org_id INTEGER,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL,
  CONSTRAINT user_id_fk
  FOREIGN KEY(user_id) REFERENCES users(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE,
  CONSTRAINT org_id_fk
  FOREIGN KEY(org_id) REFERENCES organizations(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE,
  CONSTRAINT user_id_org_id_not_null
  CHECK (user_id IS NULL <> org_id IS NULL)
);

-- Migrate repos
INSERT INTO repos (id, name, project_name, description, private, mirror, hidden, user_id, created_at, updated_at)
SELECT id, name, project_name, description, private, mirror, hidden, user_id, created_at, updated_at
FROM _repos_old;

-- Drop old table
DROP TABLE _repos_old;

-- Alter collabs table
ALTER TABLE collabs RENAME TO _collabs_old;

CREATE TABLE IF NOT EXISTS collabs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER,
  team_id INTEGER,
  repo_id INTEGER NOT NULL,
  access_level INTEGER NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL,
  UNIQUE (user_id, team_id, repo_id),
  CONSTRAINT user_id_fk
  FOREIGN KEY(user_id) REFERENCES users(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE,
  CONSTRAINT team_id_fk
  FOREIGN KEY(team_id) REFERENCES teams(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE,
  CONSTRAINT repo_id_fk
  FOREIGN KEY(repo_id) REFERENCES repos(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE,
  CONSTRAINT user_id_team_id_not_null
  CHECK (user_id IS NULL <> team_id IS NULL)
);

-- Migrate collabs
INSERT INTO collabs (id, user_id, team_id, repo_id, access_level, created_at, updated_at)
SELECT id, user_id, NULL, repo_id, access_level, created_at, updated_at
FROM _collabs_old;

-- Drop old table
DROP TABLE _collabs_old;
