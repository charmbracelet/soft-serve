package store

// Store is an interface for managing repositories, users, and settings.
type Store interface {
	RepositoryStore
	UserStore
	OrgStore
	TeamStore
	CollaboratorStore
	SettingStore
	LFSStore
	AccessTokenStore
	WebhookStore
	HandleStore
}
