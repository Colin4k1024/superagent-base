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

// LookupByID finds a user by their ID.
func (s *UserStore) LookupByID(id string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.ID == id {
			return u, true
		}
	}
	return nil, false
}

// Remove deletes a user by ID from the store. Returns false if not found.
func (s *UserStore) Remove(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, u := range s.users {
		if u.ID == id {
			delete(s.users, key)
			return true
		}
	}
	return false
}

// Update applies partial changes to a user identified by ID.
// Only non-zero fields in patch are applied. Returns false if not found.
func (s *UserStore) Update(id string, role Role, disabled *bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.ID == id {
			if role != "" {
				u.Role = role
			}
			if disabled != nil {
				u.Disabled = *disabled
			}
			return true
		}
	}
	return false
}
