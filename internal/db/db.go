// Package db provides SQLite database operations for enva.
package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
}

// EnvVar represents a single environment variable record.
type EnvVar struct {
	Path      string
	Profile   string
	Key       string
	Value     string
	UpdatedAt time.Time
}

// EnvScope represents a scope record.
type EnvScope struct {
	Path      string
	Label     sql.NullString
	CreatedAt time.Time
}

// DefaultDBPath returns the default database path (~/.local/share/enva/enva.db).
func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "enva", "enva.db"), nil
}

// Open opens or creates the database at the given path.
func Open(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, err
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// migrate runs database migrations.
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS env_scopes (
		path TEXT PRIMARY KEY,
		label TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS env_vars (
		path TEXT NOT NULL,
		profile TEXT NOT NULL,
		key TEXT NOT NULL,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (path, profile, key)
	);

	CREATE INDEX IF NOT EXISTS idx_env_vars_path_profile ON env_vars(path, profile);
	`
	_, err := db.conn.Exec(schema)
	return err
}

// GetVarsForPaths retrieves all variables for the given paths and profile.
func (db *DB) GetVarsForPaths(paths []string, profile string) ([]EnvVar, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	// Build query with placeholders
	query := `SELECT path, profile, key, value, updated_at FROM env_vars WHERE profile = ? AND path IN (`
	args := []interface{}{profile}
	for i, p := range paths {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, p)
	}
	query += `) ORDER BY path, key`

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vars []EnvVar
	for rows.Next() {
		var v EnvVar
		if err := rows.Scan(&v.Path, &v.Profile, &v.Key, &v.Value, &v.UpdatedAt); err != nil {
			return nil, err
		}
		vars = append(vars, v)
	}
	return vars, rows.Err()
}

// GetVarsForPath retrieves all variables for a specific path and profile.
func (db *DB) GetVarsForPath(path, profile string) ([]EnvVar, error) {
	query := `SELECT path, profile, key, value, updated_at FROM env_vars
	          WHERE path = ? AND profile = ? ORDER BY key`
	rows, err := db.conn.Query(query, path, profile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vars []EnvVar
	for rows.Next() {
		var v EnvVar
		if err := rows.Scan(&v.Path, &v.Profile, &v.Key, &v.Value, &v.UpdatedAt); err != nil {
			return nil, err
		}
		vars = append(vars, v)
	}
	return vars, rows.Err()
}

// SetVar upserts a variable at the given path/profile/key.
func (db *DB) SetVar(path, profile, key, value string) error {
	// Ensure scope exists
	if err := db.ensureScope(path); err != nil {
		return err
	}

	query := `INSERT INTO env_vars (path, profile, key, value, updated_at)
	          VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	          ON CONFLICT(path, profile, key)
	          DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`
	_, err := db.conn.Exec(query, path, profile, key, value)
	return err
}

// DeleteVar deletes a variable at the given path/profile/key.
func (db *DB) DeleteVar(path, profile, key string) error {
	query := `DELETE FROM env_vars WHERE path = ? AND profile = ? AND key = ?`
	_, err := db.conn.Exec(query, path, profile, key)
	return err
}

// DeleteVarsForPath deletes all variables for a path and profile.
func (db *DB) DeleteVarsForPath(path, profile string) error {
	query := `DELETE FROM env_vars WHERE path = ? AND profile = ?`
	_, err := db.conn.Exec(query, path, profile)
	return err
}

// GetVar retrieves a specific variable.
func (db *DB) GetVar(path, profile, key string) (*EnvVar, error) {
	query := `SELECT path, profile, key, value, updated_at FROM env_vars
	          WHERE path = ? AND profile = ? AND key = ?`
	var v EnvVar
	err := db.conn.QueryRow(query, path, profile, key).Scan(&v.Path, &v.Profile, &v.Key, &v.Value, &v.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ensureScope creates a scope record if it doesn't exist.
func (db *DB) ensureScope(path string) error {
	query := `INSERT OR IGNORE INTO env_scopes (path, created_at) VALUES (?, CURRENT_TIMESTAMP)`
	_, err := db.conn.Exec(query, path)
	return err
}

// SetVarsBatch sets multiple variables in a transaction.
func (db *DB) SetVarsBatch(path, profile string, vars map[string]string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Ensure scope exists
	_, err = tx.Exec(`INSERT OR IGNORE INTO env_scopes (path, created_at) VALUES (?, CURRENT_TIMESTAMP)`, path)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT INTO env_vars (path, profile, key, value, updated_at)
	                         VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	                         ON CONFLICT(path, profile, key)
	                         DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for key, value := range vars {
		if _, err := stmt.Exec(path, profile, key, value); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteVarsBatch deletes multiple variables in a transaction.
func (db *DB) DeleteVarsBatch(path, profile string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`DELETE FROM env_vars WHERE path = ? AND profile = ? AND key = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, key := range keys {
		if _, err := stmt.Exec(path, profile, key); err != nil {
			return err
		}
	}

	return tx.Commit()
}
