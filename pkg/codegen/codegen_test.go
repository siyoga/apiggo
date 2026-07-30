package codegen

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

const fixtureContract = "testdata/petstore-min/openapi.yaml"

// generatedFiles returns the module-relative paths of every file under root.
func generatedFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	require.NoError(t, filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, relErr := filepath.Rel(root, p)
			if relErr != nil {
				return relErr
			}
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	}))
	sort.Strings(files)
	return files
}

func TestGenerateGolden(t *testing.T) {
	out := t.TempDir()
	require.NoError(t, Generate(Config{
		ContractPath: fixtureContract,
		Module:       "example.com/svc",
		OutRoot:      out,
	}))

	goldenDir := "testdata/petstore-min/golden"
	for _, rel := range generatedFiles(t, out) {
		got, err := os.ReadFile(filepath.Join(out, rel))
		require.NoError(t, err)

		goldenPath := filepath.Join(goldenDir, filepath.FromSlash(rel))
		if *update {
			require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0o755))
			require.NoError(t, os.WriteFile(goldenPath, got, 0o644))
			continue
		}

		want, err := os.ReadFile(goldenPath)
		require.NoError(t, err, "missing golden for %s (run: go test -run TestGenerateGolden -update)", rel)
		assert.Equal(t, string(want), string(got), "generated %s differs from golden", rel)
	}
}

func TestOperationWithoutIDFails(t *testing.T) {
	dir := t.TempDir()
	contract := filepath.Join(dir, "no-op-id.yaml")
	require.NoError(t, os.WriteFile(contract, []byte(`openapi: 3.0.3
info: { title: t, version: "1" }
paths:
  /x:
    get:
      responses:
        '200': { description: ok }
`), 0o644))

	err := Generate(Config{ContractPath: contract, Module: "example.com/svc", OutRoot: t.TempDir()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "operationId")
}

// TestGeneratedCompiles is the end-to-end check: generate the fixture into a
// throwaway module that points at the local runtime via replace, then `go build`
// it to prove the emitted code compiles against the real server package.
func TestGeneratedCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compile E2E in -short mode")
	}

	out := t.TempDir()
	const module = "apiggo.example/e2e"
	require.NoError(t, Generate(Config{
		ContractPath: fixtureContract,
		Module:       module,
		OutRoot:      out,
	}))

	wd, err := os.Getwd()
	require.NoError(t, err)
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..")) // pkg/codegen -> repo root

	gomod := fmt.Sprintf(
		"module %s\n\ngo 1.26.2\n\nrequire github.com/siyoga/apiggo v0.0.0\n\nreplace github.com/siyoga/apiggo => %s\n",
		module, repoRoot,
	)
	require.NoError(t, os.WriteFile(filepath.Join(out, "go.mod"), []byte(gomod), 0o644))

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = out
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOPROXY=off")
	build, err := cmd.CombinedOutput()
	require.NoError(t, err, "generated code failed to compile:\n%s", build)
}
