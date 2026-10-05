package senado

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/cache"
	"quemeuvoto/internal/plataforma/planilha"
	"quemeuvoto/internal/plataforma/rede"
	"quemeuvoto/internal/plataforma/texto"
)

// ---------- Cota do Senado (CEAPS) ----------

const urlCEAPS = "https://www.senado.gov.br/transparencia/LAI/verba/despesa_ceaps_%d.csv"

// IndiceCEAPS guarda os gastos da cota dos senadores por nome parlamentar normalizado.
type IndiceCEAPS struct {
	PorNome map[string]*dominio.GastosDeputado
}

func CarregarCEAPS(ctx context.Context) (*IndiceCEAPS, error) {
	idx := &IndiceCEAPS{PorNome: map[string]*dominio.GastosDeputado{}}
	for ano := dominio.AnoInicioLeg; ano <= time.Now().Year(); ano++ {
		validade := 30 * 24 * time.Hour
		if ano == time.Now().Year() {
			validade = 20 * time.Hour
		}
		nome := fmt.Sprintf("senado_ceaps_%d.json", ano)
		porAno, ok := cache.Ler[map[string]*dominio.GastosDeputado](nome, validade)
		if !ok {
			m, err := agregarCEAPS(ctx, ano)
			if err != nil {
				return nil, fmt.Errorf("CEAPS %d: %w", ano, err)
			}
			porAno = &m
			if err := cache.Gravar(nome, m); err != nil {
				log.Printf("CEAPS %d: não foi possível gravar a cache: %v", ano, err)
			}
			log.Printf("cota do Senado %d: %d senadores", ano, len(m))
		}
		for n, g := range *porAno {
			t := idx.PorNome[n]
			if t == nil {
				t = &dominio.GastosDeputado{PorAno: map[int]float64{}, PorCategoria: map[string]float64{}, PorFornecedor: map[string]float64{}}
				idx.PorNome[n] = t
			}
			for k, v := range g.PorAno {
				t.PorAno[k] += v
			}
			for k, v := range g.PorCategoria {
				t.PorCategoria[k] += v
			}
			for k, v := range g.PorFornecedor {
				t.PorFornecedor[k] += v
			}
		}
	}
	return idx, nil
}

func agregarCEAPS(ctx context.Context, ano int) (map[string]*dominio.GastosDeputado, error) {
	resp, err := rede.Baixar(ctx, fmt.Sprintf(urlCEAPS, ano), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// A primeira linha é "ULTIMA ATUALIZACAO"; o cabeçalho vem a seguir.
	leitor := bufio.NewReader(&planilha.LeitorLatin1{Origem: resp.Body})
	if _, err := leitor.ReadString('\n'); err != nil {
		return nil, err
	}
	res := map[string]*dominio.GastosDeputado{}
	err = planilha.Ler(leitor, func(cab map[string]int, l []string) {
		valor := texto.ValorDecimal(planilha.Campo(cab, l, "VALOR_REEMBOLSADO"))
		if valor <= 0 {
			return
		}
		n := texto.Normalizar(planilha.Campo(cab, l, "SENADOR"))
		g := res[n]
		if g == nil {
			g = &dominio.GastosDeputado{PorAno: map[int]float64{}, PorCategoria: map[string]float64{}, PorFornecedor: map[string]float64{}}
			res[n] = g
		}
		g.PorAno[ano] += valor
		cat := planilha.Campo(cab, l, "TIPO_DESPESA")
		if cat == "" {
			cat = "Outras despesas"
		}
		g.PorCategoria[cat] += valor
		g.PorFornecedor[planilha.Campo(cab, l, "FORNECEDOR")] += valor
	})
	if err != nil {
		return nil, err
	}
	for _, g := range res {
		g.PorFornecedor = dominio.Maiores(g.PorFornecedor, 10)
	}
	return res, nil
}
