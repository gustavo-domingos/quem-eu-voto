package transparencia

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/fontes/tse"
	"quemeuvoto/internal/plataforma/texto"
)

// aceitaNivel filtra pela assiduidade (Câmara) ou participação em votações (Senado).
func aceitaNivel(e *EleitoDTO, nivel string) bool {
	if nivel == "" {
		return true
	}
	if e.Desempenho == nil || e.Desempenho.Percentual == nil {
		return false
	}
	p := *e.Desempenho.Percentual
	switch nivel {
	case "95":
		return p >= 95
	case "90":
		return p >= 90
	case "abaixo80":
		return p < 80
	}
	return true
}

// Cargos com eleitos em exercício e o período do mandato.
var mandatoPorCargo = map[string]string{
	"presidente":        "2023–2026",
	"governador":        "2023–2026",
	"deputado-federal":  "2023–2027",
	"deputado-estadual": "2023–2027",
	"prefeito":          "2025–2028",
	"vereador":          "2025–2028",
}

// Cargos executivos: quem concorre a outro cargo tem de deixar o mandato 6 meses antes da eleição.
var cargoExecutivo = map[string]bool{"presidente": true, "governador": true, "prefeito": true}

// EleitoDTO é quem ocupa o cargo hoje, com o resultado da eleição e o que faz em 2026.
type EleitoDTO struct {
	*tse.Candidatura
	AnoEleicao       int                `json:"anoEleicao"` // 0 quando não temos o resultado (ex.: senador eleito em 2018)
	Votacao          *dominio.Votacao   `json:"votacao"`
	Mandato          string             `json:"mandato"`
	Desempenho       *DesempenhoDTO     `json:"desempenho"`
	Candidaturas2026 []*tse.Candidatura `json:"candidaturas2026"`
	Nota             string             `json:"nota,omitempty"`
	Justica          ResumoJusticaDTO   `json:"justica"`
	CustoMandato     *float64           `json:"custoMandato"` // subsídio estimado + cota, até hoje
}

func (e *EleitoDTO) concorre2026() (concorre, mesmoCargo bool) {
	for _, c := range e.Candidaturas2026 {
		if !c.Apto {
			continue
		}
		concorre = true
		if c.ChaveCargo == e.ChaveCargo {
			mesmoCargo = true
		}
	}
	return
}

// anotarSaida explica que um titular do Executivo que concorre a outro cargo teve de deixar o mandato.
func (e *EleitoDTO) anotarSaida() {
	concorre, mesmo := e.concorre2026()
	if !cargoExecutivo[e.ChaveCargo] || !concorre || mesmo {
		return
	}
	for _, c := range e.Candidaturas2026 {
		if c.Apto && c.ChaveCargo != e.ChaveCargo {
			e.Nota = fmt.Sprintf("Concorre a %s em %d: a lei exige deixar o cargo 6 meses antes da eleição", c.Cargo, dominio.AnoEleicao)
			if len(e.Vices) > 0 {
				e.Nota += fmt.Sprintf(", por isso o mandato passou ao vice (%s)", e.Vices[0].NomeUrna)
			}
			e.Nota += "."
			return
		}
	}
}

func anoDe(data string) string {
	if len(data) >= 4 {
		return data[:4]
	}
	return "?"
}

// ResumoDTO são as estatísticas do conjunto filtrado (antes da paginação).
type ResumoDTO struct {
	Total            int                `json:"total"`
	Mulheres         int                `json:"mulheres"`
	ComGenero        int                `json:"comGenero"`
	IdadeMedia       *float64           `json:"idadeMedia"`
	FaixaEtaria      []dominio.Contagem `json:"faixaEtaria"`
	PorPartido       []dominio.Contagem `json:"porPartido"`
	Escolaridade     []dominio.Contagem `json:"escolaridade"`
	Candidatos2026   int                `json:"candidatos2026"`
	Reeleicao        int                `json:"reeleicao"`
	OutroCargo       int                `json:"outroCargo"`
	RotuloDesempenho string             `json:"rotuloDesempenho,omitempty"`
	MediaDesempenho  *float64           `json:"mediaDesempenho"`
	TotalProjetos    *int               `json:"totalProjetos"`
	ComRegistros     int                `json:"comRegistros"` // com registo oficial na Justiça/órgãos de controle
}

