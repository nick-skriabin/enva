// Package path provides path resolution utilities for enva.
// It handles canonicalization and root boundary discovery.
package path

import (
	"os"
	"path/filepath"
)

// Canonicalize returns the absolute, symlink-resolved path.
func Canonicalize(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

// FindRoot walks up from the given path to find the root boundary.
// Priority: .enva file (closest) > .git directory (closest) > filesystem root
func FindRoot(from string) (string, error) {
	canonical, err := Canonicalize(from)
	if err != nil {
		return "", err
	}

	current := canonical
	for {
		// Check for .enva marker file
		envaMarker := filepath.Join(current, ".enva")
		if info, err := os.Stat(envaMarker); err == nil && !info.IsDir() {
			return current, nil
		}

		// Check for .git directory
		gitDir := filepath.Join(current, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return current, nil
		}

		// Move to parent
		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			return current, nil
		}
		current = parent
	}
}

// BuildChain builds the path chain from rootDir to targetDir (inclusive).
// Returns paths in ascending order: [rootDir, ..., targetDir]
func BuildChain(rootDir, targetDir string) ([]string, error) {
	rootCanon, err := Canonicalize(rootDir)
	if err != nil {
		return nil, err
	}
	targetCanon, err := Canonicalize(targetDir)
	if err != nil {
		return nil, err
	}

	// Build chain by walking up from target to root
	var chain []string
	current := targetCanon
	for {
		chain = append([]string{current}, chain...)
		if current == rootCanon {
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			// Safety: shouldn't happen if rootDir is ancestor of targetDir
			break
		}
		current = parent
	}

	return chain, nil
}

// IsAncestor checks if ancestor is an ancestor of (or equal to) path.
func IsAncestor(ancestor, path string) bool {
	ancestorCanon, err := Canonicalize(ancestor)
	if err != nil {
		return false
	}
	pathCanon, err := Canonicalize(path)
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(ancestorCanon, pathCanon)
	if err != nil {
		return false
	}

	// If relative path starts with "..", ancestor is not an ancestor
	if rel == ".." || (len(rel) >= 2 && rel[:2] == "..") {
		return false
	}
	return true
}
