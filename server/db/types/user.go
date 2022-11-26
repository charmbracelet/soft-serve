package types

import (
	"net/mail"
	"time"
)

// User is a user database model.
type User struct {
	ID        int
	Name      string
	Login     *string
	Email     *string
	Password  *string
	Admin     bool
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

// Address returns the email address of the user.
func (u *User) Address() *mail.Address {
	if u.Email == nil {
		return nil
	}
	return &mail.Address{
		Name:    u.Name,
		Address: *u.Email,
	}
}

// String returns the name of the user.
func (u *User) String() string {
	return u.Name
}
