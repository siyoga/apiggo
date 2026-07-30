package main

import (
	"os"
	"path/filepath"
	"testing"

	gen "github.com/siyoga/apiggo/pkg/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "apiggo.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestLoadFileConfig(t *testing.T) {
	path := writeTemp(t, `schemaPath: api/openapi.yaml
module: github.com/acme/svc
out: build
dtoPackage: gen/dto
routerPackage: gen/router
apiPackage: handlers
`)

	fc, err := loadFileConfig(path)
	require.NoError(t, err)
	assert.Equal(t, fileConfig{
		SchemaPath: "api/openapi.yaml",
		Module:     "github.com/acme/svc",
		Out:        "build",
		DTOPkg:     "gen/dto",
		RouterPkg:  "gen/router",
		APIPkg:     "handlers",
	}, fc)
}

func TestLoadFileConfigUnknownKeyFails(t *testing.T) {
	path := writeTemp(t, "schemaPath: api.yaml\ncontract: oops\n")

	_, err := loadFileConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "contract")
}

func TestToGenConfig(t *testing.T) {
	fc := fileConfig{
		SchemaPath: "api/openapi.yaml",
		Module:     "github.com/acme/svc",
		Out:        "build",
		DTOPkg:     "gen/dto",
		RouterPkg:  "gen/router",
		APIPkg:     "handlers",
	}

	assert.Equal(t, gen.Config{
		ContractPath: "api/openapi.yaml",
		Module:       "github.com/acme/svc",
		OutRoot:      "build",
		DTOPkg:       "gen/dto",
		RouterPkg:    "gen/router",
		APIPkg:       "handlers",
	}, fc.toGenConfig())
}

func TestApplyFlagOverrides(t *testing.T) {
	cfg := gen.Config{
		ContractPath: "from-config.yaml",
		Module:       "github.com/acme/svc",
		OutRoot:      "config-out",
	}

	// Only -schema and -out were passed on the CLI.
	applyFlagOverrides(&cfg, map[string]string{
		flagSchema: "from-flag.yaml",
		flagOut:    "flag-out",
	})

	assert.Equal(t, "from-flag.yaml", cfg.ContractPath) // overridden
	assert.Equal(t, "flag-out", cfg.OutRoot)            // overridden
	assert.Equal(t, "github.com/acme/svc", cfg.Module)  // untouched
}
