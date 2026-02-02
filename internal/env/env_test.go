package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nick-skriabin/enva/internal/db"
)

func setupTestEnv(t *testing.T) (*db.DB, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "enva-env-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create canonical path
	tmpDirCanon, _ := filepath.EvalSymlinks(tmpDir)

	dbPath := filepath.Join(tmpDirCanon, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to open database: %v", err)
	}

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return database, tmpDirCanon, cleanup
}

func TestNewResolver(t *testing.T) {
	database, _, cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("with profile", func(t *testing.T) {
		r := NewResolver(database, "production")
		if r.GetProfile() != "production" {
			t.Errorf("GetProfile() = %q, want 'production'", r.GetProfile())
		}
	})

	t.Run("empty profile uses default", func(t *testing.T) {
		r := NewResolver(database, "")
		if r.GetProfile() != DefaultProfile {
			t.Errorf("GetProfile() = %q, want %q", r.GetProfile(), DefaultProfile)
		}
	})
}

func TestGetProfileFromEnv(t *testing.T) {
	t.Run("returns env var when set", func(t *testing.T) {
		os.Setenv("ENVA_PROFILE", "staging")
		defer os.Unsetenv("ENVA_PROFILE")

		got := GetProfileFromEnv()
		if got != "staging" {
			t.Errorf("GetProfileFromEnv() = %q, want 'staging'", got)
		}
	})

	t.Run("returns default when not set", func(t *testing.T) {
		os.Unsetenv("ENVA_PROFILE")

		got := GetProfileFromEnv()
		if got != DefaultProfile {
			t.Errorf("GetProfileFromEnv() = %q, want %q", got, DefaultProfile)
		}
	})
}

func TestResolverSetAndDelete(t *testing.T) {
	database, tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create test directory
	testDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(testDir, 0755)

	resolver := NewResolver(database, "default")

	// Set variable
	err := resolver.SetVar(testDir, "API_KEY", "secret", "test description")
	if err != nil {
		t.Fatalf("SetVar failed: %v", err)
	}

	// Verify via GetLocalVarsFromDB
	vars, err := resolver.GetLocalVarsFromDB(testDir)
	if err != nil {
		t.Fatalf("GetLocalVarsFromDB failed: %v", err)
	}
	if len(vars) != 1 {
		t.Errorf("GetLocalVarsFromDB returned %d vars, want 1", len(vars))
	}
	if vars[0].Key != "API_KEY" || vars[0].Value != "secret" {
		t.Errorf("GetLocalVarsFromDB[0] = {%q, %q}, want {'API_KEY', 'secret'}", vars[0].Key, vars[0].Value)
	}

	// Delete variable
	err = resolver.DeleteVar(testDir, "API_KEY")
	if err != nil {
		t.Fatalf("DeleteVar failed: %v", err)
	}

	// Verify deleted
	vars, _ = resolver.GetLocalVarsFromDB(testDir)
	if len(vars) != 0 {
		t.Errorf("After DeleteVar: %d vars, want 0", len(vars))
	}
}

func TestResolveInheritance(t *testing.T) {
	database, tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create directory structure with .enva marker at root
	root := filepath.Join(tmpDir, "project")
	child := filepath.Join(root, "child")
	grandchild := filepath.Join(child, "grandchild")

	os.MkdirAll(grandchild, 0755)
	os.WriteFile(filepath.Join(root, ".enva"), []byte{}, 0644)

	resolver := NewResolver(database, "default")

	// Set variables at different levels
	resolver.SetVar(root, "ROOT_VAR", "root_value", "")
	resolver.SetVar(root, "SHARED", "from_root", "")
	resolver.SetVar(child, "CHILD_VAR", "child_value", "")
	resolver.SetVar(child, "SHARED", "from_child", "") // Override
	resolver.SetVar(grandchild, "GRANDCHILD_VAR", "grandchild_value", "")

	// Resolve at grandchild level
	ctx, err := resolver.Resolve(grandchild)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Should have all 4 unique vars
	if len(ctx.Resolved) != 4 {
		t.Errorf("Resolved has %d vars, want 4", len(ctx.Resolved))
	}

	// Check inheritance
	tests := []struct {
		key   string
		value string
	}{
		{"ROOT_VAR", "root_value"},
		{"CHILD_VAR", "child_value"},
		{"GRANDCHILD_VAR", "grandchild_value"},
		{"SHARED", "from_child"}, // Should be overridden value
	}

	for _, tt := range tests {
		v, ok := ctx.Resolved[tt.key]
		if !ok {
			t.Errorf("Resolved missing key %q", tt.key)
			continue
		}
		if v.Value != tt.value {
			t.Errorf("Resolved[%q].Value = %q, want %q", tt.key, v.Value, tt.value)
		}
	}

	// Check override tracking
	shared := ctx.Resolved["SHARED"]
	if !shared.Overrode {
		t.Error("SHARED should have Overrode=true")
	}
	if shared.OverrodePath == "" {
		t.Error("SHARED should have OverrodePath set")
	}
}

