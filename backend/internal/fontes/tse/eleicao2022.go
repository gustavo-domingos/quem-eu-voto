package tse

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/planilha"
)

const (
	validadeVotos2022 = 365 * 24 * time.Hour // resultado fechado: a cache em disco basta
	validadeVotos2024 = 7 * 24 * time.Hour   // ainda recebe eleições suplementares
)

// IndiceGeral2022 guarda os eleitos de 2022 que não vêm de outra fonte (Presidente,
// Governador e Deputado Estadual/Distrital), os votos de todos os candidatos de 2022
// e índices para encontrar o registo de 2022 de um deputado federal ou senador.
type IndiceGeral2022 struct {
	Lista          []*Candidatura
	Votos          *dominio.ResultadoVotacao
	SqPorCPF       map[string]string `json:"-"`
	sqPorNomeNasc  map[string]string
	CpfPorNomeNasc map[string]string `json:"-"`
}

func (g *IndiceGeral2022) VotacaoDe(cpf, nomeNasc string) *dominio.Votacao {
	sq, ok := g.SqPorCPF[cpf]
	if !ok || len(cpf) != 11 {
		sq = g.sqPorNomeNasc[nomeNasc]
	}
	return g.Votos.PorCandidato[sq]
}

func situacaoEleito(sit, turno string) string {
	if s, ok := situacoesVereadorEleito[sit]; ok {
		return s
	}
	if turno == "2" {
		return "Eleito no 2º turno"
	}
	return "Eleito"
}

func CarregarGeral2022(ctx context.Context) (*IndiceGeral2022, error) {
	votos, err := carregarVotos(ctx, dominio.AnoGeral2022, false, validadeVotos2022)
	if err != nil {
		return nil, fmt.Errorf("votos: %w", err)
	}
	g := &IndiceGeral2022{Votos: votos, SqPorCPF: map[string]string{}, sqPorNomeNasc: map[string]string{}, CpfPorNomeNasc: map[string]string{}}
	porSQ := map[string]bool{}
	var vices []*Candidatura

	err = planilha.LerDoZip(ctx, fmt.Sprintf(urlCandidatosTSE, dominio.AnoGeral2022), fmt.Sprintf(csvCandidatosTSE, dominio.AnoGeral2022), func(cab map[string]int, l []string) {
		c := candidaturaDaLinha(dominio.AnoGeral2022, planilha.Campo(cab, l, "SG_UF"), cab, l)
		if len(c.Cpf) == 11 {
			g.SqPorCPF[c.Cpf] = c.ID
			g.CpfPorNomeNasc[c.NomeNasc] = c.Cpf
		}
		g.sqPorNomeNasc[c.NomeNasc] = c.ID

		sit := planilha.Campo(cab, l, "DS_SIT_TOT_TURNO")
		if !strings.HasPrefix(sit, "ELEITO") || porSQ[c.ID] {
			return
		}
		c.Apto = true
		c.Situacao = situacaoEleito(sit, planilha.Campo(cab, l, "NR_TURNO"))
		switch c.Cargo {
		case "Presidente":
			c.ChaveCargo = "presidente"
		case "Governador":
			c.ChaveCargo = "governador"
		case "Deputado Estadual", "Deputado Distrital":
			c.ChaveCargo = "deputado-estadual"
		case "Vice-presidente", "Vice-governador":
			vices = append(vices, c)
			return
		default: // Senador e Deputado Federal vêm das APIs do Senado e da Câmara (quem está em exercício hoje)
			return
		}
		porSQ[c.ID] = true
		g.Lista = append(g.Lista, c)
	})
	if err != nil {
		return nil, err
	}

	titulares := map[string]*Candidatura{}
	for _, c := range g.Lista {
		titulares[c.Ue+"|"+c.Cargo+"|"+c.Numero] = c
	}
	for _, v := range vices {
		if t, ok := titulares[v.Ue+"|"+dominio.CargoDoTitular[v.Cargo]+"|"+v.Numero]; ok {
			t.Vices = []dominio.Vice{{Cargo: v.Cargo, NomeUrna: v.NomeUrna, Partido: v.Partido}}
		}
	}
	log.Printf("eleitos 2022: %d (presidente, governadores e deputados estaduais)", len(g.Lista))
	return g, nil
}

func CarregarVotos2024(ctx context.Context) (*dominio.ResultadoVotacao, error) {
	return carregarVotos(ctx, dominio.AnoMunicipal, true, validadeVotos2024)
}
