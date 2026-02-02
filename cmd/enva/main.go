/*
enva - Per-directory environment variable manager with SQLite storage

SHELL INTEGRATION:

	Add to your shell config:

	# For bash (~/.bashrc):
	eval "$(enva hook bash)"

	# For zsh (~/.zshrc):
	eval "$(enva hook zsh)"

	# For fish (~/.config/fish/config.fish):
	enva hook fish | source

	This will automatically load/unload environment variables when you cd.

COMMANDS:

	enva hook <shell>   Print shell hook code (bash, zsh, fish)
	enva export         Print export/unset lines for current directory
	enva set KEY=VALUE  Set a variable at current directory scope
	enva unset KEY      Remove a variable from current directory scope
	enva ls             List effective environment variables (sorted)
	enva edit           Open $EDITOR to edit local vars for current directory
	enva run -- CMD     Run command with effective env merged into current env
	enva tui            Launch interactive TUI

ROOT BOUNDARY DISCOVERY:
 1. Walk up from cwd looking for .enva marker file (closest wins)
 2. If none found, look for .git/ directory (closest wins)
 3. If none found, use filesystem root /

PROFILE SUPPORT:

	Set ENVA_PROFILE environment variable to use a different profile.
	Default profile is "default".

DATABASE LOCATION:

	~/.local/share/enva/enva.db
*/
package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/nick-skriabin/enva/internal/db"
	"github.com/nick-skriabin/enva/internal/env"
	envpath "github.com/nick-skriabin/enva/internal/path"
	"github.com/nick-skriabin/enva/internal/shell"
	"github.com/nick-skriabin/enva/internal/tui"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "enva",
	Short: "Per-directory environment variable manager",
	Long: `enva manages per-directory environment variables stored in SQLite.

It provides automatic shell integration for loading/unloading environment
variables when changing directories. Use 'enva hook <shell>' to set up.`,
}

func init() {
	rootCmd.AddCommand(hookCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(unsetCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(tuiCmd)
}

// Helper to get database and resolver
func getDBAndResolver() (*db.DB, *env.Resolver, error) {
	dbPath, err := db.DefaultDBPath()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database path: %w", err)
	}

	database, err := db.Open(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	profile := env.GetProfileFromEnv()
	resolver := env.NewResolver(database, profile)

	return database, resolver, nil
}

// hookCmd prints shell hook code
var hookCmd = &cobra.Command{
	Use:   "hook [bash|zsh|fish]",
	Short: "Print shell hook code for automatic environment loading",
	Long: `Print shell-specific code that sets up automatic loading/unloading
of environment variables when changing directories.

Add to your shell config:
  # bash: eval "$(enva hook bash)"
  # zsh:  eval "$(enva hook zsh)"
  # fish: enva hook fish | source`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shellName := strings.ToLower(args[0])

		switch shellName {
		case "bash":
			fmt.Print(bashHook)
		case "zsh":
			fmt.Print(zshHook)
		case "fish":
			fmt.Print(fishHook)
		default:
			return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish)", shellName)
		}
		return nil
	},
}

const bashHook = `_enva_hook() { local s=$?; eval "$(enva export)"; return $s; }
if ! [[ "${PROMPT_COMMAND:-}" =~ _enva_hook ]]; then PROMPT_COMMAND="_enva_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}"; fi
`

const zshHook = `_enva_hook() { eval "$(enva export)"; }; autoload -Uz add-zsh-hook; add-zsh-hook precmd _enva_hook`

const fishHook = `function _enva_hook --on-variable PWD
    enva export | source
end
enva export | source
`

