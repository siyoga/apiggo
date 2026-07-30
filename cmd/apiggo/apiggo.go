// Command apiggo generates dto, router-glue and handler-stub code from an
// OpenAPI contract.
//
// Configuration comes from a YAML file (-config) and/or CLI flags; an explicitly
// set flag overrides the corresponding config-file value.
//
// Usage:
//
//	apiggo -config apiggo.yaml
//	apiggo -schema api.yaml -module github.com/acme/svc -out .
//	apiggo -output-config > apiggo.yaml
package main

import (
	"flag"
	"fmt"
	"os"

	gen "github.com/siyoga/apiggo/pkg/codegen"
	"gopkg.in/yaml.v3"
)

func main() {
	var (
		configPath   string
		outputConfig bool
	)
	flag.StringVar(&configPath, "config", "", "path to a YAML config file")
	flag.BoolVar(&outputConfig, "output-config", false, "write a starter config to stdout and exit")

	// Per-field flags. Empty defaults so flag.Visit can tell set from unset.
	flag.String(flagSchema, "", "path to the OpenAPI schema (YAML or JSON) [required]")
	flag.String(flagModule, "", "Go module path of the target project, for generated imports [required]")
	flag.String(flagOut, "", "output root directory (default \".\")")
	flag.String(flagDTOPkg, "", "module-relative package path for the dto layer (default \"generated/dto\")")
	flag.String(flagRouterPkg, "", "module-relative package path for the router layer (default \"generated/router\")")
	flag.String(flagAPIPkg, "", "module-relative package path for handler stubs (default \"api\")")
	flag.Parse()

	if outputConfig {
		fmt.Print(sampleConfig)
		return
	}

	var cfg gen.Config
	if configPath != "" {
		fc, err := loadFileConfig(configPath)
		if err != nil {
			fail(err)
		}
		cfg = fc.toGenConfig()
	}

	set := map[string]string{}
	flag.Visit(func(f *flag.Flag) {
		set[f.Name] = f.Value.String()
	})
	applyFlagOverrides(&cfg, set)

	if err := gen.Generate(cfg); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "apiggo:", err)
	os.Exit(1)
}

// fileConfig is the on-disk YAML shape of a generation run. It mirrors
// codegen.Config field-for-field; keeping it in the cmd layer lets the library
// package stay free of yaml tags. The OpenAPI document key is schemaPath.
type fileConfig struct {
	SchemaPath string `yaml:"schemaPath"`
	Module     string `yaml:"module"`
	Out        string `yaml:"out"`
	DTOPkg     string `yaml:"dtoPackage"`
	RouterPkg  string `yaml:"routerPackage"`
	APIPkg     string `yaml:"apiPackage"`
}

// loadFileConfig reads and strictly decodes a YAML config file. Unknown keys are
// a hard error so typos surface immediately rather than being silently ignored.
func loadFileConfig(path string) (fileConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return fileConfig{}, err
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	var fc fileConfig
	if err := dec.Decode(&fc); err != nil {
		return fileConfig{}, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return fc, nil
}

// toGenConfig maps the file config onto the library config. Defaults and
// required-field validation are intentionally left to codegen.Generate.
func (fc fileConfig) toGenConfig() gen.Config {
	return gen.Config{
		ContractPath: fc.SchemaPath,
		Module:       fc.Module,
		OutRoot:      fc.Out,
		DTOPkg:       fc.DTOPkg,
		RouterPkg:    fc.RouterPkg,
		APIPkg:       fc.APIPkg,
	}
}

// Flag names for the per-field CLI flags. Shared between main and the override
// logic so the two never drift.
const (
	flagSchema    = "schema"
	flagModule    = "module"
	flagOut       = "out"
	flagDTOPkg    = "dto-pkg"
	flagRouterPkg = "router-pkg"
	flagAPIPkg    = "api-pkg"
)

// applyFlagOverrides writes each explicitly-set flag value onto the matching
// config field. set holds only the flags the user actually passed (collected via
// flag.Visit), so absent flags never clobber config-file values.
func applyFlagOverrides(cfg *gen.Config, set map[string]string) {
	if v, ok := set[flagSchema]; ok {
		cfg.ContractPath = v
	}
	if v, ok := set[flagModule]; ok {
		cfg.Module = v
	}
	if v, ok := set[flagOut]; ok {
		cfg.OutRoot = v
	}
	if v, ok := set[flagDTOPkg]; ok {
		cfg.DTOPkg = v
	}
	if v, ok := set[flagRouterPkg]; ok {
		cfg.RouterPkg = v
	}
	if v, ok := set[flagAPIPkg]; ok {
		cfg.APIPkg = v
	}
}

// sampleConfig is the documented starter file emitted by -output-config. The
// values shown are the defaults codegen.applyDefaults fills in when a key is
// omitted; schemaPath and module are required and have no default.
const sampleConfig = `# apiggo generator configuration.
# Run: apiggo -config apiggo.yaml

# Path to the OpenAPI document (YAML or JSON). Required.
schemaPath: api/openapi.yaml

# Go module path of the target project, used to build generated imports. Required.
module: github.com/acme/svc

# Output root directory the generated tree is written under.
out: .

# Module-relative package paths for the three generated layers.
dtoPackage: generated/dto
routerPackage: generated/router
apiPackage: api
`
