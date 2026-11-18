package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func acceptanceDiagram(t *testing.T, binary string) {
	model := repoPath(t, "examples", "architecture.ad.hcl")
	first := runCLI(t, binary, "", "diagram", model)
	requireExit(t, first, 0)
	if first.stderr != "" || !strings.HasPrefix(first.stdout, "<!doctype html>") {
		t.Fatalf("unexpected output: %s", first.stderr)
	}
	second := runCLI(t, binary, "", "diagram", model)
	if first.stdout != second.stdout {
		t.Fatal("output is not deterministic")
	}
	dir := t.TempDir()
	for _, flag := range []string{"-o", "--output", "--output="} {
		target := filepath.Join(dir, "page.html")
		if err := os.WriteFile(target, []byte("old"), 0600); err != nil {
			t.Fatal(err)
		}
		args := []string{"diagram"}
		if flag == "--output=" {
			args = append(args, flag+target)
		} else {
			args = append(args, flag, target)
		}
		args = append(args, model)
		got := runCLI(t, binary, "", args...)
		requireExit(t, got, 0)
		if got.stdout != "" || readFile(t, target) != first.stdout {
			t.Fatalf("%s did not deliver the page", flag)
		}
	}
	missing := filepath.Join(dir, "missing", "out.html")
	got := runCLI(t, binary, "", "diagram", "-o", missing, model)
	requireExit(t, got, 1)
	if !strings.Contains(got.stderr, missing) {
		t.Fatalf("missing path in error: %s", got.stderr)
	}
	bad := writeFile(t, dir, "bad.ad.hcl", "design {\n")
	target := writeFile(t, dir, "existing.html", "original")
	got = runCLI(t, binary, "", "diagram", "-o", target, bad)
	requireExit(t, got, 1)
	if got.stdout != "" || got.stderr == "" || readFile(t, target) != "original" {
		t.Fatal("blocked assessment changed output")
	}
	for _, args := range [][]string{{"diagram"}, {"diagram", "-o"}, {"diagram", "-o", "x", "--output=y", model}, {"validate", "-o", "x", model}} {
		requireExit(t, runCLI(t, binary, "", args...), 2)
	}
	if f, err := os.OpenFile("/dev/full", os.O_WRONLY, 0); err == nil {
		defer f.Close()
		requireExit(t, execCLI(t, binary, "", f, "diagram", model), 1)
	}
	expectation := "\nexpect \"receipt_order\" {\n scope = design.receipt\n classification = \"decoupled\"\n order = [dp.exact_addition, dp.amount_admission]\n}\n"
	path := writeFile(t, dir, "expectation.ad.hcl", readFile(t, starterPath(t, "design.ad.hcl"))+expectation)
	got = runCLI(t, binary, "", "diagram", path)
	requireExit(t, got, 0)
	if !strings.Contains(got.stdout, "failed_expectation") {
		t.Fatal("missing failed expectation")
	}
	for _, input := range []string{repoPath(t, "examples", "project", "model.ad.project.hcl"), repoPath(t, "examples", "project")} {
		got = runCLI(t, binary, "", "diagram", input)
		requireExit(t, got, 0)
		report := runCLI(t, binary, "", "validate", input)
		for _, code := range findingCodes(report.stdout) {
			if !strings.Contains(got.stdout, code) {
				t.Errorf("%s: missing %s", input, code)
			}
		}
	}
	self := repoPath(t, "testdata", "spec", "examples", "requirements_hcl.ad.hcl")
	candidate := "decision.assessment_contracts.candidate.go_hcl_target"
	context := "context.semantic_changes"
	got = runCLI(t, binary, "", "diagram", "--candidate", candidate, "--context", context, self)
	requireExit(t, got, 0)
	if !strings.Contains(got.stdout, candidate) || !strings.Contains(got.stdout, context) {
		t.Fatal("selected candidate or context missing")
	}
}
