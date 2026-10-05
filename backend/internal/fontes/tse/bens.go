package tse

import (
	"context"
	"fmt"
	"log"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/planilha"
	"quemeuvoto/internal/plataforma/texto"
)

const (
	urlBensTSE = "https://cdn.tse.jus.br/estatistica/sead/odsele/bem_candidato/bem_candidato_%d.zip"
	csvBensTSE = "bem_candidato_%d_BRASIL.csv"
)

type Bem struct {
	Tipo      string  `json:"tipo"`
	Descricao string  `json:"descricao"`
	Valor     float64 `json:"valor"`
}

// IndiceBens: bens declarados em 2026 (lista) e o total declarado em 2022, por SQ_CANDIDATO.
type IndiceBens struct {
	Bens2026  map[string][]Bem
	Total2026 map[string]float64
	Total2022 map[string]float64
}

func CarregarBens(ctx context.Context) (*IndiceBens, error) {
	idx := &IndiceBens{Bens2026: map[string][]Bem{}, Total2026: map[string]float64{}, Total2022: map[string]float64{}}

	err := planilha.LerDoZip(ctx, fmt.Sprintf(urlBensTSE, dominio.AnoEleicao), fmt.Sprintf(csvBensTSE, dominio.AnoEleicao), func(cab map[string]int, l []string) {
		sq := planilha.Campo(cab, l, "SQ_CANDIDATO")
		valor := texto.ValorBR(planilha.Campo(cab, l, "VR_BEM_CANDIDATO"))
		idx.Bens2026[sq] = append(idx.Bens2026[sq], Bem{
			Tipo:      planilha.Campo(cab, l, "DS_TIPO_BEM_CANDIDATO"),
			Descricao: texto.ResumirTexto(planilha.Campo(cab, l, "DS_BEM_CANDIDATO"), 200),
			Valor:     valor,
		})
		idx.Total2026[sq] += valor
	})
	if err != nil {
		return nil, fmt.Errorf("bens %d: %w", dominio.AnoEleicao, err)
	}

	err = planilha.LerDoZip(ctx, fmt.Sprintf(urlBensTSE, dominio.AnoGeral2022), fmt.Sprintf(csvBensTSE, dominio.AnoGeral2022), func(cab map[string]int, l []string) {
		idx.Total2022[planilha.Campo(cab, l, "SQ_CANDIDATO")] += texto.ValorBR(planilha.Campo(cab, l, "VR_BEM_CANDIDATO"))
	})
	if err != nil {
		return nil, fmt.Errorf("bens %d: %w", dominio.AnoGeral2022, err)
	}
	log.Printf("bens: %d candidatos em 2026, %d em 2022", len(idx.Bens2026), len(idx.Total2022))
	return idx, nil
}
