package transparencia

import (
	"math"
	"sort"
)

// DesempenhoDTO resume a atuação no mandato, quando há dados (Câmara e Senado).
type DesempenhoDTO struct {
	Rotulo             string   `json:"rotulo"`
	Percentual         *float64 `json:"percentual"`
	Detalhe            string   `json:"detalhe,omitempty"`
	Projetos           *int     `json:"projetos"`
	Votacoes           *int     `json:"votacoes"`           // votações nominais em que registou voto
	AlinhamentoGoverno *float64 `json:"alinhamentoGoverno"` // só Câmara
	Gastos             *float64 `json:"gastos"`             // cota parlamentar desde 2023; só Câmara
	// IndiceAtuacao (0–100) é a média dos percentis de participação, projetos e votações
	// entre os colegas do mesmo cargo: 100 = melhor em tudo, 50 = na média.
	IndiceAtuacao *float64 `json:"indiceAtuacao"`
}

// calcularIndiceAtuacao preenche IndiceAtuacao comparando cada parlamentar com os demais.
func calcularIndiceAtuacao(lista []*EleitoDTO) {
	metricas := []func(*DesempenhoDTO) *float64{
		func(d *DesempenhoDTO) *float64 { return d.Percentual },
		func(d *DesempenhoDTO) *float64 {
			if d.Projetos == nil {
				return nil
			}
			v := float64(*d.Projetos)
			return &v
		},
		func(d *DesempenhoDTO) *float64 {
			if d.Votacoes == nil {
				return nil
			}
			v := float64(*d.Votacoes)
			return &v
		},
	}
	soma := map[*EleitoDTO]float64{}
	n := map[*EleitoDTO]int{}
	for _, ler := range metricas {
		var com []*EleitoDTO
		for _, e := range lista {
			if e.Desempenho != nil && ler(e.Desempenho) != nil {
				com = append(com, e)
			}
		}
		if len(com) < 2 {
			continue
		}
		sort.SliceStable(com, func(i, j int) bool { return *ler(com[i].Desempenho) < *ler(com[j].Desempenho) })
		// Percentil com empates: todos os iguais recebem a posição média.
		for i := 0; i < len(com); {
			j := i
			for j+1 < len(com) && *ler(com[j+1].Desempenho) == *ler(com[i].Desempenho) {
				j++
			}
			pct := float64(i+j) / 2 / float64(len(com)-1) * 100
			for k := i; k <= j; k++ {
				soma[com[k]] += pct
				n[com[k]]++
			}
			i = j + 1
		}
	}
	for e, s := range soma {
		// Sem participação calculada (ex.: assumiu há pouco) o índice seria enganador.
		if e.Desempenho.Percentual == nil || n[e] < 2 {
			continue
		}
		v := math.Round(s / float64(n[e]))
		e.Desempenho.IndiceAtuacao = &v
	}
}
