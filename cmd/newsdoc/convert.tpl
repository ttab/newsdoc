// Code generated by newsdoc rpc-conversion; DO NOT EDIT.

package {{.Package}}

import (
       "github.com/ttab/newsdoc"
)

{{range .Messages}}
// {{.Name}}FromRPC converts {{.Name}} protobuf messages to NewsDoc structures.
func {{.Name}}FromRPC(r *{{.Name}}) newsdoc.{{.Name}} {
    var n newsdoc.{{.Name}}

    if r == nil {
       return n
    }

{{range .Fields -}}
    {{ if eq .GoType "[]Block"}}

    for _, b := range r.{{.ProtoName}} {
        if b == nil {
           continue
        }

        n.{{.GoName}} = append(n.{{.GoName}}, BlockFromRPC(b))
    }

    {{else if eq .GoType "DataMap"}}

    if r.{{.GoName}} != nil {
       n.{{.GoName}} = make(newsdoc.DataMap)

       for k, v := range r.{{.ProtoName}} {
           n.{{.GoName}}[k] = v
       }
    }

    {{else}}
    n.{{.GoName}} = r.{{.ProtoName}}
    {{- end -}}
    {{end}}

    return n
}
{{end}}

{{range .Messages}}
// {{.Name}}ToRPC converts {{.Name}} protobuf messages to NewsDoc structures.
func {{.Name}}ToRPC(n newsdoc.{{.Name}}) *{{.Name}} {
    r := {{.Name}}{}

{{range .Fields -}}
    {{ if eq .GoType "[]Block"}}

    for _, b := range n.{{.GoName}} {
        r.{{.ProtoName}} = append(r.{{.ProtoName}}, BlockToRPC(b))
    }

    {{else if eq .GoType "DataMap"}}

    if n.{{.GoName}} != nil {
       r.{{.ProtoName}} = make(map[string]string)

       for k, v := range n.{{.GoName}} {
           r.{{.ProtoName}}[k] = v
       }
    }

    {{else}}
    r.{{.ProtoName}} = n.{{.GoName}}
    {{- end -}}
    {{end}}

    return &r
}
{{end}}