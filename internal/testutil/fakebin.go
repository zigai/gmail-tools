// Package testutil provides fake executables for tests that run external commands.
package testutil

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// FakeBinary is an executable that records each call's arguments and answers
// with the output of the first response whose pattern appears in its arguments.
type FakeBinary struct {
	Path    string
	callDir string
}

// Response is printed when Pattern appears in the space-joined arguments. An
// empty Pattern matches every call.
type Response struct {
	Pattern string
	Stdout  string
	Code    int
}

func NewFakeBinary(t *testing.T, responses ...Response) FakeBinary {
	t.Helper()

	dir := t.TempDir()
	callDir := filepath.Join(dir, "calls")

	if err := os.Mkdir(callDir, 0o700); err != nil {
		t.Fatalf("create call directory: %v", err)
	}

	var script strings.Builder

	script.WriteString("#!/bin/sh\n")
	script.WriteString("printf '%s\\n' \"$@\" > '" + callDir + "'/$(date +%s%N)-$$\n")
	script.WriteString("case \" $* \" in\n")

	for i, response := range responses {
		out := filepath.Join(dir, "stdout"+strconv.Itoa(i))
		if err := os.WriteFile(out, []byte(response.Stdout), 0o600); err != nil {
			t.Fatalf("write fake stdout: %v", err)
		}

		script.WriteString("  *'" + response.Pattern + "'*) cat '" + out + "'; exit " +
			strconv.Itoa(response.Code) + " ;;\n")
	}

	script.WriteString("esac\n")

	path := filepath.Join(dir, "fake")
	if err := os.WriteFile(path, []byte(script.String()), 0o700); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	return FakeBinary{Path: path, callDir: callDir}
}

// Calls returns the arguments of every call in order.
func (f FakeBinary) Calls(t *testing.T) [][]string {
	t.Helper()

	entries, err := os.ReadDir(f.callDir)
	if err != nil {
		t.Fatalf("read fake calls: %v", err)
	}

	calls := make([][]string, 0, len(entries))
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(f.callDir, entry.Name()))
		if err != nil {
			t.Fatalf("read fake call: %v", err)
		}

		calls = append(calls, strings.Split(strings.TrimSuffix(string(data), "\n"), "\n"))
	}

	return calls
}

// Args returns the last call's arguments, or nil when the binary never ran.
func (f FakeBinary) Args(t *testing.T) []string {
	t.Helper()

	calls := f.Calls(t)
	if len(calls) == 0 {
		return nil
	}

	return calls[len(calls)-1]
}
