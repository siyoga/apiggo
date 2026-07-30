package codegen

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

var tmpl = template.Must(template.New("apiggo").Funcs(template.FuncMap{
	"methodConst": methodConst,
	"parseStmt":   renderParse,
}).ParseFS(templatesFS, "templates/*.tmpl"))

type dtoData struct {
	Package string
	Types   []NamedType
}

type routerData struct {
	Package       string
	RuntimeImport string
	DtoImport     string
	NeedsStrconv  bool
	Ops           []Operation
}

type handlerData struct {
	Package   string
	DtoImport string
	Op        Operation
}

// render executes a named template and gofmt-formats the result.
func render(name string, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, fmt.Errorf("execute %s: %w", name, err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Surface the unformatted source so template bugs are debuggable.
		return nil, fmt.Errorf("format %s: %w\n--- generated ---\n%s", name, err, buf.String())
	}
	return formatted, nil
}

// methodConst maps an HTTP method to its net/http constant ("http.MethodGet").
func methodConst(method string) string {
	return "http.Method" + pascal(strings.ToLower(method))
}

// renderParse emits the Go statement that reads one parameter into the In struct
// and, for non-string params, converts it (auto-400 on parse failure). Optional
// non-string params are only parsed when present, so an absent value stays zero.
func renderParse(p Param) string {
	src := map[string]string{
		"path":   fmt.Sprintf("r.PathValue(%q)", p.Wire),
		"query":  fmt.Sprintf("r.URL.Query().Get(%q)", p.Wire),
		"header": fmt.Sprintf("r.Header.Get(%q)", p.Wire),
	}[p.In]

	if p.Prim == "string" {
		return fmt.Sprintf("in.%s = %s", p.GoName, src)
	}

	assign := "in." + p.GoName + " = v"
	switch p.Prim {
	case "int32":
		assign = "in." + p.GoName + " = int32(v)"
	case "float32":
		assign = "in." + p.GoName + " = float32(v)"
	}

	body := fmt.Sprintf(`v, err := %s
if err != nil {
	s.HandleError(w, r, server.ErrBadRequest("invalid parameter: %s"))
	return
}
%s`, parseCall(p.Prim, "raw"), p.Wire, assign)

	if p.Required {
		return fmt.Sprintf("{\nraw := %s\n%s\n}", src, body)
	}
	return fmt.Sprintf("if raw := %s; raw != \"\" {\n%s\n}", src, body)
}

func parseCall(prim, arg string) string {
	switch prim {
	case "int64":
		return fmt.Sprintf("strconv.ParseInt(%s, 10, 64)", arg)
	case "int32":
		return fmt.Sprintf("strconv.ParseInt(%s, 10, 32)", arg)
	case "float64":
		return fmt.Sprintf("strconv.ParseFloat(%s, 64)", arg)
	case "float32":
		return fmt.Sprintf("strconv.ParseFloat(%s, 32)", arg)
	case "bool":
		return fmt.Sprintf("strconv.ParseBool(%s)", arg)
	}
	return ""
}

// needsStrconv reports whether any operation parses a non-string parameter.
func needsStrconv(ops []Operation) bool {
	for _, op := range ops {
		for _, p := range op.Params {
			if p.Prim != "string" {
				return true
			}
		}
	}
	return false
}
