package texto

import (
	"strconv"
	"strings"
)

// ValorBR converte "1.234,56" em 1234.56.
func ValorBR(s string) float64 {
	s = strings.ReplaceAll(strings.ReplaceAll(s, ".", ""), ",", ".")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// ValorDecimal aceita os dois formatos que o TSE usa: "1.234,56" e "1234.56".
// Valores negativos (marcadores como -1) contam como 0.
func ValorDecimal(s string) float64 {
	if strings.Contains(s, ",") {
		return ValorBR(s)
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}
