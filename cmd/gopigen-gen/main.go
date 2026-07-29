// Command gopigen-gen generates dto, router-glue and handler-stub code from an
// OpenAPI contract.
//
// Usage:
//
//	gopigen-gen -contract api.yaml -module github.com/acme/svc -out .
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/siyoga/gopigen/gen"
)

func main() {
	var cfg gen.Config

	flag.StringVar(&cfg.ContractPath, "contract", "", "path to the OpenAPI contract (YAML or JSON) [required]")
	flag.StringVar(&cfg.Module, "module", "", "Go module path of the target project, for generated imports [required]")
	flag.StringVar(&cfg.OutRoot, "out", ".", "output root directory")
	flag.StringVar(&cfg.DTOPkg, "dto-pkg", "generated/dto", "module-relative package path for the dto layer")
	flag.StringVar(&cfg.RouterPkg, "router-pkg", "generated/router", "module-relative package path for the router layer")
	flag.StringVar(&cfg.APIPkg, "api-pkg", "api", "module-relative package path for handler stubs")
	flag.Parse()

	if err := gen.Generate(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "gopigen-gen:", err)
		os.Exit(1)
	}
}
