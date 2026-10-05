package transparencia

import (
	"sort"
	"strings"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/fontes/tse"
	"quemeuvoto/internal/plataforma/texto"
)

// Filtro são os critérios comuns às listagens de candidatos e de eleitos, tal como chegam do pedido.
type Filtro struct {
	Cargo, Uf, Municipio, Termo, Ordem string
	Partido, Genero, Faixa, Nivel      string
	Em2026                             string // só para eleitos: todos|candidato|reeleicao|outro|nao
	IncluirInaptos, SoJustica          bool
	Pagina                             int
}

// normalizar aplica os valores por omissão e valida a UF.
func (f *Filtro) normalizar() error {
	f.Uf = strings.ToUpper(f.Uf)
	f.Termo = texto.Normalizar(f.Termo)
	if f.Pagina < 1 {
		f.Pagina = 1
	}
	if f.Uf == "" {
		f.Uf = "TODOS"
	}
	if f.Uf != "TODOS" && !dominio.UFsValidas[f.Uf] {
		return entradaInvalida("UF inválida")
	}
	if f.Municipio == "" {
		f.Municipio = "TODOS"
	}
	return nil
}

// aceitaPessoa aplica os filtros de partido, gênero e faixa etária.
func (f Filtro) aceitaPessoa(c *tse.Candidatura) bool {
	if f.Partido != "" && c.Partido != f.Partido {
		return false
	}
	if f.Genero != "" && c.Genero != f.Genero {
		return false
	}
	if f.Faixa != "" {
		anos, ok := idade(c.Nascimento, time.Now())
		if !ok || faixaEtaria(anos) != f.Faixa {
			return false
		}
	}
	return true
}

var faixasEtarias = []string{"Até 39", "40–49", "50–59", "60–69", "70 ou mais"}

func faixaEtaria(anos int) string {
	return faixasEtarias[min(max((anos-30)/10, 0), len(faixasEtarias)-1)]
}

// partidosDe lista as siglas presentes, para o filtro de partido.
func partidosDe(lista []*tse.Candidatura) []string {
	vistos := map[string]bool{}
	var res []string
	for _, c := range lista {
		if c.Partido != "" && !vistos[c.Partido] {
			vistos[c.Partido] = true
			res = append(res, c.Partido)
		}
	}
	sort.Strings(res)
	return res
}

// filtrarEPaginar aplica o filtro, ordena e devolve a página pedida e o total.
// aceitar, se não for nil, é um critério extra (ex.: ter registos na Justiça).
// metricas guarda valores numéricos por SQ para ordenar (ex.: metricas["patrimonio"][sq]).
func filtrarEPaginar(lista []*tse.Candidatura, f Filtro, aceitar func(*tse.Candidatura) bool, metricas map[string]map[string]float64) (pagina []*tse.Candidatura, total int) {
	var filtrados []*tse.Candidatura
	for _, c := range lista {
		if c.ChaveCargo != f.Cargo || (f.Uf != "TODOS" && c.UF != f.Uf) || (!c.Apto && !f.IncluirInaptos) {
			continue
		}
		if f.Municipio != "TODOS" && c.Ue != f.Municipio {
			continue
		}
		if f.Termo != "" && !strings.Contains(c.ChaveBusca, f.Termo) && !strings.HasPrefix(c.Numero, f.Termo) {
			continue
		}
		if !f.aceitaPessoa(c) || (aceitar != nil && !aceitar(c)) {
			continue
		}
		filtrados = append(filtrados, c)
	}
	ordenarCandidatos(filtrados, f.Ordem, metricas)

	inicio := min((f.Pagina-1)*porPagina, len(filtrados))
	fim := min(inicio+porPagina, len(filtrados))
	return filtrados[inicio:fim], len(filtrados)
}

func totalPaginas(total int) int {
	return (total + porPagina - 1) / porPagina
}

func ordenarCandidatos(lista []*tse.Candidatura, ordem string, metricas map[string]map[string]float64) {
	hoje := time.Now()
	numero := func(c *tse.Candidatura, chave string) *float64 {
		if chave == "idade" {
			if a, ok := idade(c.Nascimento, hoje); ok {
				v := float64(a)
				return &v
			}
			return nil
		}
		if v, ok := metricas[chave][c.ID]; ok {
			return &v
		}
		return nil
	}
	chave, crescente := strings.TrimSuffix(ordem, "-asc"), strings.HasSuffix(ordem, "-asc")
	sort.SliceStable(lista, func(i, j int) bool {
		a, b := lista[i], lista[j]
		switch chave {
		case "numero":
			if len(a.Numero) != len(b.Numero) {
				return len(a.Numero) < len(b.Numero)
			}
			if a.Numero != b.Numero {
				return a.Numero < b.Numero
			}
		case "partido":
			if a.Partido != b.Partido {
				return a.Partido < b.Partido
			}
		case "municipio":
			if a.ChaveMunicipio != b.ChaveMunicipio {
				return a.ChaveMunicipio < b.ChaveMunicipio
			}
		case "nome":
		default:
			if menor, decidido := compararNumeros(numero(a, chave), numero(b, chave), crescente); decidido {
				return menor
			}
		}
		return a.ChaveNome < b.ChaveNome
	})
}

// compararNumeros ordena valores que podem faltar; os que faltam vão sempre para o fim.
// Devolve (a vem antes de b, a comparação decidiu).
func compararNumeros(a, b *float64, crescente bool) (bool, bool) {
	switch {
	case a == nil && b == nil:
		return false, false
	case a == nil:
		return false, true
	case b == nil:
		return true, true
	case *a == *b:
		return false, false
	case crescente:
		return *a < *b, true
	default:
		return *a > *b, true
	}
}
