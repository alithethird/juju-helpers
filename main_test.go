package main

import (
	"strings"
	"testing"
)

// --- parseNukeAllArgs ---

func TestParseNukeAllArgs_NoFlags(t *testing.T) {
	parsed, showHelp, err := parseNukeAllArgs([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if showHelp {
		t.Fatal("expected showHelp=false")
	}
	if parsed.includeCurrent {
		t.Fatal("expected includeCurrent=false")
	}
}

func TestParseNukeAllArgs_IncludeCurrent(t *testing.T) {
	parsed, showHelp, err := parseNukeAllArgs([]string{"--include-current"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if showHelp {
		t.Fatal("expected showHelp=false")
	}
	if !parsed.includeCurrent {
		t.Fatal("expected includeCurrent=true")
	}
}

func TestParseNukeAllArgs_HelpLong(t *testing.T) {
	_, showHelp, err := parseNukeAllArgs([]string{"--help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !showHelp {
		t.Fatal("expected showHelp=true for --help")
	}
}

func TestParseNukeAllArgs_HelpShort(t *testing.T) {
	_, showHelp, err := parseNukeAllArgs([]string{"-h"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !showHelp {
		t.Fatal("expected showHelp=true for -h")
	}
}

func TestParseNukeAllArgs_UnknownFlag(t *testing.T) {
	_, _, err := parseNukeAllArgs([]string{"--unknown"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if !strings.Contains(err.Error(), "--unknown") {
		t.Fatalf("error should mention the unknown flag, got: %v", err)
	}
}

func TestParseNukeAllArgs_HelpDoesNotRun(t *testing.T) {
	// Regression: --help must return showHelp=true so the caller never
	// reaches the destructive code path.
	_, showHelp, err := parseNukeAllArgs([]string{"--help"})
	if err != nil || !showHelp {
		t.Fatalf("--help should set showHelp=true without error, got showHelp=%v err=%v", showHelp, err)
	}
}

// --- parseModelsOutput ---

var sampleModelsOutput = `Controller: lxd

Model               Cloud/Region         Type  Status     Machines  Units  Access  Last connection
controller          localhost/localhost  lxd   available         1      1  admin   just now
jubilant-5b7a99c7   localhost/localhost  lxd   available         1      1  admin   23 minutes ago
jubilant-7e88abdc*  localhost/localhost  lxd   available         1      1  admin   41 minutes ago
jubilant-d5583396   localhost/localhost  lxd   available         1      1  admin   32 minutes ago
test-foo            localhost/localhost  lxd   available         1      1  admin   1 hour ago
`

func TestParseModelsOutput_TargetModels(t *testing.T) {
	models, _ := parseModelsOutput(sampleModelsOutput)
	want := []string{"jubilant-5b7a99c7", "jubilant-7e88abdc", "jubilant-d5583396", "test-foo"}
	if len(models) != len(want) {
		t.Fatalf("got models %v, want %v", models, want)
	}
	for i, m := range models {
		if m != want[i] {
			t.Errorf("models[%d] = %q, want %q", i, m, want[i])
		}
	}
}

func TestParseModelsOutput_CurrentModel(t *testing.T) {
	_, current := parseModelsOutput(sampleModelsOutput)
	if current != "jubilant-7e88abdc" {
		t.Errorf("current = %q, want %q", current, "jubilant-7e88abdc")
	}
}

func TestParseModelsOutput_ControllerExcluded(t *testing.T) {
	models, _ := parseModelsOutput(sampleModelsOutput)
	for _, m := range models {
		if m == "controller" {
			t.Error("controller model should not be included")
		}
	}
}

func TestParseModelsOutput_Empty(t *testing.T) {
	models, current := parseModelsOutput("")
	if len(models) != 0 {
		t.Errorf("expected no models, got %v", models)
	}
	if current != "" {
		t.Errorf("expected no current model, got %q", current)
	}
}

func TestParseModelsOutput_NoCurrentMarker(t *testing.T) {
	output := `Controller: lxd

Model               Cloud/Region
test-abc            localhost/localhost
`
	models, current := parseModelsOutput(output)
	if len(models) != 1 || models[0] != "test-abc" {
		t.Errorf("unexpected models: %v", models)
	}
	if current != "" {
		t.Errorf("expected current=\"\", got %q", current)
	}
}

// --- replaceOrAppend ---

func TestReplaceOrAppend_FreshFile(t *testing.T) {
	result := replaceOrAppend("", aliasBlock)
	if !strings.Contains(result, blockStart) {
		t.Error("result should contain the block start marker")
	}
	if !strings.Contains(result, blockEnd) {
		t.Error("result should contain the block end marker")
	}
	if !strings.Contains(result, `alias js="juju status"`) {
		t.Error("result should contain the js alias")
	}
}

func TestReplaceOrAppend_Idempotent(t *testing.T) {
	once := replaceOrAppend("existing content\n", aliasBlock)
	twice := replaceOrAppend(once, aliasBlock)

	countStart := strings.Count(twice, blockStart)
	if countStart != 1 {
		t.Errorf("block start marker appears %d times after two applications, want 1", countStart)
	}
	countEnd := strings.Count(twice, blockEnd)
	if countEnd != 1 {
		t.Errorf("block end marker appears %d times after two applications, want 1", countEnd)
	}
}

func TestReplaceOrAppend_PreservesExistingContent(t *testing.T) {
	existing := "export PATH=$PATH:/usr/local/bin\nsource ~/.profile\n"
	result := replaceOrAppend(existing, aliasBlock)
	if !strings.Contains(result, "export PATH=$PATH:/usr/local/bin") {
		t.Error("existing content should be preserved")
	}
	if !strings.Contains(result, "source ~/.profile") {
		t.Error("existing content should be preserved")
	}
}

func TestReplaceOrAppend_ReplacesExistingBlock(t *testing.T) {
	oldBlock := "\n" + blockStart + "\nalias old=\"old command\"\n" + blockEnd + "\n"
	existing := "some content\n" + oldBlock + "more content\n"
	result := replaceOrAppend(existing, aliasBlock)

	if strings.Contains(result, `alias old="old command"`) {
		t.Error("old block content should be replaced")
	}
	if !strings.Contains(result, `alias js="juju status"`) {
		t.Error("new block content should be present")
	}
	if !strings.Contains(result, "some content") {
		t.Error("surrounding content should be preserved")
	}
	if !strings.Contains(result, "more content") {
		t.Error("surrounding content should be preserved")
	}
}
