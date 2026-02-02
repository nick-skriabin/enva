package path

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalize(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "enva-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Get canonical form of tmpDir
	tmpDirCanon, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("Failed to eval symlinks: %v", err)
	}

	t.Run("absolute path", func(t *testing.T) {
		got, err := Canonicalize(tmpDirCanon)
		if err != nil {
			t.Errorf("Canonicalize failed: %v", err)
		}
		if got != tmpDirCanon {
			t.Errorf("Canonicalize(%q) = %q, want %q", tmpDirCanon, got, tmpDirCanon)
		}
	})

	t.Run("relative path", func(t *testing.T) {
		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)

		os.Chdir(tmpDirCanon)
		got, err := Canonicalize(".")
		if err != nil {
			t.Errorf("Canonicalize failed: %v", err)
		}
		if got != tmpDirCanon {
			t.Errorf("Canonicalize('.') = %q, want %q", got, tmpDirCanon)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		realDir := filepath.Join(tmpDirCanon, "real")
		linkDir := filepath.Join(tmpDirCanon, "link")

		if err := os.Mkdir(realDir, 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.Symlink(realDir, linkDir); err != nil {
			t.Fatalf("Failed to create symlink: %v", err)
		}

		got, err := Canonicalize(linkDir)
		if err != nil {
			t.Errorf("Canonicalize failed: %v", err)
		}
		if got != realDir {
			t.Errorf("Canonicalize(%q) = %q, want %q", linkDir, got, realDir)
		}
	})
}

func TestFindRoot(t *testing.T) {
	// Create a temp directory structure for testing
	tmpDir, err := os.MkdirTemp("", "enva-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpDirCanon, _ := filepath.EvalSymlinks(tmpDir)

	t.Run("finds .enva marker", func(t *testing.T) {
		// Create structure: root/.enva, root/sub/subsub
		root := filepath.Join(tmpDirCanon, "enva-root")
		sub := filepath.Join(root, "sub", "subsub")

		os.MkdirAll(sub, 0755)
		os.WriteFile(filepath.Join(root, ".enva"), []byte{}, 0644)

		got, err := FindRoot(sub)
		if err != nil {
			t.Errorf("FindRoot failed: %v", err)
		}
		if got != root {
			t.Errorf("FindRoot(%q) = %q, want %q", sub, got, root)
		}
	})

	t.Run("finds .git directory", func(t *testing.T) {
		// Create structure: root/.git, root/sub/subsub
		root := filepath.Join(tmpDirCanon, "git-root")
		sub := filepath.Join(root, "sub", "subsub")
		gitDir := filepath.Join(root, ".git")

		os.MkdirAll(sub, 0755)
		os.MkdirAll(gitDir, 0755)

		got, err := FindRoot(sub)
		if err != nil {
			t.Errorf("FindRoot failed: %v", err)
		}
		if got != root {
			t.Errorf("FindRoot(%q) = %q, want %q", sub, got, root)
		}
	})

	t.Run(".enva takes priority over .git", func(t *testing.T) {
		// Create structure: gitroot/.git, gitroot/envaroot/.enva, gitroot/envaroot/sub
		gitRoot := filepath.Join(tmpDirCanon, "priority-git")
		envaRoot := filepath.Join(gitRoot, "envaroot")
		sub := filepath.Join(envaRoot, "sub")

		os.MkdirAll(sub, 0755)
		os.MkdirAll(filepath.Join(gitRoot, ".git"), 0755)
		os.WriteFile(filepath.Join(envaRoot, ".enva"), []byte{}, 0644)

		got, err := FindRoot(sub)
		if err != nil {
			t.Errorf("FindRoot failed: %v", err)
		}
		if got != envaRoot {
			t.Errorf("FindRoot(%q) = %q, want %q (should find .enva before .git)", sub, got, envaRoot)
		}
	})

	t.Run("falls back to filesystem root", func(t *testing.T) {
		// Create an isolated directory with no markers
		isolated := filepath.Join(tmpDirCanon, "isolated", "sub", "subsub")
		os.MkdirAll(isolated, 0755)

		got, err := FindRoot(isolated)
		if err != nil {
			t.Errorf("FindRoot failed: %v", err)
		}
		// Should eventually reach filesystem root
		if got == "" {
			t.Errorf("FindRoot(%q) returned empty string", isolated)
		}
	})
}

func TestBuildChain(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "enva-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpDirCanon, _ := filepath.EvalSymlinks(tmpDir)

	// Create structure: root/a/b/c
	root := filepath.Join(tmpDirCanon, "root")
	a := filepath.Join(root, "a")
	b := filepath.Join(a, "b")
	c := filepath.Join(b, "c")
	os.MkdirAll(c, 0755)

	t.Run("builds complete chain", func(t *testing.T) {
		chain, err := BuildChain(root, c)
		if err != nil {
			t.Errorf("BuildChain failed: %v", err)
		}

		expected := []string{root, a, b, c}
		if len(chain) != len(expected) {
			t.Errorf("BuildChain returned %d items, want %d", len(chain), len(expected))
			return
		}

		for i, want := range expected {
			if chain[i] != want {
				t.Errorf("BuildChain[%d] = %q, want %q", i, chain[i], want)
			}
		}
	})

	t.Run("same root and target", func(t *testing.T) {
		chain, err := BuildChain(root, root)
		if err != nil {
			t.Errorf("BuildChain failed: %v", err)
		}

		if len(chain) != 1 || chain[0] != root {
			t.Errorf("BuildChain(root, root) = %v, want [%q]", chain, root)
		}
	})
}

func TestIsAncestor(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "enva-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpDirCanon, _ := filepath.EvalSymlinks(tmpDir)

	root := filepath.Join(tmpDirCanon, "root")
	child := filepath.Join(root, "child")
	grandchild := filepath.Join(child, "grandchild")
	sibling := filepath.Join(tmpDirCanon, "sibling")

	os.MkdirAll(grandchild, 0755)
	os.MkdirAll(sibling, 0755)

	tests := []struct {
		ancestor string
		path     string
		expected bool
	}{
		{root, child, true},
		{root, grandchild, true},
		{root, root, true},
		{child, grandchild, true},

		{child, root, false},
		{grandchild, root, false},
		{sibling, child, false},
		{root, sibling, false},
	}

	for _, tt := range tests {
		name := filepath.Base(tt.ancestor) + "_ancestor_of_" + filepath.Base(tt.path)
		t.Run(name, func(t *testing.T) {
			got := IsAncestor(tt.ancestor, tt.path)
			if got != tt.expected {
				t.Errorf("IsAncestor(%q, %q) = %v, want %v", tt.ancestor, tt.path, got, tt.expected)
			}
		})
	}
}
