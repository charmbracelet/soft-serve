package database

import (
	"testing"
)

func TestValidateBcryptHash(t *testing.T) {
	tests := []struct {
		name     string
		password  string
		wantErr  bool
	}{
		{
			name:     "valid bcrypt hash",
			password:  "$2a$10$gJMBzL9ojgSzRigSiuEEXuGqtm8lOgd3oSMh3QT/JayFJlcMDeLBu", // Valid bcrypt hash
			wantErr:  false,
		},
		{
			name:     "another valid bcrypt hash",
			password:  "$2a$10$w64SUzvDxuqVFawwyrCEm..up.DohSRXVXg1b7fPsab9OzojcLeCK", // Valid bcrypt hash
			wantErr:  false,
		},
		{
			name:     "plaintext password",
			password:  "mypassword123",
			wantErr:  true,
		},
		{
			name:     "short hash",
			password:  "$2a$10$abc",
			wantErr:  true,
		},
		{
			name:     "long hash",
			password:  "$2a$10$abcdefghijklmnopqrstuvwxyz1234567890123456789012345",
			wantErr:  true,
		},
		{
			name:     "wrong prefix",
			password:  "$1a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad9L2Z2Q",
			wantErr:  true,
		},
		{
			name:     "empty string",
			password:  "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBcryptHash(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBcryptHash() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetUserPasswordWithPlaintext(t *testing.T) {
	// This test verifies that storing plaintext passwords fails
	// In a real test, you would need a database connection
	// For now, we just verify the validation logic

	tests := []struct {
		name     string
		password  string
		wantErr  bool
	}{
		{
			name:     "valid bcrypt hash",
			password:  "$2a$10$gJMBzL9ojgSzRigSiuEEXuGqtm8lOgd3oSMh3QT/JayFJlcMDeLBu",
			wantErr:  false,
		},
		{
			name:     "plaintext password should fail",
			password:  "plaintext123",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBcryptHash(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBcryptHash(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}
