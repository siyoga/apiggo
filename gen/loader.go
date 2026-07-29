package gen

import (
	"context"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

// load parses and validates the OpenAPI contract, resolving $ref along the way.
// Only the openapi3 package is imported; the routers/gorillamux subpackage
// (which pulls gorilla/mux) is intentionally avoided.
func load(path string) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("load contract %q: %w", path, err)
	}

	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate contract %q: %w", path, err)
	}

	return doc, nil
}
