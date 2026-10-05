package texto

import (
	"strings"
)

// DataISO converte DD/MM/AAAA em AAAA-MM-DD.
func DataISO(data string) string {
	p := strings.Split(data, "/")
	if len(p) != 3 {
		return data
	}
	return p[2] + "-" + p[1] + "-" + p[0]
}

// FormatarDataBR converte AAAA-MM-DD em DD/MM/AAAA.
func FormatarDataBR(iso string) string {
	p := strings.Split(iso, "-")
	if len(p) != 3 {
		return iso
	}
	return p[2] + "/" + p[1] + "/" + p[0]
}
