package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/types"
	"modernc.org/sqlite"
	sqlitelib "modernc.org/sqlite/lib"
)

var _ db.Store = &Sqlite{}

// Sqlite is a SQLite database.
type Sqlite struct {
	path string
	db   *sql.DB
}

// New creates a new DB in the given path.
func New(path string) (*Sqlite, error) {
	var err error
	log.Printf("Opening SQLite db: %s\n", path)
	db, err := sql.Open("sqlite", path+
		"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	d := &Sqlite{
		db:   db,
		path: path,
	}
	if err = d.CreateDB(); err != nil {
		return nil, fmt.Errorf("failed to create db: %w", err)
	}
	return d, d.db.Ping()
}

// Close closes the database.
func (d *Sqlite) Close() error {
	return d.db.Close()
}

// CreateDB creates the database and tables.
func (d *Sqlite) CreateDB() error {
	return d.wrapTransaction(func(tx *sql.Tx) error {
		if _, err := tx.Exec(sqlCreateUserTable); err != nil {
			return err
		}
		if _, err := tx.Exec(sqlCreatePublicKeyTable); err != nil {
			return err
		}
		if _, err := tx.Exec(sqlCreateRepoTable); err != nil {
			return err
		}
		if _, err := tx.Exec(sqlCreateCollabTable); err != nil {
			return err
		}
		return nil
	})
}

// AddUser adds a new user.
func (d *Sqlite) AddUser(name, login, email, password string, isAdmin bool) error {
	var l *string
	var e *string
	var p *string
	if login != "" {
		login = strings.ToLower(login)
		l = &login
	}
	if email != "" {
		email = strings.ToLower(email)
		e = &email
	}
	if password != "" {
		p = &password
	}
	if err := d.wrapTransaction(func(tx *sql.Tx) error {
		if _, err := tx.Exec(sqlInsertUser, name, l, e, p, isAdmin); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// DeleteUser deletes a user.
func (d *Sqlite) DeleteUser(id int) error {
	return d.wrapTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(sqlDeleteUser, id)
		return err
	})
}

// GetUser returns a user by ID.
func (d *Sqlite) GetUser(id int) (*types.User, error) {
	var u types.User
	if err := d.wrapTransaction(func(tx *sql.Tx) error {
		r := tx.QueryRow(sqlSelectUser, id)
		if err := r.Scan(&u.ID, &u.Name, &u.Login, &u.Email, &u.Password, &u.Admin, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByLogin returns a user by login.
func (d *Sqlite) GetUserByLogin(login string) (*types.User, error) {
	login = strings.ToLower(login)
	var u types.User
	if err := d.wrapTransaction(func(tx *sql.Tx) error {
		r := tx.QueryRow(sqlSelectUserByLogin, login)
		if err := r.Scan(&u.ID, &u.Name, &u.Login, &u.Email, &u.Password, &u.Admin, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByLogin returns a user by login.
func (d *Sqlite) GetUserByEmail(email string) (*types.User, error) {
	email = strings.ToLower(email)
	var u types.User
	if err := d.wrapTransaction(func(tx *sql.Tx) error {
		r := tx.QueryRow(sqlSelectUserByEmail, email)
		if err := r.Scan(&u.ID, &u.Name, &u.Login, &u.Email, &u.Password, &u.Admin, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByPublicKey returns a user by public key.
func (d *Sqlite) GetUserByPublicKey(key string) (*types.User, error) {
	var u types.User
	if err := d.wrapTransaction(func(tx *sql.Tx) error {
		r := tx.QueryRow(sqlSelectUserByPublicKey, key)
		if err := r.Scan(&u.ID, &u.Name, &u.Login, &u.Email, &u.Password, &u.Admin, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &u, nil
}

// SetUserName sets the user name.
func (d *Sqlite) SetUserName(user *types.User, name string) error {
	return d.wrapTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(sqlUpdateUserName, name, user.ID)
		return err
	})
}

// SetUserLogin sets the user login.
func (d *Sqlite) SetUserLogin(user *types.User, login string) error {
	if login == "" {
		return fmt.Errorf("login cannot be empty")
	}
	login = strings.ToLower(login)
	return d.wrapTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(sqlUpdateUserLogin, login, user.ID)
		return err
	})
}

// SetUserEmail sets the user email.
func (d *Sqlite) SetUserEmail(user *types.User, email string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	email = strings.ToLower(email)
	return d.wrapTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(sqlUpdateUserEmail, email, user.ID)
		return err
	})
}

// SetUserPassword sets the user password.
func (d *Sqlite) SetUserPassword(user *types.User, password string) error {
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}
	return d.wrapTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(sqlUpdateUserPassword, password, user.ID)
		return err
	})
}

// SetUserAdmin sets the user admin.
func (d *Sqlite) SetUserAdmin(user *types.User, admin bool) error {
	return d.wrapTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(sqlUpdateUserAdmin, admin, user.ID)
		return err
	})
}

// CountUsers returns the number of users.
func (d *Sqlite) CountUsers() (int, error) {
	var count int
	if err := d.wrapTransaction(func(tx *sql.Tx) error {
		r := tx.QueryRow(sqlCountUsers)
		if err := r.Scan(&count); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return 0, err
	}
	return count, nil
}

// AddUserPublicKey adds a new user public key.
func (d *Sqlite) AddUserPublicKey(user *types.User, key string) error {
	return d.wrapTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(sqlInsertPublicKey, user.ID, key)
		return err
	})
}

// DeleteUserPublicKey deletes a user public key.
func (d *Sqlite) DeleteUserPublicKey(id int) error {
	return d.wrapTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(sqlDeletePublicKey, id)
		return err
	})
}

// GetUserPublicKeys returns the user public keys.
func (d *Sqlite) GetUserPublicKeys(user *types.User) ([]*types.PublicKey, error) {
	keys := make([]*types.PublicKey, 0)
	if err := d.wrapTransaction(func(tx *sql.Tx) error {
		rows, err := tx.Query(sqlSelectUserPublicKeys, user.ID)
		if err != nil {
			return err
		}
		if err := rows.Err(); err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var k types.PublicKey
			if err := rows.Scan(&k.ID, &k.UserID, &k.PublicKey, &k.CreatedAt, &k.UpdatedAt); err != nil {
				return err
			}
			keys = append(keys, &k)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return keys, nil
}

// AddRepo adds a new repo.
func (d *Sqlite) AddRepo(name, projectName, description string, isPrivate bool) error {
	name = strings.ToLower(name)
	return d.wrapTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(sqlInsertRepo, name, projectName, description, isPrivate)
		return err
	})
}

// DeleteRepo deletes a repo.
func (d *Sqlite) DeleteRepo(name string) error {
	name = strings.ToLower(name)
	return d.wrapTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(sqlDeleteRepoWithName, name)
		return err
	})
}

