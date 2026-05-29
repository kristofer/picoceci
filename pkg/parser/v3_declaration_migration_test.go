package parser_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var legacyTypedDeclRE = regexp.MustCompile(`\|\s*[A-Za-z_][A-Za-z0-9_]*\s*:\s*[A-Za-z_][A-Za-z0-9_<>]*`)

func TestRepositorySourcesUseV3LetDeclarations(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to resolve current test file path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))

	for _, dir := range []string{"examples", "testdata"} {
		err := filepath.WalkDir(filepath.Join(repoRoot, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			ext := filepath.Ext(path)
			if ext != ".pc" {
				return nil
			}
			assertNoLegacyDeclarations(t, path)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}

	rootEntries, err := os.ReadDir(repoRoot)
	if err != nil {
		t.Fatalf("read repo root: %v", err)
	}
	for _, entry := range rootEntries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "test") && strings.HasSuffix(name, ".pc") {
			assertNoLegacyDeclarations(t, filepath.Join(repoRoot, name))
		}
	}
}

func TestPrimaryDocsUseV3LetDeclarations(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to resolve current test file path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	for _, rel := range []string{
		"README.md",
		"LANGUAGE_SPEC.md",
		"docs/stdlib.md",
	} {
		assertNoLegacyDeclarations(t, filepath.Join(repoRoot, rel))
	}
}

func assertNoLegacyDeclarations(t *testing.T, path string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if match := legacyTypedDeclRE.Find(content); match != nil {
		t.Fatalf("%s still contains legacy declaration syntax: %q", path, string(match))
	}
}
