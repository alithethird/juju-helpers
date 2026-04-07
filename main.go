package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const aliasBlock = `
# juju-helpers aliases - DO NOT EDIT (managed by juju-helpers)
alias js="juju status"
alias jss="juju status --watch 2s --relations"
alias jm="juju models"
alias jsc="juju switch"
alias nuke="juju destroy-model --force --no-prompt --destroy-storage --no-wait"
# end juju-helpers aliases
`

const blockStart = "# juju-helpers aliases - DO NOT EDIT (managed by juju-helpers)"
const blockEnd = "# end juju-helpers aliases"

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: juju-helpers <command>

Commands:
  seed       Add juju aliases to ~/.bashrc and ~/.zshrc (idempotent)
  nuke-all [--include-current]   Destroy all models whose names start with test- or jubilant-
                                 (skips the current model by default)
`)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "seed":
		if err := seedShells(); err != nil {
			fmt.Fprintf(os.Stderr, "seed: %v\n", err)
			os.Exit(1)
		}
	case "nuke-all":
		if err := nukeAll(); err != nil {
			fmt.Fprintf(os.Stderr, "nuke-all: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
	}
}

// seedShells writes the alias block into ~/.bashrc and ~/.zshrc, replacing any
// existing managed block so the operation is idempotent.
func seedShells() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	targets := []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".zshrc"),
	}

	for _, path := range targets {
		if err := seedFile(path); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		fmt.Printf("seeded %s\n", path)
	}
	return nil
}

func seedFile(path string) error {
	// Read existing content (file may not exist yet).
	var existing []byte
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	updated := replaceOrAppend(string(existing), aliasBlock)
	return os.WriteFile(path, []byte(updated), 0644)
}

// replaceOrAppend removes any existing managed block and appends the new one.
func replaceOrAppend(content, block string) string {
	lines := strings.Split(content, "\n")
	var out []string
	skip := false
	for _, line := range lines {
		if strings.TrimSpace(line) == blockStart {
			skip = true
		}
		if !skip {
			out = append(out, line)
		}
		if skip && strings.TrimSpace(line) == blockEnd {
			skip = false
		}
	}
	result := strings.Join(out, "\n")
	// Ensure the file ends with a newline before appending.
	result = strings.TrimRight(result, "\n")
	result += "\n" + block
	return result
}

// nukeAll finds all models starting with test- or jubilant- and destroys them.
// By default the currently active model is skipped; pass --include-current to override.
func nukeAll() error {
	includeCurrent := false
	for _, arg := range os.Args[2:] {
		if arg == "--include-current" {
			includeCurrent = true
		}
	}

	models, current, err := listTargetModels()
	if err != nil {
		return err
	}

	if !includeCurrent && current != "" {
		var filtered []string
		for _, m := range models {
			if m != current {
				filtered = append(filtered, m)
			}
		}
		if len(filtered) < len(models) {
			fmt.Printf("(skipping current model %q — use --include-current to include it)\n", current)
		}
		models = filtered
	}

	if len(models) == 0 {
		fmt.Println("no models found matching test-* or jubilant-*")
		return nil
	}

	fmt.Println("Models to destroy:")
	for _, m := range models {
		fmt.Printf("  %s\n", m)
	}
	fmt.Printf("\nDestroy %d model(s)? [y/N] ", len(models))

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		fmt.Println("aborted")
		return nil
	}

	var failed []string
	for _, model := range models {
		fmt.Printf("nuking %s ... ", model)
		out, err := exec.Command(
			"juju", "destroy-model",
			"--force", "--no-prompt", "--destroy-storage", "--no-wait",
			model,
		).CombinedOutput()
		if err != nil {
			fmt.Println("FAILED")
			fmt.Fprintf(os.Stderr, "%s\n", bytes.TrimSpace(out))
			failed = append(failed, model)
		} else {
			fmt.Println("ok")
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to destroy: %s", strings.Join(failed, ", "))
	}
	return nil
}

// listTargetModels returns model names (without controller prefix or trailing *)
// that start with test- or jubilant-, plus the name of the currently active model.
func listTargetModels() (models []string, current string, err error) {
	out, err := exec.Command("juju", "models").Output()
	if err != nil {
		return nil, "", fmt.Errorf("juju models: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		raw := fields[0]
		isCurrent := strings.HasSuffix(raw, "*")
		name := strings.TrimSuffix(raw, "*")
		if isCurrent {
			current = name
		}
		if strings.HasPrefix(name, "test-") || strings.HasPrefix(name, "jubilant-") {
			models = append(models, name)
		}
	}
	return models, current, scanner.Err()
}