func TestResolveContextGetSortedVars(t *testing.T) {
	ctx := &ResolveContext{
		Resolved: map[string]*ResolvedVar{
			"ZEBRA": {Key: "ZEBRA", Value: "z"},
			"ALPHA": {Key: "ALPHA", Value: "a"},
			"MIDDLE": {Key: "MIDDLE", Value: "m"},
		},
	}

	vars := ctx.GetSortedVars()

	if len(vars) != 3 {
		t.Fatalf("GetSortedVars returned %d vars, want 3", len(vars))
	}

	expected := []string{"ALPHA", "MIDDLE", "ZEBRA"}
	for i, want := range expected {
		if vars[i].Key != want {
			t.Errorf("GetSortedVars[%d].Key = %q, want %q", i, vars[i].Key, want)
		}
	}
}

func TestResolveContextGetLocalVars(t *testing.T) {
	cwdReal := "/project/child"
	ctx := &ResolveContext{
		CwdReal: cwdReal,
		Resolved: map[string]*ResolvedVar{
			"LOCAL": {Key: "LOCAL", Value: "local", DefinedAtPath: cwdReal},
			"INHERITED": {Key: "INHERITED", Value: "inherited", DefinedAtPath: "/project"},
		},
	}

	vars := ctx.GetLocalVars()

	if len(vars) != 1 {
		t.Fatalf("GetLocalVars returned %d vars, want 1", len(vars))
	}

	if vars[0].Key != "LOCAL" {
		t.Errorf("GetLocalVars[0].Key = %q, want 'LOCAL'", vars[0].Key)
	}
}

func TestResolveContextIsLocal(t *testing.T) {
	cwdReal := "/project/child"
	ctx := &ResolveContext{CwdReal: cwdReal}

	local := &ResolvedVar{Key: "LOCAL", DefinedAtPath: cwdReal}
	inherited := &ResolvedVar{Key: "INHERITED", DefinedAtPath: "/project"}

	if !ctx.IsLocal(local) {
		t.Error("IsLocal should return true for local var")
	}

	if ctx.IsLocal(inherited) {
		t.Error("IsLocal should return false for inherited var")
	}
}

func TestSyncLocalVars(t *testing.T) {
	database, tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	testDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(testDir, 0755)

	resolver := NewResolver(database, "default")

	// Set initial vars
	resolver.SetVar(testDir, "KEEP", "keep_value", "")
	resolver.SetVar(testDir, "UPDATE", "old_value", "")
	resolver.SetVar(testDir, "DELETE", "delete_value", "")

	// Sync with new state
	newVars := map[string]db.VarData{
		"KEEP":   {Value: "keep_value", Description: ""},   // Keep same
		"UPDATE": {Value: "new_value", Description: ""},    // Update
		"NEW":    {Value: "new_var", Description: ""},      // Add
		// DELETE is missing, should be deleted
	}

	err := resolver.SyncLocalVars(testDir, newVars)
	if err != nil {
		t.Fatalf("SyncLocalVars failed: %v", err)
	}

	// Verify state
	vars, _ := resolver.GetLocalVarsFromDB(testDir)
	if len(vars) != 3 {
		t.Errorf("After sync: %d vars, want 3", len(vars))
	}

	varMap := make(map[string]string)
	for _, v := range vars {
		varMap[v.Key] = v.Value
	}

	if varMap["KEEP"] != "keep_value" {
		t.Errorf("KEEP = %q, want 'keep_value'", varMap["KEEP"])
	}
	if varMap["UPDATE"] != "new_value" {
		t.Errorf("UPDATE = %q, want 'new_value'", varMap["UPDATE"])
	}
	if varMap["NEW"] != "new_var" {
		t.Errorf("NEW = %q, want 'new_var'", varMap["NEW"])
	}
	if _, exists := varMap["DELETE"]; exists {
		t.Error("DELETE should have been removed")
	}
}

func TestSetVarsBatch(t *testing.T) {
	database, tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	testDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(testDir, 0755)

	resolver := NewResolver(database, "default")

	vars := map[string]db.VarData{
		"KEY1": {Value: "value1", Description: ""},
		"KEY2": {Value: "value2", Description: ""},
	}

	err := resolver.SetVarsBatch(testDir, vars)
	if err != nil {
		t.Fatalf("SetVarsBatch failed: %v", err)
	}

	dbVars, _ := resolver.GetLocalVarsFromDB(testDir)
	if len(dbVars) != 2 {
		t.Errorf("SetVarsBatch set %d vars, want 2", len(dbVars))
	}
}

func TestDeleteVarsBatch(t *testing.T) {
	database, tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	testDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(testDir, 0755)

	resolver := NewResolver(database, "default")

	// Set multiple vars
	resolver.SetVar(testDir, "KEY1", "value1", "")
	resolver.SetVar(testDir, "KEY2", "value2", "")
	resolver.SetVar(testDir, "KEY3", "value3", "")

	// Delete batch
	err := resolver.DeleteVarsBatch(testDir, []string{"KEY1", "KEY3"})
	if err != nil {
		t.Fatalf("DeleteVarsBatch failed: %v", err)
	}

	vars, _ := resolver.GetLocalVarsFromDB(testDir)
	if len(vars) != 1 {
		t.Errorf("After DeleteVarsBatch: %d vars, want 1", len(vars))
	}
	if vars[0].Key != "KEY2" {
		t.Errorf("Remaining var = %q, want 'KEY2'", vars[0].Key)
	}
}
