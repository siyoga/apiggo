package codegen

// Config drives a single generation run.
type Config struct {
	// ContractPath is the path to the OpenAPI contract (YAML or JSON).
	ContractPath string
	// OutRoot is the directory the generated tree is written under.
	OutRoot string
	// Module is the Go module path of the target project, used to build import
	// paths in the generated code (e.g. "github.com/acme/svc").
	Module string

	// DTOPkg, RouterPkg and APIPkg are module-relative package paths for the
	// three layers.
	DTOPkg    string // default "generated/dto"
	RouterPkg string // default "generated/router"
	APIPkg    string // default "api"
}

// runtimeImport is the import path of the apiggo runtime the generated router
// glue targets.
const runtimeImport = "github.com/siyoga/apiggo/pkg/server"
