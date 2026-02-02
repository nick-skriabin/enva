// Package env provides environment variable resolution with inheritance.
package env

import (
	"os"
	"sort"

	"github.com/nick-skriabin/enva/internal/db"
	envpath "github.com/nick-skriabin/enva/internal/path"
)

// DefaultProfile is the default profile name.
const DefaultProfile = "default"

// ResolvedVar represents a resolved environment variable with provenance.
type ResolvedVar struct {
	Key           string
	Value         string
	DefinedAtPath string
	Overrode      bool
	OverrodePath  string
}

// Resolver handles environment variable resolution.
type Resolver struct {
	db      *db.DB
	profile string
}

// NewResolver creates a new resolver.
func NewResolver(database *db.DB, profile string) *Resolver {
	if profile == "" {
		profile = DefaultProfile
	}
	return &Resolver{db: database, profile: profile}
}

// GetProfile returns the active profile.
func (r *Resolver) GetProfile() string {
	return r.profile
}

// GetProfileFromEnv returns the profile from ENVA_PROFILE env var or default.
func GetProfileFromEnv() string {
	if p := os.Getenv("ENVA_PROFILE"); p != "" {
		return p
	}
	return DefaultProfile
}

// ResolveContext holds the resolution context for a directory.
type ResolveContext struct {
	CwdReal  string
	RootDir  string
	Chain    []string
	Resolved map[string]*ResolvedVar
	Profile  string
}

// Resolve resolves environment variables for the given directory.
func (r *Resolver) Resolve(cwd string) (*ResolveContext, error) {
	// Canonicalize cwd
	cwdReal, err := envpath.Canonicalize(cwd)
	if err != nil {
		return nil, err
	}

	// Find root
	rootDir, err := envpath.FindRoot(cwdReal)
	if err != nil {
		return nil, err
	}

	// Build chain
	chain, err := envpath.BuildChain(rootDir, cwdReal)
	if err != nil {
		return nil, err
	}

	// Load vars for all chain paths
	allVars, err := r.db.GetVarsForPaths(chain, r.profile)
	if err != nil {
		return nil, err
	}

	// Group vars by path
	varsByPath := make(map[string]map[string]string)
	for _, v := range allVars {
		if varsByPath[v.Path] == nil {
			varsByPath[v.Path] = make(map[string]string)
		}
		varsByPath[v.Path][v.Key] = v.Value
	}

	// Merge in chain order (parent first, child overrides)
	resolved := make(map[string]*ResolvedVar)
	for _, path := range chain {
		pathVars := varsByPath[path]
		for key, value := range pathVars {
			if existing, ok := resolved[key]; ok {
				// Override
				resolved[key] = &ResolvedVar{
					Key:           key,
					Value:         value,
					DefinedAtPath: path,
					Overrode:      true,
					OverrodePath:  existing.DefinedAtPath,
				}
			} else {
				resolved[key] = &ResolvedVar{
					Key:           key,
					Value:         value,
					DefinedAtPath: path,
					Overrode:      false,
				}
			}
		}
	}

	return &ResolveContext{
		CwdReal:  cwdReal,
		RootDir:  rootDir,
		Chain:    chain,
		Resolved: resolved,
		Profile:  r.profile,
	}, nil
}

// GetSortedVars returns resolved vars sorted by key.
func (ctx *ResolveContext) GetSortedVars() []*ResolvedVar {
	vars := make([]*ResolvedVar, 0, len(ctx.Resolved))
	for _, v := range ctx.Resolved {
		vars = append(vars, v)
	}
	sort.Slice(vars, func(i, j int) bool {
		return vars[i].Key < vars[j].Key
	})
	return vars
}

// GetLocalVars returns only vars defined at cwdReal.
func (ctx *ResolveContext) GetLocalVars() []*ResolvedVar {
	var vars []*ResolvedVar
	for _, v := range ctx.Resolved {
		if v.DefinedAtPath == ctx.CwdReal {
			vars = append(vars, v)
		}
	}
	sort.Slice(vars, func(i, j int) bool {
		return vars[i].Key < vars[j].Key
	})
	return vars
}

// IsLocal returns true if the var is defined at cwdReal.
func (ctx *ResolveContext) IsLocal(v *ResolvedVar) bool {
	return v.DefinedAtPath == ctx.CwdReal
}

// GetLocalVarsFromDB retrieves local vars directly from the database.
func (r *Resolver) GetLocalVarsFromDB(path string) ([]db.EnvVar, error) {
	canonical, err := envpath.Canonicalize(path)
	if err != nil {
		return nil, err
	}
	return r.db.GetVarsForPath(canonical, r.profile)
}

// SetVar sets a variable at the given path.
func (r *Resolver) SetVar(path, key, value string) error {
	canonical, err := envpath.Canonicalize(path)
	if err != nil {
		return err
	}
	return r.db.SetVar(canonical, r.profile, key, value)
}

// DeleteVar deletes a variable at the given path.
func (r *Resolver) DeleteVar(path, key string) error {
	canonical, err := envpath.Canonicalize(path)
	if err != nil {
		return err
	}
	return r.db.DeleteVar(canonical, r.profile, key)
}

// SetVarsBatch sets multiple variables at the given path.
func (r *Resolver) SetVarsBatch(path string, vars map[string]string) error {
	canonical, err := envpath.Canonicalize(path)
	if err != nil {
		return err
	}
	return r.db.SetVarsBatch(canonical, r.profile, vars)
}

// DeleteVarsBatch deletes multiple variables at the given path.
func (r *Resolver) DeleteVarsBatch(path string, keys []string) error {
	canonical, err := envpath.Canonicalize(path)
	if err != nil {
		return err
	}
	return r.db.DeleteVarsBatch(canonical, r.profile, keys)
}

// SyncLocalVars synchronizes local vars: adds/updates from newVars, deletes keys not in newVars.
func (r *Resolver) SyncLocalVars(path string, newVars map[string]string) error {
	canonical, err := envpath.Canonicalize(path)
	if err != nil {
		return err
	}

	// Get existing local vars
	existing, err := r.db.GetVarsForPath(canonical, r.profile)
	if err != nil {
		return err
	}

	// Find keys to delete
	var toDelete []string
	for _, v := range existing {
		if _, ok := newVars[v.Key]; !ok {
			toDelete = append(toDelete, v.Key)
		}
	}

	// Delete removed keys
	if len(toDelete) > 0 {
		if err := r.db.DeleteVarsBatch(canonical, r.profile, toDelete); err != nil {
			return err
		}
	}

	// Upsert new/updated vars
	if len(newVars) > 0 {
		if err := r.db.SetVarsBatch(canonical, r.profile, newVars); err != nil {
			return err
		}
	}

	return nil
}
