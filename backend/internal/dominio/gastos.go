package dominio

import (
	"sort"
	"strings"

	"quemeuvoto/internal/plataforma/texto"
)

// GastosDeputado soma o que o deputado gastou da Cota para o Exercício da Atividade Parlamentar (CEAP).
type GastosDeputado struct {
	PorAno        map[int]float64    `json:"porAno"`
	PorCategoria  map[string]float64 `json:"porCategoria"`
	PorFornecedor map[string]float64 `json:"porFornecedor"` // só os maiores de cada ano, para a cache ficar pequena
}

func (g *GastosDeputado) Total() float64 {
	t := 0.0
	for _, v := range g.PorAno {
		t += v
	}
	return t
}

// Maiores devolve as n entradas de maior valor.
func Maiores(m map[string]float64, n int) map[string]float64 {
	if len(m) <= n {
		return m
	}
	type par struct {
		k string
		v float64
	}
	lista := make([]par, 0, len(m))
	for k, v := range m {
		lista = append(lista, par{k, v})
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].v > lista[j].v })
	res := make(map[string]float64, n)
	for _, p := range lista[:n] {
		res[p.k] = p.v
	}
	return res
}

// ---------- Grupos de despesas das cotas ----------

// grupoDespesa junta as categorias da Câmara e do Senado em grupos comparáveis.
func grupoDespesa(categoria string) string {
	c := texto.Normalizar(categoria)
	switch {
	case strings.Contains(c, "PASSAGE"), strings.Contains(c, "LOCOMOCAO"), strings.Contains(c, "HOSPEDAGEM"),
		strings.Contains(c, "COMBUST"), strings.Contains(c, "TAXI"), strings.Contains(c, "VEICULO"), strings.Contains(c, "FRETAMENTO"),
		strings.Contains(c, "AERONAVE"), strings.Contains(c, "EMBARCAC"):
		return "Viagens e deslocamentos"
	case strings.Contains(c, "DIVULGACAO"):
		return "Divulgação do mandato"
	case strings.Contains(c, "ESCRITORIO"), strings.Contains(c, "IMOVE"), strings.Contains(c, "TELEFON"), strings.Contains(c, "POSTA"),
		strings.Contains(c, "ASSINATURA"), strings.Contains(c, "MATERIAL"):
		return "Escritório e gabinete"
	case strings.Contains(c, "CONSULTORI"), strings.Contains(c, "ASSESSORI"), strings.Contains(c, "TRABALHOS TECNICOS"), strings.Contains(c, "PESQUISA"):
		return "Consultorias e assessorias"
	case strings.Contains(c, "SEGURANCA"):
		return "Segurança"
	case strings.Contains(c, "ALIMENTACAO"):
		return "Alimentação"
	case strings.Contains(c, "CURSO"), strings.Contains(c, "EVENTO"), strings.Contains(c, "PARTICIPACAO"):
		return "Cursos e eventos"
	}
	return "Outras despesas"
}

func AgruparCategorias(porCategoria map[string]float64) []Valor {
	grupos := map[string]float64{}
	for cat, v := range porCategoria {
		grupos[grupoDespesa(cat)] += v
	}
	return OrdenarValores(grupos, 12)
}
