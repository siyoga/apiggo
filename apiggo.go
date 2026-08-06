// Package apiggo turns an OpenAPI contract into Go source: it loads the spec,
// builds an intermediate representation, and renders three layers — DTOs, HTTP
// adapters, and handler stubs. The generated HTTP layer depends only on the
// standard library plus the companion runtime in
// [github.com/siyoga/apiggo/server].
package apiggo

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Generate runs one generation pass: load contract -> build IR -> render layers.
func Generate(cfg Config) error {
	applyDefaults(&cfg)
	if cfg.ContractPath == "" {
		return fmt.Errorf("contract path is required")
	}
	if cfg.Module == "" {
		return fmt.Errorf("module path is required")
	}

	doc, err := load(cfg.ContractPath)
	if err != nil {
		return err
	}
	spec, err := buildSpec(doc)
	if err != nil {
		return err
	}
	return renderAll(cfg, spec)
}

func applyDefaults(cfg *Config) {
	if cfg.OutRoot == "" {
		cfg.OutRoot = "."
	}
	if cfg.DTOPkg == "" {
		cfg.DTOPkg = "generated/dto"
	}
	if cfg.RouterPkg == "" {
		cfg.RouterPkg = "generated/router"
	}
	if cfg.APIPkg == "" {
		cfg.APIPkg = "api"
	}
}

func renderAll(cfg Config, spec *Spec) error {
	dtoImport := cfg.Module + "/" + cfg.DTOPkg

	// Layer 1: dto (always overwritten).
	dto, err := render("dto.go.tmpl", dtoData{
		Package: path.Base(cfg.DTOPkg),
		Types:   spec.Named,
	})
	if err != nil {
		return err
	}
	if err := writeFile(filepath.Join(cfg.OutRoot, filepath.FromSlash(cfg.DTOPkg), "dto.go"), dto); err != nil {
		return err
	}

	// Layer 2: HTTP adapters (always overwritten).
	router, err := render("router.go.tmpl", routerData{
		Package:       path.Base(cfg.RouterPkg),
		RuntimeImport: runtimeImport,
		DtoImport:     dtoImport,
		NeedsStrconv:  needsStrconv(spec.Operations),
		Ops:           spec.Operations,
	})
	if err != nil {
		return err
	}
	if err := writeFile(filepath.Join(cfg.OutRoot, filepath.FromSlash(cfg.RouterPkg), "router.go"), router); err != nil {
		return err
	}

	// Layer 3: handler stubs (written once, never clobbered).
	for _, op := range spec.Operations {
		pkg := handlerPkg(op.ID)
		stub, err := render("handler.go.tmpl", handlerData{
			Package:   pkg,
			DtoImport: dtoImport,
			Op:        op,
		})
		if err != nil {
			return err
		}
		dst := filepath.Join(cfg.OutRoot, filepath.FromSlash(cfg.APIPkg), pkg, "handler.go")
		if err := writeFileIfAbsent(dst, stub); err != nil {
			return err
		}
	}

	return nil
}

// handlerPkg derives a lowercase, identifier-safe package name from an op ID.
func handlerPkg(id string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(id) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	name := b.String()
	if name == "" {
		name = "op"
	}
	return name
}

func writeFile(pathName string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(pathName), 0o755); err != nil {
		return err
	}
	return os.WriteFile(pathName, content, 0o644)
}

func writeFileIfAbsent(pathName string, content []byte) error {
	if _, err := os.Stat(pathName); err == nil {
		return nil // exists: never clobber business logic
	}
	return writeFile(pathName, content)
}
