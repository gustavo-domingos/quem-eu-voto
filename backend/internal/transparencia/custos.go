package transparencia

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/fontes/camara"
	"quemeuvoto/internal/fontes/senado"
	"quemeuvoto/internal/plataforma/texto"
)

// Custo do mandato de deputados federais e senadores: o que existe em dados abertos
// (cotas parlamentares e funcionários de gabinete) mais o subsídio, fixado em lei.

// ---------- Subsídio (salário) ----------

// Valores do subsídio mensal de deputados e senadores (iguais nas duas Casas).
// Fonte: Decreto Legislativo nº 172/2022. Se houver reajuste, acrescente uma linha aqui.
var tabelaSubsidio = []struct {
	desde string // AAAA-MM-DD
	valor float64
}{
	{"2023-02-01", 33763.00},
	{"2023-04-01", 39293.32},
	{"2024-02-01", 41650.92},
	{"2025-02-01", 44008.52},
}

const fonteSubsidio = "Decreto Legislativo nº 172/2022"

func subsidioEm(data string) float64 {
	v := 0.0
	for _, s := range tabelaSubsidio {
		if data >= s.desde {
			v = s.valor
		}
	}
	return v
}

// SubsidioDTO é uma estimativa: soma o subsídio de cada dia em exercício (proporcional ao
// mês) desde o início da legislatura. Não inclui 13º, ajuda de custo nem descontos.
type SubsidioDTO struct {
	Mensal float64 `json:"mensal"` // valor atual
	Dias   int     `json:"dias"`   // dias em exercício contados
	Total  float64 `json:"total"`
	Desde  string  `json:"desde"`
	Fonte  string  `json:"fonte"`
}

func estimarSubsidio(periodos []dominio.Periodo, hoje time.Time) *SubsidioDTO {
	s := &SubsidioDTO{Fonte: fonteSubsidio, Mensal: subsidioEm(hoje.Format("2006-01-02"))}
	contados := map[string]bool{} // evita contar duas vezes dias de períodos sobrepostos
	for _, p := range periodos {
		ini, err := time.Parse("2006-01-02", max(p.Inicio, dominio.InicioLegislatura))
		if err != nil {
			continue
		}
		fim := hoje
		if p.Fim != "" {
			if f, err := time.Parse("2006-01-02", p.Fim); err == nil && f.Before(hoje) {
				fim = f
			}
		}
		for d := ini; !d.After(fim); d = d.AddDate(0, 0, 1) {
			dia := d.Format("2006-01-02")
			if contados[dia] {
				continue
			}
			contados[dia] = true
			diasNoMes := time.Date(d.Year(), d.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
			s.Total += subsidioEm(dia) / float64(diasNoMes)
			s.Dias++
			if s.Desde == "" || dia < s.Desde {
				s.Desde = dia
			}
		}
	}
	if s.Dias == 0 {
		return nil
	}
	s.Total = math.Round(s.Total*100) / 100
	return s
}

// ---------- Resumo do custo do mandato ----------

type CotaDTO struct {
	Nome     string          `json:"nome"`
	Total    float64         `json:"total"`
	PorGrupo []dominio.Valor `json:"porGrupo"`
	PorAno   []dominio.Valor `json:"porAno"`
}

type CustoMandatoDTO struct {
	Subsidio    *SubsidioDTO `json:"subsidio"`
	Cota        *CotaDTO     `json:"cota"`
	Secretarios *int         `json:"secretarios"` // pessoas contratadas no gabinete (Câmara)
	Total       float64      `json:"total"`       // subsídio estimado + cota
	NaoIncluido []string     `json:"naoIncluido"`
}

func resumirCota(nome string, g *dominio.GastosDeputado) *CotaDTO {
	if g == nil {
		return nil
	}
	c := &CotaDTO{Nome: nome, Total: math.Round(g.Total()*100) / 100, PorGrupo: dominio.AgruparCategorias(g.PorCategoria)}
	anos := make([]int, 0, len(g.PorAno))
	for a := range g.PorAno {
		anos = append(anos, a)
	}
	sort.Ints(anos)
	for _, a := range anos {
		c.PorAno = append(c.PorAno, dominio.Valor{Rotulo: fmt.Sprint(a), Valor: math.Round(g.PorAno[a]*100) / 100})
	}
	return c
}

func montarCusto(subsidio *SubsidioDTO, cota *CotaDTO, secretarios *int, naoIncluido []string) *CustoMandatoDTO {
	c := &CustoMandatoDTO{Subsidio: subsidio, Cota: cota, Secretarios: secretarios, NaoIncluido: naoIncluido}
	if subsidio != nil {
		c.Total += subsidio.Total
	}
	if cota != nil {
		c.Total += cota.Total
	}
	c.Total = math.Round(c.Total*100) / 100
	return c
}

var naoIncluidoCamara = []string{
	"Salários dos secretários parlamentares (verba de gabinete): a Câmara não publica o valor por deputado em dados abertos.",
	"Auxílio-moradia ou imóvel funcional, 13º salário e ajuda de custo.",
}

var naoIncluidoSenado = []string{
	"Salários dos assessores do gabinete: o Senado não publica o valor por senador em dados abertos.",
	"Auxílio-moradia ou imóvel funcional, 13º salário e ajuda de custo.",
}

// custoDeputado estima o custo do mandato de um deputado federal até hoje.
func (d *Servico) custoDeputado(ctx context.Context, depID int, det camara.DetalheDeputado) *CustoMandatoDTO {
	desde := dominio.InicioLegislatura
	if pres := d.presenca.Obter(ctx); pres != nil {
		if p, ok := pres.PorDeputado[depID]; ok && p.Desde != "" {
			desde = p.Desde
		}
	}
	var cota *CotaDTO
	if c := d.cota.Obter(ctx); c != nil {
		cota = resumirCota("Cota para o Exercício da Atividade Parlamentar (Câmara)", c.PorDeputado[depID])
	}
	var secretarios *int
	if g := d.gabinetes.Obter(ctx); g != nil && det.Gabinete() != "" {
		n := g.PorGabinete[det.Gabinete()]
		secretarios = &n
	}
	return montarCusto(estimarSubsidio([]dominio.Periodo{{Inicio: desde}}, time.Now()), cota, secretarios, naoIncluidoCamara)
}

// custoSenador estima o custo do mandato de um senador desde o início do exercício (ou de 2023).
func (d *Servico) custoSenador(ctx context.Context, s senado.Senador) *CustoMandatoDTO {
	var cota *CotaDTO
	if c := d.ceaps.Obter(ctx); c != nil {
		cota = resumirCota("Cota para o Exercício da Atividade Parlamentar dos Senadores (CEAPS)", c.PorNome[texto.Normalizar(s.Nome)])
	}
	return montarCusto(estimarSubsidio(s.Exercicios, time.Now()), cota, nil, naoIncluidoSenado)
}