// exportCmd prints shell export/unset lines
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Print shell export/unset lines for effective environment",
	Long: `Print shell commands to load/unload environment variables for the
current directory. Tracks previously loaded variables and unsets them
when they're no longer needed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		database, resolver, err := getDBAndResolver()
		if err != nil {
			return err
		}
		defer database.Close()

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get cwd: %w", err)
		}

		ctx, err := resolver.Resolve(cwd)
		if err != nil {
			return fmt.Errorf("failed to resolve environment: %w", err)
		}

		// Get current vars
		newVars := ctx.GetSortedVars()
		newKeys := make(map[string]bool)
		newVals := make(map[string]string)
		for _, v := range newVars {
			newKeys[v.Key] = true
			newVals[v.Key] = v.Value
		}

		// Get previously loaded keys and path from env
		prevKeysStr := os.Getenv("__ENVA_LOADED_KEYS")
		prevPath := os.Getenv("__ENVA_LOADED_PATH")
		var prevKeys []string
		prevKeysSet := make(map[string]bool)
		if prevKeysStr != "" {
			prevKeys = strings.Split(prevKeysStr, ":")
			for _, k := range prevKeys {
				if k != "" {
					prevKeysSet[k] = true
				}
			}
		}

		// Count changes
		var unsetCount, loadCount int

		// Unset keys that are no longer in the environment
		for _, key := range prevKeys {
			if key != "" && !newKeys[key] {
				fmt.Printf("unset %s\n", key)
				unsetCount++
			}
		}

		// Export new values
		for _, v := range newVars {
			fmt.Println(shell.FormatExport(v.Key, v.Value))
			if !prevKeysSet[v.Key] {
				loadCount++
			}
		}

		// Update the tracking variables
		var keysList []string
		for _, v := range newVars {
			keysList = append(keysList, v.Key)
		}

		// Track current path
		cwdReal := ctx.CwdReal
		if len(keysList) > 0 {
			fmt.Printf("export __ENVA_LOADED_KEYS='%s'\n", strings.Join(keysList, ":"))
			fmt.Printf("export __ENVA_LOADED_PATH='%s'\n", cwdReal)
		} else if prevKeysStr != "" {
			fmt.Println("unset __ENVA_LOADED_KEYS")
			fmt.Println("unset __ENVA_LOADED_PATH")
		}

		// Print status message to stderr
		if unsetCount > 0 && len(newVars) == 0 {
			fmt.Fprintf(os.Stderr, "enva: unloaded %d var(s)\n", unsetCount)
		} else if loadCount > 0 && prevPath != cwdReal {
			fmt.Fprintf(os.Stderr, "enva: loaded %d var(s)\n", len(newVars))
		} else if unsetCount > 0 || loadCount > 0 {
			if prevPath != cwdReal {
				fmt.Fprintf(os.Stderr, "enva: loaded %d var(s)\n", len(newVars))
			}
		}

		return nil
	},
}

// setCmd sets a variable at current directory scope
var setCmd = &cobra.Command{
	Use:   "set KEY=VALUE",
	Short: "Set an environment variable at current directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value, ok := shell.ParseKeyValue(args[0])
		if !ok {
			return fmt.Errorf("invalid format: expected KEY=VALUE")
		}

		if !shell.IsValidKey(key) {
			return fmt.Errorf("invalid key: must match [A-Za-z_][A-Za-z0-9_]*")
		}

		database, resolver, err := getDBAndResolver()
		if err != nil {
			return err
		}
		defer database.Close()

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get cwd: %w", err)
		}

		if err := resolver.SetVar(cwd, key, value); err != nil {
			return fmt.Errorf("failed to set variable: %w", err)
		}

		fmt.Printf("Set %s at %s\n", key, cwd)
		return nil
	},
}

// unsetCmd deletes a variable from current directory scope
var unsetCmd = &cobra.Command{
	Use:   "unset KEY",
	Short: "Remove an environment variable from current directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		if !shell.IsValidKey(key) {
			return fmt.Errorf("invalid key: must match [A-Za-z_][A-Za-z0-9_]*")
		}

		database, resolver, err := getDBAndResolver()
		if err != nil {
			return err
		}
		defer database.Close()

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get cwd: %w", err)
		}

		if err := resolver.DeleteVar(cwd, key); err != nil {
			return fmt.Errorf("failed to unset variable: %w", err)
		}

		fmt.Printf("Unset %s at %s\n", key, cwd)
		return nil
	},
}

// lsCmd lists effective variables
var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List effective environment variables",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, resolver, err := getDBAndResolver()
		if err != nil {
			return err
		}
		defer database.Close()

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get cwd: %w", err)
		}

		ctx, err := resolver.Resolve(cwd)
		if err != nil {
			return fmt.Errorf("failed to resolve environment: %w", err)
		}

		vars := ctx.GetSortedVars()
		for _, v := range vars {
			fmt.Printf("%s=%s\n", v.Key, v.Value)
		}
		return nil
	},
}

// editCmd opens $EDITOR for editing local vars
var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit local environment variables in $EDITOR",
	Long: `Opens $EDITOR with KEY=VALUE lines for local variables at the current
