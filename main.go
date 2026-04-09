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
  nuke-all [--include-current]
             Destroy all models whose names start with test- or jubilant-.
             Skips the currently active model by default.`)
	os.Exit(1)
}

func nukeAllUsage() {
	fmt.Println(`Usage: juju-helpers nuke-all [--include-current]

Destroy all juju models whose names start with test- or jubilant-.
The currently active model is skipped unless --include-current is given.`)
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
		if err := nukeAll(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "nuke-all: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
	}
}

// nukeAllArgs holds parsed flags for the nuke-all command.
type nukeAllArgs struct {
	includeCurrent bool
}

// parseNukeAllArgs parses flags for nuke-all. Returns (nil, true) when --help
// was requested. Returns an error for any unrecognised flag.
func parseNukeAllArgs(args []string) (*nukeAllArgs, bool, error) {
	result := &nukeAllArgs{}
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return nil, true, nil
		case "--include-current":
			result.includeCurrent = true
		default:
			return nil, false, fmt.Errorf("unknown flag %q", arg)
		}
	}
	return result, false, nil
}

// nukeAll is the entry point for the nuke-all command.
func nukeAll(args []string) error {
	parsed, showHelp, err := parseNukeAllArgs(args)
	if err != nil {
		nukeAllUsage()
		return err
	}
	if showHelp {
		nukeAllUsage()
		return nil
	}

	modelsOutput, err := exec.Command("juju", "models").Output()
	if err != nil {
		return fmt.Errorf("juju models: %w", err)
	}

	models, current := parseModelsOutput(string(modelsOutput))

	if !parsed.includeCurrent && current != "" {
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

// parseModelsOutput parses the output of `juju models` and returns the names
// of models starting with test- or jubilant-, plus the currently active model.
func parseModelsOutput(output string) (models []string, current string) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
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
	return models, current
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
	result := strings.TrimRight(strings.Join(out, "\n"), "\n")
	result += "\n" + block
	return result
}
