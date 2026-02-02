package db

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "enva-db-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestOpenAndClose(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if db == nil {
		t.Fatal("Database should not be nil")
	}
}

func TestSetAndGetVar(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	path := "/test/path"
	profile := "default"
	key := "API_KEY"
	value := "secret123"

	// Set variable
	err := db.SetVar(path, profile, key, value, "")
	if err != nil {
		t.Fatalf("SetVar failed: %v", err)
	}

	// Get variable
	v, err := db.GetVar(path, profile, key)
	if err != nil {
		t.Fatalf("GetVar failed: %v", err)
	}

	if v == nil {
		t.Fatal("GetVar returned nil")
	}
	if v.Key != key {
		t.Errorf("GetVar Key = %q, want %q", v.Key, key)
	}
	if v.Value != value {
		t.Errorf("GetVar Value = %q, want %q", v.Value, value)
	}
	if v.Path != path {
		t.Errorf("GetVar Path = %q, want %q", v.Path, path)
	}
	if v.Profile != profile {
		t.Errorf("GetVar Profile = %q, want %q", v.Profile, profile)
	}
}

func TestSetVarUpsert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	path := "/test/path"
	profile := "default"
	key := "KEY"

	// Set initial value
	db.SetVar(path, profile, key, "first", "")

	// Update value
	db.SetVar(path, profile, key, "second", "")

	// Get value
	v, _ := db.GetVar(path, profile, key)
	if v.Value != "second" {
		t.Errorf("Upsert failed: got %q, want 'second'", v.Value)
	}
}

func TestGetVarNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	v, err := db.GetVar("/nonexistent", "default", "KEY")
	if err != nil {
		t.Fatalf("GetVar failed: %v", err)
	}
	if v != nil {
		t.Error("GetVar should return nil for nonexistent key")
	}
}

func TestDeleteVar(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	path := "/test/path"
	profile := "default"
	key := "KEY"

	// Set and then delete
	db.SetVar(path, profile, key, "value", "")
	err := db.DeleteVar(path, profile, key)
	if err != nil {
		t.Fatalf("DeleteVar failed: %v", err)
	}

	// Verify deleted
	v, _ := db.GetVar(path, profile, key)
	if v != nil {
		t.Error("Variable should be deleted")
	}
}

func TestGetVarsForPath(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	path := "/test/path"
	profile := "default"

	// Set multiple variables
	db.SetVar(path, profile, "KEY1", "value1", "")
	db.SetVar(path, profile, "KEY2", "value2", "")
	db.SetVar(path, profile, "KEY3", "value3", "")

	// Also set for different path
	db.SetVar("/other/path", profile, "OTHER", "other", "")

	vars, err := db.GetVarsForPath(path, profile)
	if err != nil {
		t.Fatalf("GetVarsForPath failed: %v", err)
	}

	if len(vars) != 3 {
		t.Errorf("GetVarsForPath returned %d vars, want 3", len(vars))
	}
}

func TestGetVarsForPaths(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	profile := "default"

	// Set variables at different paths
	db.SetVar("/root", profile, "ROOT_VAR", "root", "")
	db.SetVar("/root/child", profile, "CHILD_VAR", "child", "")
	db.SetVar("/root/child/grandchild", profile, "GRANDCHILD_VAR", "grandchild", "")
	db.SetVar("/other", profile, "OTHER_VAR", "other", "")

	paths := []string{"/root", "/root/child", "/root/child/grandchild"}
	vars, err := db.GetVarsForPaths(paths, profile)
	if err != nil {
		t.Fatalf("GetVarsForPaths failed: %v", err)
	}

	if len(vars) != 3 {
		t.Errorf("GetVarsForPaths returned %d vars, want 3", len(vars))
	}

	// Verify OTHER_VAR is not included
	for _, v := range vars {
		if v.Key == "OTHER_VAR" {
			t.Error("GetVarsForPaths should not include vars from /other")
		}
	}
}

func TestGetVarsForPathsEmpty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	vars, err := db.GetVarsForPaths([]string{}, "default")
	if err != nil {
		t.Fatalf("GetVarsForPaths failed: %v", err)
	}
	if vars != nil {
		t.Errorf("GetVarsForPaths([]) should return nil, got %v", vars)
	}
}

func TestProfileIsolation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	path := "/test/path"

	// Set same key in different profiles
	db.SetVar(path, "default", "KEY", "default_value", "")
	db.SetVar(path, "production", "KEY", "prod_value", "")

	// Get from each profile
	defaultVar, _ := db.GetVar(path, "default", "KEY")
	prodVar, _ := db.GetVar(path, "production", "KEY")

	if defaultVar.Value != "default_value" {
		t.Errorf("default profile value = %q, want 'default_value'", defaultVar.Value)
	}
	if prodVar.Value != "prod_value" {
		t.Errorf("production profile value = %q, want 'prod_value'", prodVar.Value)
	}
}

func TestSetVarsBatch(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	path := "/test/path"
	profile := "default"
	vars := map[string]VarData{
		"KEY1": {Value: "value1", Description: "desc1"},
		"KEY2": {Value: "value2", Description: ""},
		"KEY3": {Value: "value3", Description: "desc3"},
	}

	err := db.SetVarsBatch(path, profile, vars)
	if err != nil {
		t.Fatalf("SetVarsBatch failed: %v", err)
	}

	// Verify all vars were set
	dbVars, _ := db.GetVarsForPath(path, profile)
	if len(dbVars) != 3 {
		t.Errorf("SetVarsBatch set %d vars, want 3", len(dbVars))
	}
}

func TestDeleteVarsBatch(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	path := "/test/path"
	profile := "default"

	// Set multiple vars
	db.SetVar(path, profile, "KEY1", "value1", "")
	db.SetVar(path, profile, "KEY2", "value2", "")
	db.SetVar(path, profile, "KEY3", "value3", "")

	// Delete two of them
	err := db.DeleteVarsBatch(path, profile, []string{"KEY1", "KEY3"})
	if err != nil {
		t.Fatalf("DeleteVarsBatch failed: %v", err)
	}

	// Verify only KEY2 remains
	vars, _ := db.GetVarsForPath(path, profile)
	if len(vars) != 1 {
		t.Errorf("After DeleteVarsBatch: %d vars, want 1", len(vars))
	}
	if vars[0].Key != "KEY2" {
		t.Errorf("Remaining var = %q, want 'KEY2'", vars[0].Key)
	}
}

func TestDeleteVarsBatchEmpty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Should not error on empty list
	err := db.DeleteVarsBatch("/path", "default", []string{})
	if err != nil {
		t.Errorf("DeleteVarsBatch([]) failed: %v", err)
	}
}

func TestDeleteVarsForPath(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	path := "/test/path"
	profile := "default"

	db.SetVar(path, profile, "KEY1", "value1", "")
	db.SetVar(path, profile, "KEY2", "value2", "")
	db.SetVar("/other", profile, "OTHER", "other", "")

	err := db.DeleteVarsForPath(path, profile)
	if err != nil {
		t.Fatalf("DeleteVarsForPath failed: %v", err)
	}

	// Path should have no vars
	vars, _ := db.GetVarsForPath(path, profile)
	if len(vars) != 0 {
		t.Errorf("After DeleteVarsForPath: %d vars, want 0", len(vars))
	}

	// Other path should still have vars
	otherVars, _ := db.GetVarsForPath("/other", profile)
	if len(otherVars) != 1 {
		t.Errorf("Other path should still have 1 var, got %d", len(otherVars))
	}
}