func contagensOrdenadas(m map[string]int, max int) []dominio.Contagem {
	lista := make([]dominio.Contagem, 0, len(m))
	for k, v := range m {
		lista = append(lista, dominio.Contagem{Rotulo: k, Total: v})
	}
	sort.Slice(lista, func(i, j int) bool {
		if lista[i].Total != lista[j].Total {
			return lista[i].Total > lista[j].Total
		}
		return lista[i].Rotulo < lista[j].Rotulo
	})
	if max > 0 && len(lista) > max {
		outros := 0
		for _, c := range lista[max:] {
			outros += c.Total
		}
		lista = append(lista[:max], dominio.Contagem{Rotulo: "Outros", Total: outros})
	}
	return lista
}

func resumir(lista []*EleitoDTO) ResumoDTO {
	r := ResumoDTO{Total: len(lista)}
	hoje := time.Now()
	partidos, escolaridade := map[string]int{}, map[string]int{}
	faixas := []string{"Até 39", "40–49", "50–59", "60–69", "70 ou mais"}
	porFaixa := map[string]int{}
	somaIdade, comIdade := 0, 0
	somaDesemp, comDesemp, projetos, comProjetos := 0.0, 0, 0, 0

	for _, e := range lista {
		if e.Justica.Total > 0 {
			r.ComRegistros++
		}
		partidos[e.Partido]++
		if e.Escolaridade != "" {
			escolaridade[e.Escolaridade]++
		}
		if e.Genero != "" {
			r.ComGenero++
			if e.Genero == "F" {
				r.Mulheres++
			}
		}
		if a, ok := idade(e.Nascimento, hoje); ok {
			somaIdade += a
			comIdade++
			i := min(max((a-30)/10, 0), len(faixas)-1)
			porFaixa[faixas[i]]++
		}
		if concorre, mesmo := e.concorre2026(); concorre {
			r.Candidatos2026++
			if mesmo {
				r.Reeleicao++
			} else {
				r.OutroCargo++
			}
		}
		if e.Desempenho != nil {
			r.RotuloDesempenho = e.Desempenho.Rotulo
			if e.Desempenho.Percentual != nil {
				somaDesemp += *e.Desempenho.Percentual
				comDesemp++
			}
			if e.Desempenho.Projetos != nil {
				projetos += *e.Desempenho.Projetos
				comProjetos++
			}
		}
	}

	if comIdade > 0 {
		m := math.Round(float64(somaIdade)/float64(comIdade)*10) / 10
		r.IdadeMedia = &m
		for _, f := range faixas {
			r.FaixaEtaria = append(r.FaixaEtaria, dominio.Contagem{Rotulo: f, Total: porFaixa[f]})
		}
	}
	if comDesemp > 0 {
		m := math.Round(somaDesemp/float64(comDesemp)*10) / 10
		r.MediaDesempenho = &m
	}
	if comProjetos > 0 {
		r.TotalProjetos = &projetos
	}
	r.PorPartido = contagensOrdenadas(partidos, 8)
	r.Escolaridade = contagensOrdenadas(escolaridade, 0)
	return r
}

type RespostaEleitosDTO struct {
	Eleitos      []*EleitoDTO `json:"eleitos"`
	Total        int          `json:"total"`
	Pagina       int          `json:"pagina"`
	TotalPaginas int          `json:"totalPaginas"`
	Resumo       ResumoDTO    `json:"resumo"`
	AnoEleicao   int          `json:"anoEleicao2026"`
	Avisos       []string     `json:"avisos"`
	Partidos     []string     `json:"partidos"` // opções para o filtro
}

