package tse

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/cache"
	"quemeuvoto/internal/plataforma/planilha"
	"quemeuvoto/internal/plataforma/texto"
	"quemeuvoto/internal/plataforma/zipremoto"
)

// Prestação de contas eleitoral dos candidatos de 2026 (TSE). Durante a campanha os
// candidatos entregam relatórios parciais; a prestação final é entregue até 30 dias
// depois da eleição, por isso os valores ainda podem crescer.
const (
	urlContasTSE     = "https://cdn.tse.jus.br/estatistica/sead/odsele/prestacao_contas/prestacao_de_contas_eleitorais_candidatos_%d.zip"
	validadeCampanha = 12 * time.Hour
	maxItensCampanha = 8
)

// GastosCampanha é o resumo das contas de campanha de uma candidatura.
type GastosCampanha struct {
	Despesas         float64         `json:"despesas"`        // despesas contratadas
	Receitas         float64         `json:"receitas"`        // recursos arrecadados
	DinheiroPublico  float64         `json:"dinheiroPublico"` // Fundo Especial (FEFC) + Fundo Partidário
	PorTipoDespesa   []dominio.Valor `json:"porTipoDespesa"`
	PorOrigemReceita []dominio.Valor `json:"porOrigemReceita"`
	Fornecedores     []dominio.Valor `json:"fornecedores"`
	Doadores         []dominio.Valor `json:"doadores"`
	UltimaPrestacao  string          `json:"ultimaPrestacao,omitempty"` // AAAA-MM-DD
}

// ResumoCampanhaDTO vai em cada cartão.
type ResumoCampanhaDTO struct {
	Despesas        float64  `json:"despesas"`
	DinheiroPublico float64  `json:"dinheiroPublico"`
	Limite          *float64 `json:"limite"`
}

type IndiceCampanha struct {
	PorSQ    map[string]*GastosCampanha `json:"porSQ"`
	GeradoEm time.Time                  `json:"geradoEm"`
}

