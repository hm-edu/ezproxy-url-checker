package main

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readList(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		return nil
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

func TestGenerate(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config.txt"), strings.Join([]string{
		"# comments and unrelated directives are ignored",
		"LoginPort 80",
		"IncludeFile databases/scholar.txt",
		"includefile databases/nested.txt",
		"",
		"Title Lehigh",
		"URL https://lts.lehigh.edu/some/path?x=1",
		"U lts.lehigh.edu/other",
		"URL -Refresh https://preserve.lehigh.edu/",
		"H www.lehigh.edu",
		"HJ https://www.lehigh.edu/js",
		"Host preserve.lehigh.edu",
		"HostJavascript hj.lehigh.edu",
		"D lib.lehigh.edu",
		"Domain lib.lehigh.edu",
		"DJ ignored.lehigh.edu",
		"DomainJavascript ignored.lehigh.edu",
		"",
	}, "\n"))
	// CRLF line endings
	writeFile(t, filepath.Join(dir, "databases", "scholar.txt"),
		"T Google Scholar\r\nU https://scholar.google.com\r\nDJ scholar.google.com\r\nD google.com\r\n")
	// tab separated, and an include loop back to config.txt
	writeFile(t, filepath.Join(dir, "databases", "nested.txt"),
		"IncludeFile\tconfig.txt\nH\tnested.example.org\n")

	out := filepath.Join(t.TempDir(), "config")
	var logBuf strings.Builder
	if err := generate(dir, out, &logBuf); err != nil {
		t.Fatalf("generate: %v", err)
	}

	want := map[string][]string{
		"urls.txt":    {"lts.lehigh.edu", "preserve.lehigh.edu", "scholar.google.com"},
		"hosts.txt":   {"hj.lehigh.edu", "nested.example.org", "preserve.lehigh.edu", "www.lehigh.edu"},
		"domains.txt": {"google.com", "lib.lehigh.edu"},
	}
	for name, wantLines := range want {
		if got := readList(t, filepath.Join(out, name)); !slices.Equal(got, wantLines) {
			t.Errorf("%s = %q, want %q", name, got, wantLines)
		}
	}

	for _, file := range []string{"config.txt", "databases/scholar.txt", "databases/nested.txt"} {
		if !strings.Contains(logBuf.String(), filepath.Join(dir, file)) {
			t.Errorf("log does not mention %s:\n%s", file, logBuf.String())
		}
	}
}

func TestGenerateWritesEmptyLists(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config.txt"), "LoginPort 80\n")

	out := filepath.Join(t.TempDir(), "config")
	if err := generate(dir, out, io.Discard); err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, name := range listFiles {
		if got := readList(t, filepath.Join(out, name)); len(got) != 0 {
			t.Errorf("%s = %q, want empty file", name, got)
		}
	}
}

func TestGenerateErrors(t *testing.T) {
	dir := t.TempDir()
	if err := generate(filepath.Join(dir, "missing"), t.TempDir(), io.Discard); err == nil {
		t.Error("expected an error for a missing directory")
	}
	if err := generate(dir, t.TempDir(), io.Discard); err == nil {
		t.Error("expected an error when config.txt is missing")
	}
	writeFile(t, filepath.Join(dir, "config.txt"), "IncludeFile nope.txt\n")
	if err := generate(dir, t.TempDir(), io.Discard); err == nil {
		t.Error("expected an error for a missing IncludeFile")
	}
}

func TestRunGenerate(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config.txt"), "H example.org\n")
	out := filepath.Join(t.TempDir(), "lists")

	if err := runGenerate([]string{"-out", out, dir}, io.Discard, io.Discard); err != nil {
		t.Fatalf("runGenerate: %v", err)
	}
	if got := readList(t, filepath.Join(out, "hosts.txt")); !slices.Equal(got, []string{"example.org"}) {
		t.Errorf("hosts.txt = %q, want [example.org]", got)
	}

	if err := runGenerate(nil, io.Discard, io.Discard); err == nil {
		t.Error("expected an error when no directory is passed")
	}
	if err := runGenerate([]string{"-h"}, io.Discard, io.Discard); err != nil {
		t.Errorf("-h should not be an error, got %v", err)
	}
}

func TestDirectiveValue(t *testing.T) {
	tests := map[string]string{
		"example.com":                      "example.com",
		"https://example.com":              "example.com",
		"HTTP://example.com/path/x?y=1":    "example.com",
		"example.com/path":                 "example.com",
		"https://example.com:8080/x":       "example.com:8080",
		"-Refresh https://example.com/":    "example.com",
		"-Form=post -Encoded example.com/": "example.com",
		"-OnlyQualifier":                   "",
	}
	for in, want := range tests {
		if got := directiveValue(strings.Fields(in)); got != want {
			t.Errorf("directiveValue(%q) = %q, want %q", in, got, want)
		}
	}
}
