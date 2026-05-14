package rbac

import "sync"

// UserStore manages user lookup by API key.
// In production, replace with a database-backed implementation.
type UserStore struct {
	mu    sync.RWMutex
	users map[string]*User // keyed by API key
}

// NewUserStore creates an empty user store.
func NewUserStore() *UserStore {
	return &UserStore{users: make(map[string]*User)}
}

// Register adds a user to the store, keyed by their API key.
func (s *UserStore) Register(user *User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.APIKey] = user
}

// LookupByAPIKey finds a user by their API key.
func (s *UserStore) LookupByAPIKey(key string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[key]
	return u, ok
}

// List returns all registered users (API keys are not included in JSON output).
func (s *UserStore) List() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, u)
	}
	return result
}
