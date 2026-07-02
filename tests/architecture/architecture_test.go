// Package architecture_test contains fitness functions that enforce structural invariants.
// Run with: go test ./tests/architecture/...
//
// These tests scan Go source files for import patterns that would violate the
// DDD layering rules of this codebase. They require no external dependencies.
package architecture_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const repoRoot = "../.." // relative to tests/architecture/

// collectGoFiles returns all .go files under dir, excluding vendor and test helpers.
func collectGoFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "vendor" || d.Name() == ".git") {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// importsOf parses a Go file and returns its import paths.
func importsOf(path string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	var imports []string
	for _, imp := range f.Imports {
		p := strings.Trim(imp.Path.Value, `"`)
		imports = append(imports, p)
	}
	return imports, nil
}

// TestDomains_DoNotImportPlatformAPI enforces that no file under domains/ imports
// platform/api — domains should be agnostic of HTTP concerns.
func TestDomains_DoNotImportPlatformAPI(t *testing.T) {
	dir := filepath.Join(repoRoot, "domains")
	files, err := collectGoFiles(dir)
	if err != nil {
		t.Fatalf("walk domains/: %v", err)
	}
	const forbidden = "github.com/qeetgroup/qeet-notify/platform/api"
	for _, f := range files {
		imports, err := importsOf(f)
		if err != nil {
			t.Logf("skip parse error in %s: %v", f, err)
			continue
		}
		for _, imp := range imports {
			if strings.HasPrefix(imp, forbidden) {
				t.Errorf("%s imports %s (platform/api must not be imported by domains)", f, imp)
			}
		}
	}
}

// TestChannels_DoNotImportEachOther enforces that channel workers are independent —
// no channel package may import a sibling channel package.
func TestChannels_DoNotImportEachOther(t *testing.T) {
	channelsDir := filepath.Join(repoRoot, "domains", "channels")
	entries, err := os.ReadDir(channelsDir)
	if err != nil {
		t.Fatalf("read domains/channels/: %v", err)
	}

	var channels []string
	for _, e := range entries {
		if e.IsDir() {
			channels = append(channels, e.Name())
		}
	}

	for _, ch := range channels {
		chDir := filepath.Join(channelsDir, ch)
		files, err := collectGoFiles(chDir)
		if err != nil {
			t.Fatalf("walk channels/%s: %v", ch, err)
		}
		for _, f := range files {
			imports, err := importsOf(f)
			if err != nil {
				continue
			}
			for _, imp := range imports {
				for _, sibling := range channels {
					if sibling == ch {
						continue
					}
					forbidden := "github.com/qeetgroup/qeet-notify/domains/channels/" + sibling
					if strings.HasPrefix(imp, forbidden) {
						t.Errorf("%s (channel %s) imports sibling channel %s — channels must be independent", f, ch, sibling)
					}
				}
			}
		}
	}
}

// TestPlatform_DoNotImportDomains enforces that platform/ packages do not import
// any domains/ package — platform is the foundation layer.
func TestPlatform_DoNotImportDomains(t *testing.T) {
	dir := filepath.Join(repoRoot, "platform")
	files, err := collectGoFiles(dir)
	if err != nil {
		t.Fatalf("walk platform/: %v", err)
	}
	const forbidden = "github.com/qeetgroup/qeet-notify/domains"
	for _, f := range files {
		imports, err := importsOf(f)
		if err != nil {
			t.Logf("skip parse error in %s: %v", f, err)
			continue
		}
		for _, imp := range imports {
			if strings.HasPrefix(imp, forbidden) {
				t.Errorf("%s (platform) imports %s (domains must not be imported by platform)", f, imp)
			}
		}
	}
}
