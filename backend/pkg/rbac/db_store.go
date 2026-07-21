package rbac

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// UserStorer is the interface for admin user storage operations.
// Both the in-memory UserStore and the DB-backed DBUserStore implement it.
type UserStorer interface {
	Register(user *User)
	LookupByAPIKey(key string) (*User, bool)
	List() []*User
	LookupByID(id string) (*User, bool)
	Remove(id string) bool
	Update(id string, role Role, disabled *bool) bool
}

// adminUserRecord is the GORM model for the admin_user table.
// Kept separate from the platform's user table to avoid coupling.
type adminUserRecord struct {
	ID        string         `gorm:"column:id;primaryKey;size:64"`
	Name      string         `gorm:"column:name;not null;size:255"`
	Email     string         `gorm:"column:email;not null;default:'';size:255"`
	APIKey    string         `gorm:"column:api_key;not null;uniqueIndex;size:255"`
	Role      string         `gorm:"column:role;not null;default:'viewer';size:64"`
	Disabled  bool           `gorm:"column:disabled;not null;default:false"`
	CreatedAt int64          `gorm:"column:created_at;not null;autoCreateTime:milli"`
	UpdatedAt int64          `gorm:"column:updated_at;not null;autoUpdateTime:milli"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (adminUserRecord) TableName() string { return "admin_user" }

// DBUserStore is a database-backed implementation of UserStorer.
type DBUserStore struct {
	db *gorm.DB
}

// NewDBUserStore creates a DBUserStore and auto-migrates the admin_user table.
// Returns an error if migration fails.
func NewDBUserStore(db *gorm.DB) (*DBUserStore, error) {
	if err := db.AutoMigrate(&adminUserRecord{}); err != nil {
		return nil, fmt.Errorf("admin_user auto-migrate: %w", err)
	}
	return &DBUserStore{db: db}, nil
}

// Register upserts a user keyed by their API key.
func (s *DBUserStore) Register(user *User) {
	rec := adminUserRecord{
		ID:       user.ID,
		Name:     user.Name,
		Email:    user.Email,
		APIKey:   user.APIKey,
		Role:     string(user.Role),
		Disabled: user.Disabled,
	}
	// FirstOrCreate on api_key; if it already exists, update the row.
	var existing adminUserRecord
	result := s.db.Where("api_key = ?", user.APIKey).First(&existing)
	if result.Error == gorm.ErrRecordNotFound {
		s.db.Create(&rec)
	} else if result.Error == nil {
		s.db.Model(&existing).Updates(map[string]interface{}{
			"name":       rec.Name,
			"email":      rec.Email,
			"role":       rec.Role,
			"disabled":   rec.Disabled,
			"updated_at": time.Now().UnixMilli(),
		})
	}
}

// LookupByAPIKey finds a user by API key.
func (s *DBUserStore) LookupByAPIKey(key string) (*User, bool) {
	var rec adminUserRecord
	if err := s.db.Where("api_key = ?", key).First(&rec).Error; err != nil {
		return nil, false
	}
	return recordToRBACUser(&rec), true
}

// List returns all non-deleted admin users.
func (s *DBUserStore) List() []*User {
	var recs []adminUserRecord
	s.db.Find(&recs)
	users := make([]*User, 0, len(recs))
	for i := range recs {
		users = append(users, recordToRBACUser(&recs[i]))
	}
	return users
}

// LookupByID finds a user by string ID.
func (s *DBUserStore) LookupByID(id string) (*User, bool) {
	var rec adminUserRecord
	if err := s.db.Where("id = ?", id).First(&rec).Error; err != nil {
		return nil, false
	}
	return recordToRBACUser(&rec), true
}

// Remove soft-deletes a user by ID. Returns false if not found.
func (s *DBUserStore) Remove(id string) bool {
	result := s.db.Where("id = ?", id).Delete(&adminUserRecord{})
	return result.RowsAffected > 0
}

// Update applies partial changes to the user identified by ID.
func (s *DBUserStore) Update(id string, role Role, disabled *bool) bool {
	updates := map[string]interface{}{
		"updated_at": time.Now().UnixMilli(),
	}
	if role != "" {
		updates["role"] = string(role)
	}
	if disabled != nil {
		updates["disabled"] = *disabled
	}
	result := s.db.Model(&adminUserRecord{}).Where("id = ?", id).Updates(updates)
	return result.RowsAffected > 0
}

func recordToRBACUser(rec *adminUserRecord) *User {
	return &User{
		ID:        rec.ID,
		Name:      rec.Name,
		Email:     rec.Email,
		APIKey:    rec.APIKey,
		Role:      Role(rec.Role),
		Disabled:  rec.Disabled,
		CreatedAt: time.UnixMilli(rec.CreatedAt),
	}
}
