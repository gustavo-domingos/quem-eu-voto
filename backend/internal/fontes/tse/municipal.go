package tse

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/planilha"
	"quemeuvoto/internal/plataforma/texto"
)

const (
	eleicaoMunicipalRegular = "Eleições Municipais 2024"
	IntervaloMunicipal      = 24 * time.Hour
)

type Municipio struct {
	Codigo string `json:"codigo"`
	Nome   string `json:"nome"`
}

// IndiceMunicipal guarda prefeitos e vereadores eleitos e a lista de municípios por UF.
type IndiceMunicipal struct {
	Prefeitos    []*Candidatura
	Vereadores   []*Candidatura
	Municipios   map[string][]Municipio
	AtualizadoEm time.Time
}

func (idx *IndiceMunicipal) Lista(chaveCargo string) []*Candidatura {
	if chaveCargo == "prefeito" {
		return idx.Prefeitos
	}
	return idx.Vereadores
}

var situacoesVereadorEleito = map[string]string{
	"ELEITO POR QP":    "Eleito por quociente partidário",
	"ELEITO POR MÉDIA": "Eleito por média",
}

// CarregarMunicipal lê o ficheiro de candidatos de 2024 (~245 MB descompactado) e
// guarda só os eleitos. Se um município teve eleição suplementar (eleição anulada),
// fica o prefeito eleito mais recentemente.
func CarregarMunicipal(ctx context.Context) (*IndiceMunicipal, error) {
	type prefeitoEleito struct {
		c     *Candidatura
		data  string // AAAA-MM-DD da eleição
		cdEle string
	}
	prefeitos := map[string]prefeitoEleito{} // por código do município
	vices := map[string]*Candidatura{}       // município|eleição|número
	nomes := map[string]map[string]string{}  // UF -> código -> nome
	idx := &IndiceMunicipal{AtualizadoEm: time.Now(), Municipios: map[string][]Municipio{}}

	err := planilha.LerDoZip(ctx, fmt.Sprintf(urlCandidatosTSE, dominio.AnoMunicipal), fmt.Sprintf(csvCandidatosTSE, dominio.AnoMunicipal), func(cab map[string]int, l []string) {
		uf, ue := planilha.Campo(cab, l, "SG_UF"), planilha.Campo(cab, l, "SG_UE")
		if nomes[uf] == nil {
			nomes[uf] = map[string]string{}
		}
		municipio := texto.TituloMunicipio(planilha.Campo(cab, l, "NM_UE"))
		nomes[uf][ue] = municipio

		cargo := planilha.Campo(cab, l, "DS_CARGO")
		sit := planilha.Campo(cab, l, "DS_SIT_TOT_TURNO")
		dsEleicao := planilha.Campo(cab, l, "DS_ELEICAO")
		data := texto.DataISO(planilha.Campo(cab, l, "DT_ELEICAO"))

		eleito := sit == "ELEITO" || situacoesVereadorEleito[sit] != ""
		if !eleito || (cargo != "PREFEITO" && cargo != "VICE-PREFEITO" && cargo != "VEREADOR") {
			return
		}

		c := candidaturaDaLinha(dominio.AnoMunicipal, uf, cab, l)
		c.Apto = true
		c.Municipio = municipio
		c.ChaveMunicipio = texto.Normalizar(municipio)
		c.ChaveBusca += " " + c.ChaveMunicipio
		if dsEleicao != eleicaoMunicipalRegular {
			c.Observacao = "Eleito(a) em eleição suplementar em " + texto.FormatarDataBR(data)
		}

		switch cargo {
		case "VEREADOR":
			c.ChaveCargo = "vereador"
			c.Situacao = situacoesVereadorEleito[sit]
			idx.Vereadores = append(idx.Vereadores, c)
		case "VICE-PREFEITO":
			vices[ue+"|"+planilha.Campo(cab, l, "CD_ELEICAO")+"|"+c.Numero] = c
		case "PREFEITO":
			c.ChaveCargo = "prefeito"
			c.Situacao = "Eleito"
			if planilha.Campo(cab, l, "NR_TURNO") == "2" {
				c.Situacao = "Eleito no 2º turno"
			}
			if atual, ok := prefeitos[ue]; !ok || data > atual.data {
				prefeitos[ue] = prefeitoEleito{c: c, data: data, cdEle: planilha.Campo(cab, l, "CD_ELEICAO")}
			}
		}
	})
	if err != nil {
		return nil, err
	}

	for ue, p := range prefeitos {
		if v, ok := vices[ue+"|"+p.cdEle+"|"+p.c.Numero]; ok {
			p.c.Vices = []dominio.Vice{{Cargo: "Vice-prefeito", NomeUrna: v.NomeUrna, Partido: v.Partido}}
		}
		idx.Prefeitos = append(idx.Prefeitos, p.c)
	}
	for uf, porCodigo := range nomes {
		for codigo, nome := range porCodigo {
			idx.Municipios[uf] = append(idx.Municipios[uf], Municipio{Codigo: codigo, Nome: nome})
		}
		sort.Slice(idx.Municipios[uf], func(i, j int) bool {
			return texto.Normalizar(idx.Municipios[uf][i].Nome) < texto.Normalizar(idx.Municipios[uf][j].Nome)
		})
	}
	if len(idx.Prefeitos) == 0 || len(idx.Vereadores) == 0 {
		return nil, fmt.Errorf("ficheiro do TSE de %d sem eleitos", dominio.AnoMunicipal)
	}
	log.Printf("municipal %d: %d prefeitos e %d vereadores eleitos", dominio.AnoMunicipal, len(idx.Prefeitos), len(idx.Vereadores))
	return idx, nil
}
