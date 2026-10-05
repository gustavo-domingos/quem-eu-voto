package camara

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/cache"
	"quemeuvoto/internal/plataforma/planilha"
	"quemeuvoto/internal/plataforma/texto"
)

const urlCotaCamara = "https://www.camara.leg.br/cotas/Ano-%d.csv.zip"

type IndiceCota struct {
	PorDeputado map[int]*dominio.GastosDeputado
}

func CarregarCota(ctx context.Context) (*IndiceCota, error) {
	idx := &IndiceCota{PorDeputado: map[int]*dominio.GastosDeputado{}}
	for ano := dominio.AnoInicioLeg; ano <= time.Now().Year(); ano++ {
		validade := 30 * 24 * time.Hour
		if ano == time.Now().Year() {
			validade = 20 * time.Hour
		}
		nome := fmt.Sprintf("camara_cota_%d.json", ano)
		porAno, ok := cache.Ler[map[int]*dominio.GastosDeputado](nome, validade)
		if !ok {
			m, err := agregarCotaAno(ctx, ano)
			if err != nil {
				return nil, fmt.Errorf("cota %d: %w", ano, err)
			}
			porAno = &m
			if err := cache.Gravar(nome, m); err != nil {
				log.Printf("cota %d: não foi possível gravar a cache: %v", ano, err)
			}
			log.Printf("cota parlamentar %d: %d deputados", ano, len(m))
		}
		for id, g := range *porAno {
			t := idx.PorDeputado[id]
			if t == nil {
				t = &dominio.GastosDeputado{PorAno: map[int]float64{}, PorCategoria: map[string]float64{}, PorFornecedor: map[string]float64{}}
				idx.PorDeputado[id] = t
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

func agregarCotaAno(ctx context.Context, ano int) (map[int]*dominio.GastosDeputado, error) {
	zr, err := planilha.BaixarZip(ctx, fmt.Sprintf(urlCotaCamara, ano))
	if err != nil {
		return nil, err
	}
	f, err := zr.Open(fmt.Sprintf("Ano-%d.csv", ano))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	res := map[int]*dominio.GastosDeputado{}
	err = planilha.Ler(f, func(cab map[string]int, l []string) {
		id, err := strconv.Atoi(planilha.Campo(cab, l, "ideCadastro"))
		if err != nil { // lideranças e gabinetes, sem deputado
			return
		}
		valor, err := strconv.ParseFloat(strings.ReplaceAll(planilha.Campo(cab, l, "vlrLiquido"), ",", "."), 64)
		if err != nil {
			return
		}
		g := res[id]
		if g == nil {
			g = &dominio.GastosDeputado{PorAno: map[int]float64{}, PorCategoria: map[string]float64{}, PorFornecedor: map[string]float64{}}
			res[id] = g
		}
		g.PorAno[ano] += valor
		g.PorCategoria[texto.FraseCapitalizada(strings.TrimSuffix(planilha.Campo(cab, l, "txtDescricao"), "."))] += valor
		g.PorFornecedor[planilha.Campo(cab, l, "txtFornecedor")] += valor
	})
	if err != nil {
		return nil, err
	}
	for _, g := range res {
		g.PorFornecedor = dominio.Maiores(g.PorFornecedor, 10)
	}
	return res, nil
}
