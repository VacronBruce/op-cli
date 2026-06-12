package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Check (and update) op-cli configuration",
	Long: `Run a setup health check: config file, credentials, connection,
project, sprint, and shell completion — each line shows [ok] or [--] with
the exact fix for anything missing.

Flags update ~/.oprc in place (only the given key's line; comments and the
rest of the file are preserved), then the checks run against the new values.

Examples:
  op setup                               Show setup status
  op setup --sprint="App_06/15/2026"     Point config at the new sprint
  op setup --project=app                 Set the default project
  op setup --api-key=<key>               Store a (new) API key`,
	Args:        cobra.NoArgs,
	Annotations: skipClientInit(),
	RunE:        runSetup,
	// The issues themselves are the output; a usage dump after them is noise.
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.Flags().String("url", "", "Set the OpenProject base URL in ~/.oprc")
	setupCmd.Flags().String("api-key", "", "Set the API key in ~/.oprc")
	setupCmd.Flags().String("project", "", "Set the default project in ~/.oprc")
	setupCmd.Flags().String("sprint", "", "Set the default sprint in ~/.oprc")
}

// oprcPath is a seam for tests.
var oprcPath = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".oprc")
}

// shellRCPath returns the rc file install.sh wires completion into for the
// user's shell, or "" for unsupported shells.
var shellRCPath = func() (shell, rc string) {
	shell = filepath.Base(os.Getenv("SHELL"))
	home, err := os.UserHomeDir()
	if err != nil {
		return shell, ""
	}
	switch shell {
	case "zsh":
		return shell, filepath.Join(home, ".zshrc")
	case "bash":
		return shell, filepath.Join(home, ".bashrc")
	}
	return shell, ""
}

func runSetup(cmd *cobra.Command, args []string) error {
	// Apply any --url/--api-key/--project/--sprint updates first.
	updates := map[string]string{}
	for _, key := range []string{"url", "api-key", "project", "sprint"} {
		if val, _ := cmd.Flags().GetString(key); val != "" {
			updates[strings.ReplaceAll(key, "-", "_")] = val
		}
	}
	if len(updates) > 0 {
		if err := updateOprc(oprcPath(), updates); err != nil {
			return err
		}
		for k, v := range updates {
			viper.Set(k, v)
			fmt.Printf("Set %s in %s\n", k, oprcPath())
		}
		fmt.Println()
	}

	fmt.Println("op-cli setup status:")
	fmt.Println()

	failures := 0
	check := func(ok bool, label, fix string) {
		if ok {
			fmt.Printf("[ok] %s\n", label)
			return
		}
		failures++
		fmt.Printf("[--] %s\n", label)
		if fix != "" {
			fmt.Printf("     fix: %s\n", fix)
		}
	}

	// Config file
	path := oprcPath()
	_, statErr := os.Stat(path)
	check(statErr == nil, fmt.Sprintf("config: %s", path),
		"run install.sh, or create it (see README 'Setup')")

	// URL + API key (viper merges ~/.oprc and OP_* env vars)
	url := viper.GetString("url")
	apiKey := viper.GetString("api_key")
	check(url != "", "url: "+orMissing(url),
		"op setup --url=https://openpr.epochbase.com")
	keyOK := apiKey != "" && apiKey != "YOUR_API_KEY_HERE"
	check(keyOK, "api_key: "+maskKey(apiKey),
		fmt.Sprintf("create a token at %s/my/access_token, then: op setup --api-key=<key>", orMissing(url)))

	// Connection (only meaningful once url+key are present)
	if url != "" && keyOK {
		if client == nil {
			client = api.NewClient(url, apiKey, viper.GetString("project"))
		}
		me, err := client.GetMe()
		if err != nil {
			check(false, "connection: "+err.Error(), "check url, api_key, and network/VPN")
			// Project/sprint checks would only repeat the same network error
			// as bogus "not found"s — skip them until the connection works.
		} else {
			check(true, fmt.Sprintf("connection: logged in as %s", me.Name), "")

			// Project
			project := viper.GetString("project")
			if project == "" {
				check(false, "project: not set", "op setup --project=<identifier>  (see: op projects)")
			} else if _, err := client.GetProject(project); err != nil {
				check(false, fmt.Sprintf("project: %q not found", project), "op projects")
			} else {
				check(true, "project: "+project, "")

				// Sprint (optional but every sprint-scoped command depends on it)
				sprint := viper.GetString("sprint")
				if sprint == "" {
					check(false, "sprint: not set", `op setup --sprint="<name>"  (see: op sprint list)`)
				} else if _, err := client.ResolveVersion(project, sprint); err != nil {
					check(false, fmt.Sprintf("sprint: %q does not resolve in %q", sprint, project), "op sprint list")
				} else {
					check(true, fmt.Sprintf("sprint: %q", sprint), "")
				}
			}
		}
	} else {
		check(false, "connection: skipped (url/api_key missing)", "")
	}

	// Shell completion
	shell, rc := shellRCPath()
	if rc == "" {
		check(false, fmt.Sprintf("completion: shell %q not auto-configured", shell),
			"see: op completion --help")
	} else {
		data, _ := os.ReadFile(rc)
		enabled := strings.Contains(string(data), "op completion "+shell)
		check(enabled, fmt.Sprintf("completion: %s (%s)", shell, rc),
			fmt.Sprintf(`echo 'command -v op &>/dev/null && source <(op completion %s)' >> %s`, shell, rc))
	}

	fmt.Printf("\nop-cli %s\n", Version)
	if failures > 0 {
		return fmt.Errorf("%d setup issue(s) found", failures)
	}
	fmt.Println("All good.")
	return nil
}

// updateOprc sets scalar keys in the .oprc YAML by replacing each key's line
// in place (or appending it), preserving comments and everything else.
// Values are quoted so names with spaces or #
// survive YAML parsing.
func updateOprc(path string, updates map[string]string) error {
	if path == "" {
		return fmt.Errorf("cannot locate home directory for ~/.oprc")
	}
	var lines []string
	if data, err := os.ReadFile(path); err == nil {
		lines = strings.Split(string(data), "\n")
	}

	for key, val := range updates {
		newLine := fmt.Sprintf("%s: %q", key, val)
		replaced := false
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Only top-level scalar lines; never touch indented (nested) keys.
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") &&
				(strings.HasPrefix(trimmed, key+":") || strings.HasPrefix(trimmed, "# "+key+":")) {
				lines[i] = newLine
				replaced = true
				break
			}
		}
		if !replaced {
			// Append before any trailing empty lines.
			for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
				lines = lines[:len(lines)-1]
			}
			lines = append(lines, newLine)
		}
	}

	out := strings.Join(lines, "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return os.WriteFile(path, []byte(out), 0600)
}

func orMissing(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}

// maskKey shows just enough of the key to recognize it.
func maskKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if key == "YOUR_API_KEY_HERE" {
		return "(placeholder — not a real key)"
	}
	if len(key) <= 8 {
		return "set"
	}
	return key[:4] + "…" + key[len(key)-4:]
}
