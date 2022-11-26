package types

import "time"

// Config is the Soft Serve application configuration.
type Config struct {
	ID           int
	Name         string
	Host         string
	Port         int
	AnonAccess   string
	AllowKeyless bool
	CreatedAt    *time.Time
	UpdatedAt    *time.Time
}
