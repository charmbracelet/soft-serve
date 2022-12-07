package fakedb

import (
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/types"
)

var _ db.Store = &FakeDB{}

// FakeDB is a fake database for testing.
type FakeDB struct{}

// GetConfig implements db.Store.
func (*FakeDB) GetConfig() (*types.Config, error) {
	return nil, nil
}

// SetConfigAllowKeyless implements db.Store.
func (*FakeDB) SetConfigAllowKeyless(bool) error {
	return nil
}

// SetConfigAnonAccess implements db.Store.
func (*FakeDB) SetConfigAnonAccess(string) error {
	return nil
}

// SetConfigHost implements db.Store.
func (*FakeDB) SetConfigHost(string) error {
	return nil
}

// SetConfigName implements db.Store.
func (*FakeDB) SetConfigName(string) error {
	return nil
}

// SetConfigPort implements db.Store.
func (*FakeDB) SetConfigPort(int) error {
	return nil
}

// AddUser implements db.Store.
func (*FakeDB) AddUser(name string, login string, email string, password string, isAdmin bool) error {
	return nil
}

// CountUsers implements db.Store.
func (*FakeDB) CountUsers() (int, error) {
	return 0, nil
}

// DeleteUser implements db.Store.
func (*FakeDB) DeleteUser(int) error {
	return nil
}

// GetUser implements db.Store.
func (*FakeDB) GetUser(int) (*types.User, error) {
	return nil, nil
}

// GetUserByEmail implements db.Store.
func (*FakeDB) GetUserByEmail(string) (*types.User, error) {
	return nil, nil
}

// GetUserByLogin implements db.Store.
func (*FakeDB) GetUserByLogin(string) (*types.User, error) {
	return nil, nil
}

// GetUserByPublicKey implements db.Store.
func (*FakeDB) GetUserByPublicKey(string) (*types.User, error) {
	return nil, nil
}

// SetUserAdmin implements db.Store.
func (*FakeDB) SetUserAdmin(*types.User, bool) error {
	return nil
}

// SetUserEmail implements db.Store.
func (*FakeDB) SetUserEmail(*types.User, string) error {
	return nil
}

// SetUserLogin implements db.Store.
func (*FakeDB) SetUserLogin(*types.User, string) error {
	return nil
}

// SetUserName implements db.Store.
func (*FakeDB) SetUserName(*types.User, string) error {
	return nil
}

// SetUserPassword implements db.Store.
func (*FakeDB) SetUserPassword(*types.User, string) error {
	return nil
}

// AddUserPublicKey implements db.Store.
func (*FakeDB) AddUserPublicKey(*types.User, string) error {
	return nil
}

// DeleteUserPublicKey implements db.Store.
func (*FakeDB) DeleteUserPublicKey(int) error {
	return nil
}

// GetUserPublicKeys implements db.Store.
func (*FakeDB) GetUserPublicKeys(*types.User) ([]*types.PublicKey, error) {
	return nil, nil
}

// AddRepo implements db.Store.
func (*FakeDB) AddRepo(name string, projectName string, description string, isPrivate bool) error {
	return nil
}

// DeleteRepo implements db.Store.
func (*FakeDB) DeleteRepo(string) error {
	return nil
}

// GetRepo implements db.Store.
func (*FakeDB) GetRepo(string) (*types.Repo, error) {
	return nil, nil
}

// SetRepoName implements db.Store.
func (*FakeDB) SetRepoName(string, string) error {
	return nil
}

// SetRepoDescription implements db.Store.
func (*FakeDB) SetRepoDescription(string, string) error {
	return nil
}

// SetRepoPrivate implements db.Store.
func (*FakeDB) SetRepoPrivate(string, bool) error {
	return nil
}

// SetRepoProjectName implements db.Store.
func (*FakeDB) SetRepoProjectName(string, string) error {
	return nil
}

// AddRepoCollab implements db.Store.
func (*FakeDB) AddRepoCollab(string, *types.User) error {
	return nil
}

// DeleteRepoCollab implements db.Store.
func (*FakeDB) DeleteRepoCollab(int, int) error {
	return nil
}

// ListRepoCollabs implements db.Store.
func (*FakeDB) ListRepoCollabs(string) ([]*types.User, error) {
	return nil, nil
}

// ListRepoPublicKeys implements db.Store.
func (*FakeDB) ListRepoPublicKeys(string) ([]*types.PublicKey, error) {
	return nil, nil
}

// IsRepoPublicKeyCollab implements db.Store.
func (*FakeDB) IsRepoPublicKeyCollab(string, string) (bool, error) {
	return false, nil
}

// Close implements db.Store.
func (*FakeDB) Close() error {
	return nil
}

// CreateDB implements db.Store.
func (*FakeDB) CreateDB() error {
	return nil
}
