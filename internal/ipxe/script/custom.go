package script

import "net/url"

// CustomScript is the template for the custom script.
// It will either chain to a URL or execute an iPXE script.
var CustomScript = `#!ipxe

echo Loading custom Tinkerbell iPXE script...

{{- if .Chain }}
chain --autofree {{ .Chain }}
{{- else }}
{{ .Script }}
{{- end }}
`

// Custom holds either a URL to chain to or a script to execute.
// There is no validation of the script.
type Custom struct {
	Chain  *url.URL
	Script string
}
