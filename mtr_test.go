package main

import (
	"strings"
	"testing"
)

func TestFindMtrBinary_NotFound(t *testing.T) {
	_, err := findMtrBinary([]string{"/nonexistent/path"})
	if err == nil {
		t.Fatal("expected error when mtr not found")
	}
}

func TestBuildMtrArgs(t *testing.T) {
	args := buildMtrArgs("google.com")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--json") {
		t.Error("expected --json flag")
	}
	if !strings.Contains(joined, "--report") {
		t.Error("expected --report flag")
	}
	if !strings.Contains(joined, "google.com") {
		t.Error("expected target in args")
	}
}