directory. After saving, parses the file and applies changes (upserts/deletes).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		database, resolver, err := getDBAndResolver()
		if err != nil {
			return err
		}
		defer database.Close()

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get cwd: %w", err)
		}

		cwdCanon, err := envpath.Canonicalize(cwd)
		if err != nil {
			return fmt.Errorf("failed to canonicalize cwd: %w", err)
		}

		// Get current local vars
		localVars, err := resolver.GetLocalVarsFromDB(cwd)
		if err != nil {
			return fmt.Errorf("failed to get local vars: %w", err)
		}

		// Build content
		var lines []string
		sort.Slice(localVars, func(i, j int) bool {
			return localVars[i].Key < localVars[j].Key
		})
		for _, v := range localVars {
			lines = append(lines, fmt.Sprintf("%s=%s", v.Key, v.Value))
		}
		content := strings.Join(lines, "\n")
		if content != "" {
			content += "\n"
		}

		// Create temp file
		tmpFile, err := os.CreateTemp("", "enva-edit-*.env")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		if _, err := tmpFile.WriteString(content); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		tmpFile.Close()

		// Open editor
		editorCmd := exec.Command(editor, tmpPath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}

		// Read updated content
		newContent, err := os.ReadFile(tmpPath)
		if err != nil {
			return fmt.Errorf("failed to read temp file: %w", err)
		}

		// Parse new content
		newVars, invalid := shell.ParseEnvFile(string(newContent))
		if len(invalid) > 0 {
			return fmt.Errorf("invalid lines in file: %v", invalid)
		}

		// Sync vars
		if err := resolver.SyncLocalVars(cwdCanon, newVars); err != nil {
			return fmt.Errorf("failed to sync vars: %w", err)
		}

		fmt.Printf("Updated local vars at %s\n", cwdCanon)
		return nil
	},
}

// runCmd executes a command with the effective environment
var runCmd = &cobra.Command{
	Use:   "run -- COMMAND [ARGS...]",
	Short: "Run a command with effective environment",
	Long: `Executes the given command with the effective environment variables
merged into the current process environment.`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Find -- separator
		cmdArgs := args
		for i, arg := range args {
			if arg == "--" {
				cmdArgs = args[i+1:]
				break
			}
		}

		if len(cmdArgs) == 0 {
			return fmt.Errorf("no command specified")
		}

		database, resolver, err := getDBAndResolver()
		if err != nil {
			return err
		}
		defer database.Close()

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get cwd: %w", err)
		}

		ctx, err := resolver.Resolve(cwd)
		if err != nil {
			return fmt.Errorf("failed to resolve environment: %w", err)
		}

		// Build environment: current env + enva vars
		envMap := make(map[string]string)
		for _, e := range os.Environ() {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		// Override with enva vars
		for _, v := range ctx.GetSortedVars() {
			envMap[v.Key] = v.Value
		}

		// Convert back to slice
		var environ []string
		for k, v := range envMap {
			environ = append(environ, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(environ)

		// Find command path
		cmdPath, err := exec.LookPath(cmdArgs[0])
		if err != nil {
			return fmt.Errorf("command not found: %s", cmdArgs[0])
		}

		// Exec (replaces current process)
		return syscall.Exec(cmdPath, cmdArgs, environ)
	},
}

// tuiCmd launches the TUI
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, resolver, err := getDBAndResolver()
		if err != nil {
			return err
		}
		defer database.Close()

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get cwd: %w", err)
		}

		return tui.Run(database, resolver, cwd)
	},
}
