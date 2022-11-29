package sqlite

var (
	sqlCreateConfigTable = `CREATE TABLE IF NOT EXISTS config (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		host TEXT NOT NULL,
		port INTEGER NOT NULL,
		anon_access TEXT NOT NULL,
		allow_keyless BOOLEAN NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL
	);`

	sqlCreateUserTable = `CREATE TABLE IF NOT EXISTS user (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		login TEXT UNIQUE,
		email TEXT UNIQUE,
		password TEXT,
		admin BOOLEAN NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL
	);`

	sqlCreatePublicKeyTable = `CREATE TABLE IF NOT EXISTS public_key (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		public_key TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL,
		UNIQUE (user_id, public_key),
		CONSTRAINT user_id_fk
		FOREIGN KEY(user_id) REFERENCES user(id)
		ON DELETE CASCADE
		ON UPDATE CASCADE
	);`

	sqlCreateRepoTable = `CREATE TABLE IF NOT EXISTS repo (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		project_name TEXT NOT NULL,
		description TEXT NOT NULL,
		private BOOLEAN NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL
	);`

	sqlCreateCollabTable = `CREATE TABLE IF NOT EXISTS collab (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		repo_id INTEGER NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL,
		UNIQUE (user_id, repo_id),
		CONSTRAINT user_id_fk
		FOREIGN KEY(user_id) REFERENCES user(id)
		ON DELETE CASCADE
		ON UPDATE CASCADE,
		CONSTRAINT repo_id_fk
		FOREIGN KEY(repo_id) REFERENCES repo(id)
		ON DELETE CASCADE
		ON UPDATE CASCADE
	);`

	// Config.
	sqlInsertConfig        = `INSERT INTO config (name, host, port, anon_access, allow_keyless, updated_at) VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP);`
	sqlSelectConfig        = `SELECT id, name, host, port, anon_access, allow_keyless, created_at, updated_at FROM config WHERE id = ?;`
	sqlUpdateConfigName    = `UPDATE config SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;`
	sqlUpdateConfigHost    = `UPDATE config SET host = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;`
	sqlUpdateConfigPort    = `UPDATE config SET port = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;`
	sqlUpdateConfigAnon    = `UPDATE config SET anon_access = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;`
	sqlUpdateConfigKeyless = `UPDATE config SET allow_keyless = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;`

	// User.
	sqlInsertUser            = `INSERT INTO user (name, login, email, password, admin, updated_at) VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP);`
	sqlDeleteUser            = `DELETE FROM user WHERE id = ?;`
	sqlSelectUser            = `SELECT id, name, login, email, password, admin, created_at, updated_at FROM user WHERE id = ?;`
	sqlSelectUserByLogin     = `SELECT id, name, login, email, password, admin, created_at, updated_at FROM user WHERE login = ?;`
	sqlSelectUserByEmail     = `SELECT id, name, login, email, password, admin, created_at, updated_at FROM user WHERE email = ?;`
	sqlSelectUserByPublicKey = `SELECT u.id, u.name, u.login, u.email, u.password, u.admin, u.created_at, u.updated_at FROM user u INNER JOIN public_key pk ON u.id = pk.user_id WHERE pk.public_key = ?;`
	sqlUpdateUserName        = `UPDATE user SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;`
	sqlUpdateUserLogin       = `UPDATE user SET login = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;`
	sqlUpdateUserEmail       = `UPDATE user SET email = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;`
	sqlUpdateUserPassword    = `UPDATE user SET password = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;`
	sqlUpdateUserAdmin       = `UPDATE user SET admin = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;`
	sqlCountUsers            = `SELECT COUNT(*) FROM user;`

	// Public Key.
	sqlInsertPublicKey      = `INSERT INTO public_key (user_id, public_key, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP);`
	sqlDeletePublicKey      = `DELETE FROM public_key WHERE id = ?;`
	sqlSelectUserPublicKeys = `SELECT id, user_id, public_key, created_at, updated_at FROM public_key WHERE user_id = ?;`

	// Repo.
	sqlInsertRepo                  = `INSERT INTO repo (name, project_name, description, private, updated_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP);`
	sqlDeleteRepo                  = `DELETE FROM repo WHERE id = ?;`
	sqlDeleteRepoWithName          = `DELETE FROM repo WHERE name = ?;`
	sqlSelectRepoByName            = `SELECT id, name, project_name, description, private, created_at, updated_at FROM repo WHERE name = ?;`
	sqlUpdateRepoProjectNameByName = `UPDATE repo SET project_name = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?;`
	sqlUpdateRepoDescriptionByName = `UPDATE repo SET description = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?;`
	sqlUpdateRepoPrivateByName     = `UPDATE repo SET private = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?;`

	// Collab.
	sqlInsertCollab               = `INSERT INTO collab (user_id, repo_id, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP);`
	sqlInsertCollabByName         = `INSERT INTO collab (user_id, repo_id, updated_at) VALUES (?, (SELECT id FROM repo WHERE name = ?), CURRENT_TIMESTAMP);`
	sqlDeleteCollab               = `DELETE FROM collab WHERE user_id = ? AND repo_id = ?;`
	sqlDeleteCollabByName         = `DELETE FROM collab WHERE user_id = ? AND repo_id = (SELECT id FROM repo WHERE name = ?);`
	sqlSelectRepoCollabs          = `SELECT user.id, user.name, user.login, user.email, user.admin, user.created_at, user.updated_at FROM user INNER JOIN collab ON user.id = collab.user_id WHERE collab.repo_id = ?;`
	sqlSelectRepoCollabsByName    = `SELECT user.id, user.name, user.login, user.email, user.admin, user.created_at, user.updated_at FROM user INNER JOIN collab ON user.id = collab.user_id WHERE collab.repo_id = (SELECT id FROM repo WHERE name = ?);`
	sqlSelectRepoPublicKeys       = `SELECT public_key.id, public_key.user_id, public_key.public_key, public_key.created_at, public_key.updated_at FROM public_key INNER JOIN collab ON public_key.user_id = collab.user_id WHERE collab.repo_id = ?;`
	sqlSelectRepoPublicKeysByName = `SELECT public_key.id, public_key.user_id, public_key.public_key, public_key.created_at, public_key.updated_at FROM public_key INNER JOIN collab ON public_key.user_id = collab.user_id WHERE collab.repo_id = (SELECT id FROM repo WHERE name = ?);`
)
