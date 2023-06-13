syntax = "proto3";

package {{.Package}};

{{range $k, $v := .Options -}}
option {{$k}} = {{$v | printf "%q"}};
{{end}}

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
