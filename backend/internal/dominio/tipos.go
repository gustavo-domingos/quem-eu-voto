package dominio

import (
	"math"
	"sort"
)

// Contagem é uma linha de um gráfico de barras do resumo.
type Contagem struct {
	Rotulo string `json:"rotulo"`
	Total  int    `json:"total"`
}

type Valor struct {
	Rotulo string  `json:"rotulo"`
	Valor  float64 `json:"valor"`
}

func OrdenarValores(m map[string]float64, max int) []Valor {
	lista := make([]Valor, 0, len(m))
	for k, v := range m {
		lista = append(lista, Valor{k, math.Round(v*100) / 100})
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].Valor > lista[j].Valor })
	if len(lista) > max {
		lista = lista[:max]
	}
	return lista
}
