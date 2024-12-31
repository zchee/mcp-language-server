package main

var notificationTemplate = `
// {{.GoName}} sends a {{.Category}} notification for {{.Name}}
func (w *Wrapper) {{.GoName}}(params protocol.{{.RequestType}}) error {
    return w.client.Notify("{{.Name}}", params)
}
`

var requestTemplate = `
// {{.GoName}} sends a {{.Category}} request for {{.Name}}
{{- $method := . -}}
{{- if eq (len .ResponseTypes) 1 }}
func (w *Wrapper) {{.GoName}}(params protocol.{{.RequestType}}) ({{(index .ResponseTypes 0).GoType}}, error) {
    var result {{(index .ResponseTypes 0).GoType}}
    err := w.client.Call("{{.Name}}", params, &result)
    return result, err
}
{{- else }}
// Returns: {{range $i, $rt := .ResponseTypes}}{{if $i}} or {{end}}{{$rt.GoType}}{{end}}
func (w *Wrapper) {{.GoName}}(params protocol.{{.RequestType}}) (interface{}, error) {
    {{ range $i, $rt := .ResponseTypes }}
    // Try type {{$rt.Type}}
    {
        var result{{$i}} {{$rt.GoType}}
        err := w.client.Call("{{$method.Name}}", params, &result{{$i}})
        if err == nil {
            {{- if $rt.NeedsConvert }}
            {{- if $rt.IsSlice }}
            return convert{{$rt.Type}}SliceTo{{(index $method.ResponseTypes 0).Type}}Slice(result{{$i}}), nil
            {{- else }}
            return convert{{$rt.Type}}To{{(index $method.ResponseTypes 0).Type}}(result{{$i}}), nil
            {{- end }}
            {{- else }}
            return result{{$i}}, nil
            {{- end }}
        }
    }
    {{- end }}
    return nil, fmt.Errorf("all response type attempts failed")
}
{{- end }}
`