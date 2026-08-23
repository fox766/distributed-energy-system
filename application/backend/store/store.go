package store

import (
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

// UserRecord holds local auth data for a user.
type UserRecord struct {
	UserID       string `json:"userid"`
	PasswordHash string `json:"passwordhash"`
	Role         string `json:"role"`
	// DeviceType is the generation device a PRODUCER registered with; the
	// generation scheduler uses it instead of rotating through device types.
	DeviceType string `json:"devicetype"`
}

// DefaultDeviceType is used for rows predating the device_type column and for
// accounts that never generate (admin, consumers).
const DefaultDeviceType = "SOLAR_PANEL"

// CredentialStore persists user credentials in SQLite.
type CredentialStore struct {
	mu sync.RWMutex
	db *sql.DB
}

// New creates or opens a SQLite credential store.
func New(dbPath string) (*CredentialStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode for better concurrent access
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA foreign_keys=ON")

	cs := &CredentialStore{db: db}
	if err := cs.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return cs, nil
}

func (cs *CredentialStore) migrate() error {
	_, err := cs.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			username     TEXT PRIMARY KEY,
			user_id      TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role         TEXT NOT NULL DEFAULT 'CONSUMER',
			created_at   TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_users_user_id ON users(user_id);
	`)
	if err != nil {
		return err
	}
	return cs.addColumnIfMissing("device_type",
		"ALTER TABLE users ADD COLUMN device_type TEXT NOT NULL DEFAULT '"+DefaultDeviceType+"'")
}

// addColumnIfMissing runs ddl only when the column is absent, so that databases
// created before the column was introduced upgrade in place. SQLite has no
// "ADD COLUMN IF NOT EXISTS", hence the PRAGMA probe.
func (cs *CredentialStore) addColumnIfMissing(column, ddl string) error {
	rows, err := cs.db.Query("PRAGMA table_info(users)")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = cs.db.Exec(ddl)
	return err
}

// Close closes the database connection.
func (cs *CredentialStore) Close() error {
	return cs.db.Close()
}

// CreateUser stores username + bcrypt hash + userID. Returns error if username exists.
func (cs *CredentialStore) CreateUser(username, password, userID, role, deviceType string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if deviceType == "" {
		deviceType = DefaultDeviceType
	}

	_, err = cs.db.Exec(
		"INSERT INTO users (username, user_id, password_hash, role, device_type) VALUES (?, ?, ?, ?, ?)",
		username, userID, string(hash), role, deviceType,
	)
	if err != nil {
		if isConstraintError(err) {
			return fmt.Errorf("username already exists")
		}
		return err
	}
	return nil
}

// VerifyPassword checks whether the password matches the stored hash for username.
func (cs *CredentialStore) VerifyPassword(username, password string) error {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var hash string
	err := cs.db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&hash)
	if err == sql.ErrNoRows {
		return fmt.Errorf("username not found")
	}
	if err != nil {
		return err
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GetUserRecord returns the stored record for a username.
func (cs *CredentialStore) GetUserRecord(username string) (*UserRecord, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var rec UserRecord
	err := cs.db.QueryRow(
		"SELECT user_id, password_hash, role, device_type FROM users WHERE username = ?", username,
	).Scan(&rec.UserID, &rec.PasswordHash, &rec.Role, &rec.DeviceType)
	if err != nil {
		return nil, false
	}
	return &rec, true
}

// Exists checks whether a username is registered.
func (cs *CredentialStore) Exists(username string) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var count int
	cs.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	return count > 0
}

// GetAllUsers returns a copy of all user records, keyed by username.
func (cs *CredentialStore) GetAllUsers() map[string]*UserRecord {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make(map[string]*UserRecord)
	rows, err := cs.db.Query("SELECT username, user_id, password_hash, role, device_type FROM users")
	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var username string
		var rec UserRecord
		if err := rows.Scan(&username, &rec.UserID, &rec.PasswordHash, &rec.Role, &rec.DeviceType); err == nil {
			result[username] = &rec
		}
	}
	return result
}

func isConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "UNIQUE constraint") ||
		contains(msg, "PRIMARY KEY") ||
		contains(msg, "constraint failed")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && search(s, sub)
}

func search(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
