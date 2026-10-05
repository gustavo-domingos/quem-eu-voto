package tse

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/planilha"
	"quemeuvoto/internal/plataforma/texto"
)

const (
	urlVagasTSE = "https://cdn.tse.jus.br/estatistica/sead/odsele/consulta_vagas/consulta_vagas_%d.zip"
	csvVagasTSE = "consulta_vagas_%d_BRASIL.csv"
)

type IndiceVagas struct {
	porUF        map[string]map[string]int // UF ("BR" para Presidente) -> chave do cargo -> vagas em 2026
	porMunicipio map[string]map[string]int // código do município -> "prefeito"/"vereador" -> vagas em 2024
}

func CarregarVagas(ctx context.Context) (*IndiceVagas, error) {
	idx := &IndiceVagas{porUF: map[string]map[string]int{}, porMunicipio: map[string]map[string]int{}}
	somar := func(m map[string]map[string]int, chave, cargo string, n int) {
		if m[chave] == nil {
			m[chave] = map[string]int{}
		}
		m[chave][cargo] += n
	}

	err := planilha.LerDoZip(ctx, fmt.Sprintf(urlVagasTSE, dominio.AnoEleicao), fmt.Sprintf(csvVagasTSE, dominio.AnoEleicao), func(cab map[string]int, l []string) {
		cargo, ok := dominio.CargosPrincipais[texto.TituloCargo(planilha.Campo(cab, l, "DS_CARGO"))]
		if !ok {
			return
		}
		n, _ := strconv.Atoi(planilha.Campo(cab, l, "QT_VAGA"))
		somar(idx.porUF, planilha.Campo(cab, l, "SG_UF"), cargo, n)
	})
	if err != nil {
		return nil, fmt.Errorf("vagas %d: %w", dominio.AnoEleicao, err)
	}

	err = planilha.LerDoZip(ctx, fmt.Sprintf(urlVagasTSE, dominio.AnoMunicipal), fmt.Sprintf(csvVagasTSE, dominio.AnoMunicipal), func(cab map[string]int, l []string) {
		// Só as eleições regulares; as suplementares repetem vagas já contadas.
		if !strings.HasPrefix(planilha.Campo(cab, l, "DS_ELEICAO"), "Eleições Municipais") {
			return
		}
		cargo := map[string]string{"Prefeito": "prefeito", "Vereador": "vereador"}[planilha.Campo(cab, l, "DS_CARGO")]
		if cargo == "" {
			return
		}
		n, _ := strconv.Atoi(planilha.Campo(cab, l, "QT_VAGA"))
		somar(idx.porMunicipio, planilha.Campo(cab, l, "SG_UE"), cargo, n)
	})
	if err != nil {
		return nil, fmt.Errorf("vagas %d: %w", dominio.AnoMunicipal, err)
	}
	log.Printf("vagas: %d UFs (2026), %d municípios (2024)", len(idx.porUF), len(idx.porMunicipio))
	return idx, nil
}

// VagasDTO diz quantos representantes o local elege em cada cargo.
type VagasDTO struct {
	Local          string         `json:"local"` // "BR", UF ou código do município
	Vagas2026      map[string]int `json:"vagas2026"`
	CadeirasSenado int            `json:"cadeirasSenado"` // total de senadores do estado (ou do país)
	Municipais     map[string]int `json:"municipais,omitempty"`
	Fonte          string         `json:"fonte"`
}

func (idx *IndiceVagas) De(uf, municipio string) VagasDTO {
	v := VagasDTO{Local: uf, Vagas2026: map[string]int{}, Fonte: "TSE – consulta de vagas (eleições 2026 e 2024)"}
	if uf == "TODOS" || uf == "" {
		v.Local = "BR"
		for u, cargos := range idx.porUF {
			for c, n := range cargos {
				if u != "BR" || c == "presidente" {
					v.Vagas2026[c] += n
				}
			}
		}
		v.CadeirasSenado = dominio.CadeirasSenadoPorUF * len(dominio.UFsValidas)
	} else {
		for c, n := range idx.porUF[uf] {
			v.Vagas2026[c] = n
		}
		v.Vagas2026["presidente"] = idx.porUF["BR"]["presidente"]
		v.CadeirasSenado = dominio.CadeirasSenadoPorUF
	}
	if municipio != "" && municipio != "TODOS" {
		v.Municipais = idx.porMunicipio[municipio]
	}
	return v
}
