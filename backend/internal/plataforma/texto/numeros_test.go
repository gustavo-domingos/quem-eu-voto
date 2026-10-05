package texto

import (
	"testing"
)

func TestValorDecimal(t *testing.T) {
	casos := map[string]float64{"3176572.53": 3176572.53, "1.234,56": 1234.56, "11562724": 11562724, "-1": 0}
	for entrada, esperado := range casos {
		if got := ValorDecimal(entrada); got != esperado {
			t.Errorf("valorDecimal(%q) = %v, esperado %v", entrada, got, esperado)
		}
	}
}
