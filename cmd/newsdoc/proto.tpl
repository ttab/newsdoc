syntax = "proto3";

package {{.Package}};

{{range .Messages}}
{{- range .Comment}}{{.}}
{{end -}}
message {{.Name}} {
{{- range .Fields}}
  {{- range .Comment}}
  {{.}}{{end}}
  {{.Type}} {{.Name}} = {{.FieldNumber}};{{end}}
}

{{end}}
