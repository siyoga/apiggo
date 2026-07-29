package gen

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/getkin/kin-openapi/openapi3"
)

// Spec is the generator-facing view of a whole contract.
type Spec struct {
	Named      []NamedType // all dto types, sorted by name
	Operations []Operation
}

// Operation is one contract operation to emit a descriptor + stub for.
type Operation struct {
	ID      string // PascalCase base (from operationId)
	Method  string // "GET"
	Pattern string // "/users/{id}" (Go 1.22 mux syntax matches OpenAPI)
	Params  []Param
	HasBody bool
	InType  string // "<ID>In"
	OutType string // "<ID>Out"
	Errors  []string
}

// Param is a path/query/header parameter carried into the In struct.
type Param struct {
	Wire     string // wire name ("id")
	GoName   string // field name ("Id")
	In       string // "path" | "query" | "header"
	Prim     string // "string"|"int64"|"int32"|"float64"|"float32"|"bool"
	Required bool
}

var methodOrder = []string{
	http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
	http.MethodDelete, http.MethodHead, http.MethodOptions, http.MethodTrace,
}

// buildSpec turns a validated contract into the generator IR.
func buildSpec(doc *openapi3.T) (*Spec, error) {
	m := newMapper()

	// Register component schemas first so $ref names resolve to them and output
	// order is stable.
	if doc.Components != nil {
		for _, name := range sortedKeys(doc.Components.Schemas) {
			ref := doc.Components.Schemas[name]
			if ref.Value != nil {
				m.registerNamed(pascal(name), ref.Value)
			}
		}
	}

	var ops []Operation
	paths := doc.Paths.Map()
	for _, p := range sortedKeys(paths) {
		item := paths[p]
		operations := item.Operations()
		for _, method := range methodOrder {
			op := operations[method]
			if op == nil {
				continue
			}
			built, err := m.buildOp(method, p, op)
			if err != nil {
				return nil, err
			}
			ops = append(ops, built)
		}
	}

	return &Spec{Named: m.named(), Operations: ops}, nil
}

func (m *mapper) buildOp(method, path string, op *openapi3.Operation) (Operation, error) {
	if op.OperationID == "" {
		return Operation{}, fmt.Errorf("operation %s %s has no operationId (required for stable names)", method, path)
	}
	id := pascal(op.OperationID)

	var params []Param
	inFields := make([]Field, 0)

	for _, pref := range op.Parameters {
		p := pref.Value
		if p == nil || (p.In != "path" && p.In != "query" && p.In != "header") {
			continue
		}
		goName := pascal(p.Name)
		prim := primOf(p.Schema)
		params = append(params, Param{
			Wire:     p.Name,
			GoName:   goName,
			In:       p.In,
			Prim:     prim,
			Required: p.Required || p.In == "path", // path params are always required
		})
		inFields = append(inFields, Field{
			GoName: goName,
			Type:   prim,
			JSON:   fmt.Sprintf("json:%q", p.Name),
		})
	}

	hasBody := false
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		if mt := op.RequestBody.Value.Content.Get("application/json"); mt != nil && mt.Schema != nil {
			bodyType := m.goType(mt.Schema, id+"Body")
			inFields = append(inFields, Field{GoName: "Body", Type: bodyType, JSON: `json:"-"`})
			hasBody = true
		}
	}

	m.put(&NamedType{Name: id + "In", Struct: inFields})
	m.buildOut(id+"Out", op)
	errNames := m.buildErrors(id, op)

	return Operation{
		ID:      id,
		Method:  method,
		Pattern: path,
		Params:  params,
		HasBody: hasBody,
		InType:  id + "In",
		OutType: id + "Out",
		Errors:  errNames,
	}, nil
}

// buildOut registers the <ID>Out type: a struct if the 200 body is an inline
// object, an alias otherwise, or an empty struct when there is no 200 body.
func (m *mapper) buildOut(name string, op *openapi3.Operation) {
	schema := successSchema(op)
	if schema == nil {
		m.put(&NamedType{Name: name, Struct: []Field{}})
		return
	}
	expr := m.goType(schema, name)
	if expr == name {
		return // goType already registered a struct literally named <name>
	}
	m.put(&NamedType{Name: name, Alias: expr})
}

// buildErrors emits a distinct APIError type per non-2xx response.
func (m *mapper) buildErrors(id string, op *openapi3.Operation) []string {
	if op.Responses == nil {
		return nil
	}

	respMap := op.Responses.Map()
	var names []string
	for _, key := range sortedKeys(respMap) {
		code, err := strconv.Atoi(key)
		if err != nil || code < 400 {
			continue
		}

		name := id + statusWord(code)
		var fields []Field
		if resp := respMap[key].Value; resp != nil {
			if mt := resp.Content.Get("application/json"); mt != nil && mt.Schema != nil && mt.Schema.Value != nil {
				fields = m.fields(name, mt.Schema.Value)
			}
		}

		msg := fmt.Sprintf("%q", http.StatusText(code))
		if hasStringField(fields, "Message") {
			msg = "e.Message"
		}

		m.put(&NamedType{
			Name:        name,
			Struct:      fields,
			APIError:    true,
			Status:      code,
			MessageExpr: msg,
		})
		names = append(names, name)
	}
	return names
}

func successSchema(op *openapi3.Operation) *openapi3.SchemaRef {
	if op.Responses == nil {
		return nil
	}
	for _, code := range []string{"200", "201"} {
		if rr := op.Responses.Map()[code]; rr != nil && rr.Value != nil {
			if mt := rr.Value.Content.Get("application/json"); mt != nil {
				return mt.Schema
			}
		}
	}
	return nil
}

func primOf(ref *openapi3.SchemaRef) string {
	if ref == nil || ref.Value == nil {
		return "string"
	}
	s := ref.Value
	switch {
	case s.Type.Is("integer"):
		if s.Format == "int32" {
			return "int32"
		}
		return "int64"
	case s.Type.Is("number"):
		if s.Format == "float" {
			return "float32"
		}
		return "float64"
	case s.Type.Is("boolean"):
		return "bool"
	default:
		return "string"
	}
}

func hasStringField(fields []Field, goName string) bool {
	for _, f := range fields {
		if f.GoName == goName && f.Type == "string" {
			return true
		}
	}
	return false
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