func CarregarCampanha(ctx context.Context) (*IndiceCampanha, error) {
	nomeCache := fmt.Sprintf("campanha_%d.json", dominio.AnoEleicao)
	if idx, ok := cache.Ler[IndiceCampanha](nomeCache, validadeCampanha); ok {
		log.Printf("campanha %d: lida da cache em disco (%d candidaturas)", dominio.AnoEleicao, len(idx.PorSQ))
		return idx, nil
	}
	inicio := time.Now()
	z, err := zipremoto.Abrir(ctx, fmt.Sprintf(urlContasTSE, dominio.AnoEleicao))
	if err != nil {
		return nil, err
	}

	type acumulado struct {
		g                                      *GastosCampanha
		tipos, origens, fornecedores, doadores map[string]float64
	}
	porSQ := map[string]*acumulado{}
	obter := func(sq string) *acumulado {
		a, ok := porSQ[sq]
		if !ok {
			a = &acumulado{g: &GastosCampanha{}, tipos: map[string]float64{}, origens: map[string]float64{},
				fornecedores: map[string]float64{}, doadores: map[string]float64{}}
			porSQ[sq] = a
		}
		return a
	}
	ultima := func(a *acumulado, data string) {
		if d := texto.DataISO(data); d > a.g.UltimaPrestacao {
			a.g.UltimaPrestacao = d
		}
	}
	// Nomes do TSE vêm por vezes em maiúsculas, outras não: para agrupar, usa a versão da Receita (RFB).
	nome := func(cab map[string]int, l []string, rfb, informado string) string {
		if n := texto.SemMarcador(planilha.Campo(cab, l, rfb)); n != "" {
			return n
		}
		return strings.ToUpper(texto.SemMarcador(planilha.Campo(cab, l, informado)))
	}

	ler := func(arquivo string, fn func(cab map[string]int, l []string)) error {
		fluxo, err := z.AbrirFluxo(ctx, arquivo)
		if err != nil {
			return err
		}
		defer fluxo.Close()
		return planilha.Ler(&planilha.LeitorLatin1{Origem: fluxo}, fn)
	}

	err = ler(fmt.Sprintf("despesas_contratadas_candidatos_%d_BRASIL.csv", dominio.AnoEleicao), func(cab map[string]int, l []string) {
		valor := texto.ValorBR(planilha.Campo(cab, l, "VR_DESPESA_CONTRATADA"))
		if valor == 0 {
			return
		}
		a := obter(planilha.Campo(cab, l, "SQ_CANDIDATO"))
		a.g.Despesas += valor
		a.tipos[texto.FraseCapitalizada(texto.SemMarcador(planilha.Campo(cab, l, "DS_ORIGEM_DESPESA")))] += valor
		if f := nome(cab, l, "NM_FORNECEDOR_RFB", "NM_FORNECEDOR"); f != "" {
			a.fornecedores[f] += valor
		}
		ultima(a, planilha.Campo(cab, l, "DT_PRESTACAO_CONTAS"))
	})
	if err != nil {
		return nil, fmt.Errorf("despesas: %w", err)
	}

	err = ler(fmt.Sprintf("receitas_candidatos_%d_BRASIL.csv", dominio.AnoEleicao), func(cab map[string]int, l []string) {
		valor := texto.ValorBR(planilha.Campo(cab, l, "VR_RECEITA"))
		if valor == 0 {
			return
		}
		a := obter(planilha.Campo(cab, l, "SQ_CANDIDATO"))
		a.g.Receitas += valor
		fonte := planilha.Campo(cab, l, "DS_FONTE_RECEITA")
		origem := texto.SemMarcador(planilha.Campo(cab, l, "DS_ORIGEM_RECEITA"))
		switch {
		case strings.Contains(fonte, "FUNDO ESPECIAL"):
			a.g.DinheiroPublico += valor
			origem = "Fundo Eleitoral (FEFC, dinheiro público)"
		case strings.Contains(fonte, "FUNDO PARTID"):
			a.g.DinheiroPublico += valor
			origem = "Fundo Partidário (dinheiro público)"
		case origem == "":
			origem = "Não informada"
		}
		a.origens[origem] += valor
		if d := nome(cab, l, "NM_DOADOR_RFB", "NM_DOADOR"); d != "" {
			a.doadores[d] += valor
		}
		ultima(a, planilha.Campo(cab, l, "DT_PRESTACAO_CONTAS"))
	})
	if err != nil {
		return nil, fmt.Errorf("receitas: %w", err)
	}

	idx := &IndiceCampanha{PorSQ: make(map[string]*GastosCampanha, len(porSQ)), GeradoEm: time.Now()}
	for sq, a := range porSQ {
		g := a.g
		g.Despesas, g.Receitas, g.DinheiroPublico = math.Round(g.Despesas*100)/100, math.Round(g.Receitas*100)/100, math.Round(g.DinheiroPublico*100)/100
		g.PorTipoDespesa = dominio.OrdenarValores(a.tipos, maxItensCampanha)
		g.PorOrigemReceita = dominio.OrdenarValores(a.origens, maxItensCampanha)
		g.Fornecedores = dominio.OrdenarValores(a.fornecedores, maxItensCampanha)
		g.Doadores = dominio.OrdenarValores(a.doadores, maxItensCampanha)
		idx.PorSQ[sq] = g
	}
	log.Printf("campanha %d: %d candidaturas com contas em %s", dominio.AnoEleicao, len(idx.PorSQ), time.Since(inicio).Round(time.Second))
	if err := cache.Gravar(nomeCache, idx); err != nil {
		log.Printf("campanha: não foi possível gravar a cache: %v", err)
	}
	return idx, nil
}

func (idx *IndiceCampanha) Resumo(c *Candidatura) *ResumoCampanhaDTO {
	r := &ResumoCampanhaDTO{}
	if c.LimiteGastos > 0 {
		l := c.LimiteGastos
		r.Limite = &l
	}
	if idx != nil {
		if g := idx.PorSQ[c.ID]; g != nil {
			r.Despesas, r.DinheiroPublico = g.Despesas, g.DinheiroPublico
		}
	}
	if r.Despesas == 0 && r.Limite == nil {
		return nil
	}
	return r
}

// DespesasPorSQ é usado para ordenar candidatos pelo gasto de campanha.
func (idx *IndiceCampanha) DespesasPorSQ() map[string]float64 {
	m := map[string]float64{}
	if idx == nil {
		return m
	}
	for sq, g := range idx.PorSQ {
		m[sq] = g.Despesas
	}
	return m
}