// GetRepo returns a repo by name.
func (d *Sqlite) GetRepo(name string) (*types.Repo, error) {
	name = strings.ToLower(name)
	var r types.Repo
	if err := d.wrapTransaction(func(tx *sql.Tx) error {
		rows := tx.QueryRow(sqlSelectRepoByName, name)
		if err := rows.Scan(&r.ID, &r.Name, &r.ProjectName, &r.Description, &r.Private, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return err
		}
		if err := rows.Err(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &r, nil
}

// SetRepoProjectName sets the repo project name.
func (d *Sqlite) SetRepoProjectName(name string, projectName string) error {
	name = strings.ToLower(name)
	return d.wrapTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(sqlUpdateRepoProjectNameByName, projectName, name)
		return err
	})
}

// SetRepoDescription sets the repo description.
func (d *Sqlite) SetRepoDescription(name string, description string) error {
	name = strings.ToLower(name)
	return d.wrapTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(sqlUpdateRepoDescriptionByName, description,
			name)
		return err
	})
}

// SetRepoPrivate sets the repo private.
func (d *Sqlite) SetRepoPrivate(name string, private bool) error {
	name = strings.ToLower(name)
	return d.wrapTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(sqlUpdateRepoPrivateByName, private, name)
		return err
	})
}

// AddRepoCollab adds a new repo collaborator.
func (d *Sqlite) AddRepoCollab(repo string, user *types.User) error {
	return d.wrapTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(sqlInsertCollabByName, repo, user.ID)
		return err
	})
}

// DeleteRepoCollab deletes a repo collaborator.
func (d *Sqlite) DeleteRepoCollab(userID int, repoID int) error {
	return d.wrapTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(sqlDeleteCollab, repoID, userID)
		return err
	})
}

// ListRepoCollabs returns a list of repo collaborators.
func (d *Sqlite) ListRepoCollabs(repo string) ([]*types.User, error) {
	collabs := make([]*types.User, 0)
	if err := d.wrapTransaction(func(tx *sql.Tx) error {
		rows, err := tx.Query(sqlSelectRepoCollabsByName, repo)
		if err != nil {
			return err
		}
		if err := rows.Err(); err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var c types.User
			if err := rows.Scan(&c.ID, &c.Name, &c.Login, &c.Email, &c.Admin, &c.CreatedAt, &c.UpdatedAt); err != nil {
				return err
			}
			collabs = append(collabs, &c)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return collabs, nil
}

// ListRepoPublicKeys returns a list of repo public keys.
func (d *Sqlite) ListRepoPublicKeys(repo string) ([]*types.PublicKey, error) {
	keys := make([]*types.PublicKey, 0)
	if err := d.wrapTransaction(func(tx *sql.Tx) error {
		rows, err := tx.Query(sqlSelectRepoPublicKeysByName, repo)
		if err != nil {
			return err
		}
		if err := rows.Err(); err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var k types.PublicKey
			if err := rows.Scan(&k.ID, &k.UserID, &k.PublicKey, &k.CreatedAt, &k.UpdatedAt); err != nil {
				return err
			}
			keys = append(keys, &k)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return keys, nil
}

// IsRepoPublicKeyCollab returns true if the public key is a collaborator for the repository.
func (d *Sqlite) IsRepoPublicKeyCollab(repo string, key string) (bool, error) {
	var count int
	if err := d.wrapTransaction(func(tx *sql.Tx) error {
		rows := tx.QueryRow(sqlSelectRepoPublicKeyCollabByName, repo, key)
		if err := rows.Scan(&count); err != nil {
			return err
		}
		if err := rows.Err(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false, err
	}
	return count > 0, nil
}

// WrapTransaction runs the given function within a transaction.
func (d *Sqlite) wrapTransaction(f func(tx *sql.Tx) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("error starting transaction: %s", err)
		return err
	}
	for {
		err = f(tx)
		if err != nil {
			switch {
			case errors.Is(err, sql.ErrNoRows):
			default:
				serr, ok := err.(*sqlite.Error)
				if ok {
					switch serr.Code() {
					case sqlitelib.SQLITE_BUSY:
						continue
					}
					log.Printf("error in transaction: %d: %s", serr.Code(), serr)
				} else {
					log.Printf("error in transaction: %s", err)
				}
			}
			return err
		}
		err = tx.Commit()
		if err != nil {
			log.Printf("error committing transaction: %s", err)
			return err
		}
		break
	}
	return nil
}
