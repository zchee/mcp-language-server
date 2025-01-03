package main

var notificationTemplate = `
// {{.GoName}} sends a {{.Category}} notification for {{.Name}}
func (w *Wrapper) {{.GoName}}({{if ne .RequestType "struct{}"}}params protocol.{{.RequestType}}{{end}}) error {
    {{if eq .RequestType "struct{}"}}
    return w.client.Notify("{{.Name}}", struct{}{})
    {{else}}
    return w.client.Notify("{{.Name}}", params)
    {{end}}
}
`
var requestTemplate = `
// {{.GoName}} sends a {{.Category}} request for {{.Name}}
{{- $method := . -}}
{{- if eq (len .ResponseTypes) 0 }}
func (w *Wrapper) {{.GoName}}({{if ne .RequestType "struct{}"}}params protocol.{{.RequestType}}{{end}}) error {
    {{if eq .RequestType "struct{}"}}
    return w.client.Call("{{.Name}}", struct{}{}, nil)
    {{else}}
    return w.client.Call("{{.Name}}", params, nil)
    {{end}}
}
{{- else if eq (len .ResponseTypes) 1 }}
func (w *Wrapper) {{.GoName}}({{if ne .RequestType "struct{}"}}params protocol.{{.RequestType}}{{end}}) ({{(index .ResponseTypes 0).GoType}}, error) {
    var result {{(index .ResponseTypes 0).GoType}}
    {{if eq .RequestType "struct{}"}}
    err := w.client.Call("{{.Name}}", struct{}{}, &result)
    {{else}}
    err := w.client.Call("{{.Name}}", params, &result)
    {{end}}
    return result, err
}
{{- else }}
// Returns: {{range $i, $rt := .ResponseTypes}}{{if $i}} or {{end}}{{$rt.GoType}}{{end}}
func (w *Wrapper) {{.GoName}}({{if ne .RequestType "struct{}"}}params protocol.{{.RequestType}}{{end}}) (interface{}, error) {
    // Make single call and get raw response
    var rawResult json.RawMessage
    {{if eq .RequestType "struct{}"}}
    err := w.client.Call("{{.Name}}", struct{}{}, &rawResult)
    {{else}}
    err := w.client.Call("{{.Name}}", params, &rawResult)
    {{end}}
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }

    {{ range $i, $rt := .ResponseTypes }}
    // Try type {{$rt.Type}}
    {
        var result{{$i}} {{$rt.GoType}}
        decoder := json.NewDecoder(bytes.NewReader(rawResult))
        decoder.UseNumber()
        decoder.DisallowUnknownFields()
        if err := decoder.Decode(&result{{$i}}); err == nil {
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
    return nil, fmt.Errorf("response did not match any expected type")
}
{{- end }}
`