func ordenarEleitos(lista []*EleitoDTO, ordem string) {
	hoje := time.Now()
	numero := func(e *EleitoDTO, chave string) *float64 {
		v := func(x float64) *float64 { return &x }
		d := e.Desempenho
		switch chave {
		case "votos":
			if e.Votacao != nil {
				return v(float64(e.Votacao.Votos))
			}
		case "idade":
			if a, ok := idade(e.Nascimento, hoje); ok {
				return v(float64(a))
			}
		case "indice":
			if d != nil {
				return d.IndiceAtuacao
			}
		case "desempenho":
			if d != nil {
				return d.Percentual
			}
		case "projetos":
			if d != nil && d.Projetos != nil {
				return v(float64(*d.Projetos))
			}
		case "votacoes":
			if d != nil && d.Votacoes != nil {
				return v(float64(*d.Votacoes))
			}
		case "gastos":
			if d != nil {
				return d.Gastos
			}
		case "governo":
			if d != nil {
				return d.AlinhamentoGoverno
			}
		case "custo":
			return e.CustoMandato
		}
		return nil
	}
	// "-asc" no fim inverte para crescente (ex.: "desempenho-asc" = menor assiduidade primeiro).
	chave, crescente := strings.TrimSuffix(ordem, "-asc"), strings.HasSuffix(ordem, "-asc")
	sort.SliceStable(lista, func(i, j int) bool {
		a, b := lista[i], lista[j]
		switch chave {
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

// eleitosDoCargo monta a lista completa (todas as UFs) de quem exerce o cargo.
func (d *Servico) eleitosDoCargo(ctx context.Context, cargo string) (lista []*EleitoDTO, avisos []string, ok bool) {
	cands := d.candidatos.Obter(ctx)
	buscar2026 := func(cpf, nomeNasc string) []*tse.Candidatura {
		if cands == nil {
			return nil
		}
		return cands.BuscarPor(cpf, nomeNasc)
	}
	if cands == nil {
		avisos = append(avisos, "Candidaturas de 2026 indisponíveis no momento.")
	}

	switch cargo {
	case "presidente", "governador", "deputado-estadual":
		g := d.geral2022.Obter(ctx)
		if g == nil {
			return nil, append(avisos, "Eleitos de 2022 indisponíveis no momento (o primeiro carregamento pode levar alguns minutos)."), ctx.Err() == nil
		}
		for _, c := range g.Lista {
			if c.ChaveCargo == cargo {
				lista = append(lista, &EleitoDTO{
					Candidatura: c, AnoEleicao: dominio.AnoGeral2022, Votacao: g.Votos.PorCandidato[c.ID],
					Mandato: mandatoPorCargo[cargo], Candidaturas2026: buscar2026(c.Cpf, c.NomeNasc),
				})
			}
		}

	case "prefeito", "vereador":
		m := d.municipal.Obter(ctx)
		if m == nil {
			return nil, append(avisos, "Eleitos municipais indisponíveis no momento."), ctx.Err() == nil
		}
		votos := d.votos2024.Obter(ctx)
		if votos == nil {
			avisos = append(avisos, "Votações de 2024 indisponíveis no momento.")
		}
		for _, c := range m.Lista(cargo) {
			e := &EleitoDTO{Candidatura: c, AnoEleicao: dominio.AnoMunicipal, Mandato: mandatoPorCargo[cargo],
				Candidaturas2026: buscar2026("", c.NomeNasc)}
			if votos != nil {
				e.Votacao = votos.PorCandidato[c.ID]
			}
			lista = append(lista, e)
		}

	case "deputado-federal":
		todos, err := d.deputadosEmExercicio(ctx)
		if err != nil {
			return nil, append(avisos, "Não foi possível consultar a Câmara dos Deputados."), ctx.Err() == nil
		}
		montados, okM := d.montarDeputados(ctx, todos)
		if !okM {
			return nil, nil, false
		}
		avisos = append(avisos, montados.Avisos...)
		detalhes := d.detalhesDe(ctx, todos)
		g := d.geral2022.Obter(ctx)
		votacoesCD, cota := d.votacoesCD.Obter(ctx), d.cota.Obter(ctx)
		for _, dep := range montados.Deputados {
			det := detalhes[dep.ID]
			c := &tse.Candidatura{
				ID: "camara-" + strconv.Itoa(dep.ID), Cargo: "Deputado Federal", NomeUrna: dep.Nome, Nome: det.NomeCivil,
				Partido: dep.Partido, UF: dep.UF, Foto: dep.Foto, Apto: true,
				ChaveCargo: "deputado-federal", Genero: dominio.GeneroCurto(det.Sexo), Escolaridade: det.Escolaridade,
				Nascimento: det.DataNascimento, Cpf: det.CPF, NomeNasc: dominio.ChaveNomeNasc(det.NomeCivil, det.DataNascimento),
			}
			c.ChaveNome = texto.Normalizar(c.NomeUrna)
			c.ChaveBusca = texto.Normalizar(c.NomeUrna + " " + c.Nome + " " + c.Partido)
			e := &EleitoDTO{Candidatura: c, AnoEleicao: dominio.AnoGeral2022, Mandato: mandatoPorCargo[cargo],
				Candidaturas2026: dep.Candidaturas, Desempenho: &DesempenhoDTO{Rotulo: "Assiduidade", Percentual: dep.Assiduidade, Projetos: dep.TotalProjetos}}
			if dep.Assiduidade != nil {
				e.Desempenho.Detalhe = fmt.Sprintf("%d/%d sessões", *dep.SessoesPresentes, *dep.SessoesTotal)
			}
			if votacoesCD != nil {
				if vd := votacoesCD.PorDeputado[dep.ID]; vd != nil {
					total := 0
					for _, n := range vd.Contagem {
						total += n
					}
					e.Desempenho.Votacoes = &total
					if vd.Comparaveis > 0 {
						a := math.Round(float64(vd.Alinhados)/float64(vd.Comparaveis)*1000) / 10
						e.Desempenho.AlinhamentoGoverno = &a
					}
				}
			}
			if cota != nil {
				if gd := cota.PorDeputado[dep.ID]; gd != nil {
					t := math.Round(gd.Total())
					e.Desempenho.Gastos = &t
				}
			}
			if g != nil {
				e.Votacao = g.VotacaoDe(det.CPF, dominio.ChaveNomeNasc(det.NomeCivil, det.DataNascimento))
			}
			if custo := d.custoDeputado(ctx, dep.ID, det); custo != nil {
				e.CustoMandato = &custo.Total
			}
			lista = append(lista, e)
		}

	case "senador":
		senadores := d.senadores.Obter(ctx)
		if senadores == nil {
			return nil, append(avisos, "Não foi possível consultar o Senado Federal."), ctx.Err() == nil
		}
		ids := make([]int, len(*senadores))
		for i, s := range *senadores {
			ids[i] = s.Codigo
		}
		estat := d.estatSenado.ObterVarios(ctx, ids)
		g := d.geral2022.Obter(ctx)
		for _, s := range *senadores {
			st, temEstat := estat[s.Codigo]
			c := &tse.Candidatura{
				ID: "senado-" + strconv.Itoa(s.Codigo), Cargo: "Senador", NomeUrna: s.Nome, Nome: s.NomeCompleto,
				Partido: s.Partido, UF: s.UF, Foto: s.Foto, Apto: true, Vices: s.Suplentes,
				ChaveCargo: "senador", Genero: dominio.GeneroCurto(s.Sexo), Nascimento: st.DataNascimento,
			}
			if s.Suplente {
				c.Situacao = "Suplente em exercício"
			}
			c.ChaveNome = texto.Normalizar(c.NomeUrna)
			c.ChaveBusca = texto.Normalizar(c.NomeUrna + " " + c.Nome + " " + c.Partido)
			nomeNasc := dominio.ChaveNomeNasc(s.NomeCompleto, st.DataNascimento)
			c.NomeNasc = nomeNasc
			e := &EleitoDTO{Candidatura: c, Mandato: anoDe(s.MandatoInicio) + "–" + anoDe(s.MandatoFim),
				Candidaturas2026: buscar2026("", nomeNasc)}
			switch {
			case s.Suplente:
			case strings.HasPrefix(s.MandatoInicio, "2023"):
				e.AnoEleicao = dominio.AnoGeral2022
				if g != nil {
					e.Votacao = g.VotacaoDe("", nomeNasc)
				}
			case strings.HasPrefix(s.MandatoInicio, "2019"):
				e.AnoEleicao = 2018
			}
			if custo := d.custoSenador(ctx, s); custo != nil {
				e.CustoMandato = &custo.Total
			}
			if temEstat {
				presente := st.Presente
				e.Desempenho = &DesempenhoDTO{Rotulo: "Participação", Percentual: st.Participacao(), Projetos: &st.Projetos,
					Votacoes: &presente, Detalhe: fmt.Sprintf("%d de %d votações nominais", st.Presente, st.Votacoes)}
			}
			lista = append(lista, e)
		}
	}

	calcularIndiceAtuacao(lista)
	sancoes := d.sancoes.Obter(ctx)
	if sancoes == nil {
		avisos = append(avisos, "Cadastros de sanções (CGU) e decisões da Justiça Eleitoral indisponíveis no momento.")
	}
	g := d.geral2022.Obter(ctx)
	for _, e := range lista {
		if e.Candidaturas2026 == nil {
			e.Candidaturas2026 = []*tse.Candidatura{}
		}
		e.anotarSaida()
		if sancoes != nil {
			cpf := d.cpfDe(ctx, e.Cpf, e.NomeNasc)
			sqs := []string{e.ID}
			for _, c := range e.Candidaturas2026 {
				sqs = append(sqs, c.ID)
			}
			if g != nil && cpf != "" {
				sqs = append(sqs, g.SqPorCPF[cpf])
			}
			e.Justica = resumoJustica(sancoes.Buscar(cpf, e.Nome, sqs...), cpf != "")
		}
	}
	return lista, avisos, ctx.Err() == nil
}

// ListarEleitos devolve uma página de quem exerce o cargo, com o resumo estatístico do
// conjunto filtrado (não só da página).
func (d *Servico) ListarEleitos(ctx context.Context, f Filtro) (*RespostaEleitosDTO, error) {
	if err := f.normalizar(); err != nil {
		return nil, err
	}
	if _, ok := mandatoPorCargo[f.Cargo]; !ok && f.Cargo != "senador" {
		return nil, entradaInvalida("Cargo inválido")
	}
	todos, avisos, ok := d.eleitosDoCargo(ctx, f.Cargo)
	if !ok {
		return nil, ctx.Err()
	}

	var filtrados []*EleitoDTO
	var doLocal []*tse.Candidatura
	for _, e := range todos {
		if f.Uf != "TODOS" && e.UF != f.Uf && f.Cargo != "presidente" {
			continue
		}
		if f.Municipio != "TODOS" && e.Ue != f.Municipio {
			continue
		}
		doLocal = append(doLocal, e.Candidatura)
		if !f.aceitaPessoa(e.Candidatura) || !aceitaNivel(e, f.Nivel) {
			continue
		}
		if f.Termo != "" && !strings.Contains(e.ChaveBusca, f.Termo) && !strings.HasPrefix(e.Numero, f.Termo) {
			continue
		}
		if f.SoJustica && e.Justica.Total == 0 {
			continue
		}
		if !aceitaSituacao2026(e, f.Em2026) {
			continue
		}
		filtrados = append(filtrados, e)
	}
	ordenarEleitos(filtrados, f.Ordem)

	inicio := min((f.Pagina-1)*porPagina, len(filtrados))
	fim := min(inicio+porPagina, len(filtrados))
	if avisos == nil {
		avisos = []string{}
	}
	return &RespostaEleitosDTO{
		Eleitos:      append([]*EleitoDTO{}, filtrados[inicio:fim]...),
		Total:        len(filtrados),
		Pagina:       f.Pagina,
		TotalPaginas: totalPaginas(len(filtrados)),
		Resumo:       resumir(filtrados),
		AnoEleicao:   dominio.AnoEleicao,
		Avisos:       avisos,
		Partidos:     partidosDe(doLocal),
	}, nil
}

// aceitaSituacao2026 aplica o filtro "em 2026": todos, candidato, reeleicao, outro (cargo) ou nao.
func aceitaSituacao2026(e *EleitoDTO, em2026 string) bool {
	concorre, mesmo := e.concorre2026()
	switch em2026 {
	case "candidato":
		return concorre
	case "reeleicao":
		return mesmo
	case "outro":
		return concorre && !mesmo
	case "nao":
		return !concorre
	}
	return true
}
