package dominio

import (
	"testing"
)

func TestGrupoDespesa(t *testing.T) {
	casos := map[string]string{
		"PASSAGEM AÉREA - SIGEPA": "Viagens e deslocamentos",
		"Locomoção, hospedagem, alimentação, combustíveis e lubrificantes":        "Viagens e deslocamentos",
		"MANUTENÇÃO DE ESCRITÓRIO DE APOIO À ATIVIDADE PARLAMENTAR":               "Escritório e gabinete",
		"Divulgação da atividade parlamentar":                                     "Divulgação do mandato",
		"Contratação de consultorias, assessorias, pesquisas, trabalhos técnicos": "Consultorias e assessorias",
	}
	for categoria, esperado := range casos {
		if got := grupoDespesa(categoria); got != esperado {
			t.Errorf("grupoDespesa(%q) = %q, esperado %q", categoria, got, esperado)
		}
	}
}
