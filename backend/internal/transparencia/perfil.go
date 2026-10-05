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
	"quemeuvoto/internal/fontes/camara"
	"quemeuvoto/internal/fontes/noticias"
	"quemeuvoto/internal/fontes/sancoes"
	"quemeuvoto/internal/fontes/senado"
	"quemeuvoto/internal/fontes/tse"
	"quemeuvoto/internal/plataforma/texto"
)

// ---------- DTOs ----------

type PessoalDTO struct {
	Nome         string             `json:"nome"`
	NomeUrna     string             `json:"nomeUrna"`
	Idade        *int               `json:"idade"`
	Nascimento   string             `json:"nascimento,omitempty"`
	Naturalidade string             `json:"naturalidade,omitempty"`
	Genero       string             `json:"genero,omitempty"`
	Escolaridade string             `json:"escolaridade,omitempty"`
	EstadoCivil  string             `json:"estadoCivil,omitempty"`
	CorRaca      string             `json:"corRaca,omitempty"`
	Ocupacao     string             `json:"ocupacao,omitempty"`
	Email        string             `json:"email,omitempty"`
	Redes        []noticias.LinkDTO `json:"redes"`
}

type PatrimonioDTO struct {
	Total2026 float64   `json:"total2026"`
	Bens      []tse.Bem `json:"bens"`
	Total2022 *float64  `json:"total2022"`
	Variacao  *float64  `json:"variacao"` // % entre 2022 e 2026
}

type GastosDTO struct {
	Total          float64         `json:"total"`
	PorAno         []dominio.Valor `json:"porAno"`
	Categorias     []dominio.Valor `json:"categorias"`
	Fornecedores   []dominio.Valor `json:"fornecedores"`
	MediaDeputados float64         `json:"mediaDeputados"`
	Posicao        int             `json:"posicao"` // 1 = quem mais gastou
	De             int             `json:"de"`
}

type VotacoesDTO struct {
	Casa               string                `json:"casa"`
	Total              int                   `json:"total"`
	Contagem           []dominio.Contagem    `json:"contagem"`
	AlinhamentoGoverno *float64              `json:"alinhamentoGoverno"`
	Comparaveis        int                   `json:"comparaveis"`
	Recentes           []dominio.VotoRecente `json:"recentes"`
}

type MandatoDTO struct {
	Cargo      string            `json:"cargo"`
	Local      string            `json:"local"`
	Periodo    string            `json:"periodo"`
	Situacao   string            `json:"situacao,omitempty"`
	AnoEleicao int               `json:"anoEleicao"`
	Votacao    *dominio.Votacao  `json:"votacao"`
	Vices      []dominio.Vice    `json:"vices,omitempty"`
	Nota       string            `json:"nota,omitempty"`
	Desempenho *DesempenhoDTO    `json:"desempenho"`
	Votacoes   *VotacoesDTO      `json:"votacoes"`
	Gastos     *GastosDTO        `json:"gastos"`
	Projetos   []dominio.Projeto `json:"projetos"`
	Pagina     *noticias.LinkDTO `json:"pagina"`
	Custo      *CustoMandatoDTO  `json:"custo"`
}

type PerfilDTO struct {
	ID                string                    `json:"id"`
	Foto              string                    `json:"foto"`
	Partido           string                    `json:"partido"`
	UF                string                    `json:"uf"`
	Municipio         string                    `json:"municipio,omitempty"`
	Pessoal           PessoalDTO                `json:"pessoal"`
	Candidaturas2026  []*tse.Candidatura        `json:"candidaturas2026"`
	Mandato           *MandatoDTO               `json:"mandato"`
	Patrimonio        *PatrimonioDTO            `json:"patrimonio"`
	Historico         []*tse.CandidaturaPassada `json:"historico"`
	Indicadores       []Indicador               `json:"indicadores"`
	Justica           []sancoes.RegistroJustica `json:"justica"`
	JusticaVerificada bool                      `json:"justicaVerificada"` // havia CPF para cruzar com a CGU
	JusticaDataCGU    string                    `json:"justicaDataCGU,omitempty"`
	NomeBusca         string                    `json:"nomeBusca"`
	Propostas         []PropostaRefDTO          `json:"propostas"` // propostas de governo entregues ao TSE
	// Apuracao2026 traz o resultado (em tempo real) de cada candidatura de 2026, pela id.
	Apuracao2026 map[string]*tse.ResultadoCandidato `json:"apuracao2026"`
	// Campanha2026 traz as contas de campanha de cada candidatura de 2026, pela id.
	Campanha2026 map[string]*CampanhaPerfilDTO `json:"campanha2026"`
	AnoEleicao   int                           `json:"anoEleicao2026"`
	Avisos       []string                      `json:"avisos"`
}

// CampanhaPerfilDTO junta as contas declaradas, o limite legal e o custo por voto.
type CampanhaPerfilDTO struct {
	*tse.GastosCampanha
	Cargo        string   `json:"cargo"`
	Limite       *float64 `json:"limite"`
	CustoPorVoto *float64 `json:"custoPorVoto"`
}

// PropostaRefDTO aponta para uma proposta de governo (Presidente, Governador ou Prefeito).
type PropostaRefDTO struct {
	Rotulo string `json:"rotulo"`
	Ano    string `json:"ano"`
	UF     string `json:"uf"`
	SQ     string `json:"sq"`
}

// ---------- Montagem ----------

// pessoa reúne as chaves para cruzar fontes e os registos já encontrados.
type pessoa struct {
	cpf, nomeNasc string
	tse           *tse.Candidatura // registo do TSE com mais dados pessoais (2026 de preferência)
	deputado      *DeputadoDTO
	detalhe       camara.DetalheDeputado
	senador       *senado.Senador
	estatSenado   *senado.EstatSenado
	eleito        *tse.Candidatura // eleito de 2022 (pres/gov/dep. estadual) ou 2024 (prefeito/vereador)
	anoEleito     int
}

func nomeRede(u string) string {
	for _, r := range []string{"instagram", "facebook", "twitter", "x.com", "youtube", "tiktok", "linkedin", "threads"} {
		if strings.Contains(strings.ToLower(u), r) {
			if r == "x.com" {
				return "X"
			}
			return texto.FraseCapitalizada(r)
		}
	}
	return "Rede social"
}

func resumirVotos(casa string, v *camara.VotosDeputado) *VotacoesDTO {
	res := &VotacoesDTO{Casa: casa, Comparaveis: v.Comparaveis, Recentes: v.Recentes}
	for k, n := range v.Contagem {
		res.Total += n
		if k == "" {
			k = "Não informado"
		}
		res.Contagem = append(res.Contagem, dominio.Contagem{Rotulo: k, Total: n})
	}
	sort.Slice(res.Contagem, func(i, j int) bool { return res.Contagem[i].Total > res.Contagem[j].Total })
	if v.Comparaveis > 0 {
		a := float64(v.Alinhados) / float64(v.Comparaveis) * 100
		res.AlinhamentoGoverno = &a
	}
	return res
}

func resumirGastos(c *camara.IndiceCota, depID int, deputados []DeputadoDTO) *GastosDTO {
	g := c.PorDeputado[depID]
	if g == nil {
		return nil
	}
	res := &GastosDTO{Total: math.Round(g.Total()*100) / 100, Categorias: dominio.OrdenarValores(g.PorCategoria, 6), Fornecedores: dominio.OrdenarValores(g.PorFornecedor, 5)}
	for ano := dominio.AnoInicioLeg; ano <= time.Now().Year(); ano++ {
		res.PorAno = append(res.PorAno, dominio.Valor{Rotulo: strconv.Itoa(ano), Valor: math.Round(g.PorAno[ano]*100) / 100})
	}
	// Comparação com os deputados em exercício.
	var totais []float64
	for _, d := range deputados {
		if gd := c.PorDeputado[d.ID]; gd != nil {
			totais = append(totais, gd.Total())
		}
	}
	soma := 0.0
	for _, t := range totais {
		soma += t
		if t > g.Total() {
			res.Posicao++
		}
	}
	res.Posicao++
	res.De = len(totais)
	if len(totais) > 0 {
		res.MediaDeputados = math.Round(soma/float64(len(totais))*100) / 100
	}
	return res
}

func (d *Servico) montarPerfil(ctx context.Context, id string) (*PerfilDTO, error) {
	p := &pessoa{}
	cands := d.candidatos.Obter(ctx)
	g := d.geral2022.Obter(ctx)
	m := d.municipal.Obter(ctx)

	// Deputados e senadores atuais, para cruzar com qualquer ponto de partida.
	todos, _ := d.deputadosEmExercicio(ctx)
	montados, ok := d.montarDeputados(ctx, todos)
	if !ok {
		return nil, ctx.Err()
	}
	detalhes := d.detalhesDe(ctx, todos)
	var senadores []senado.Senador
	estat := map[int]senado.EstatSenado{}
	if s := d.senadores.Obter(ctx); s != nil {
		senadores = *s
		ids := make([]int, len(senadores))
		for i, sen := range senadores {
			ids[i] = sen.Codigo
		}
		estat = d.estatSenado.ObterVarios(ctx, ids)
	}
	deputadoPorID := map[int]*DeputadoDTO{}
	for i := range montados.Deputados {
		deputadoPorID[montados.Deputados[i].ID] = &montados.Deputados[i]
	}

	// 1. Ponto de partida.
	switch {
	case strings.HasPrefix(id, "camara-"):
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "camara-"))
		dep, ok := deputadoPorID[n]
		if !ok {
			return nil, nil
		}
		p.deputado, p.detalhe = dep, detalhes[n]
		p.cpf, p.nomeNasc = p.detalhe.CPF, dominio.ChaveNomeNasc(p.detalhe.NomeCivil, p.detalhe.DataNascimento)
	case strings.HasPrefix(id, "senado-"):
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "senado-"))
		for i := range senadores {
			if senadores[i].Codigo == n {
				p.senador = &senadores[i]
			}
		}
		if p.senador == nil {
			return nil, nil
		}
		st := estat[n]
		p.estatSenado = &st
		p.nomeNasc = dominio.ChaveNomeNasc(p.senador.NomeCompleto, st.DataNascimento)
	default:
		var base *tse.Candidatura
		if cands != nil {
			base = cands.PorSQ[id]
		}
		if base == nil && g != nil {
			for _, c := range g.Lista {
				if c.ID == id {
					base, p.eleito, p.anoEleito = c, c, dominio.AnoGeral2022
				}
			}
		}
		if base == nil && m != nil {
			for _, lista := range [][]*tse.Candidatura{m.Prefeitos, m.Vereadores} {
				for _, c := range lista {
					if c.ID == id {
						base, p.eleito, p.anoEleito = c, c, dominio.AnoMunicipal
					}
				}
			}
		}
		if base == nil {
			return nil, nil
		}
		p.tse = base
		if len(base.Cpf) == 11 {
			p.cpf = base.Cpf
		}
		p.nomeNasc = base.NomeNasc
	}

	// 2. Cruzamentos.
	mesma := func(cpf, nomeNasc string) bool {
		return (len(p.cpf) == 11 && cpf == p.cpf) || (nomeNasc != "" && nomeNasc == p.nomeNasc)
	}
	if p.deputado == nil {
		for depID, det := range detalhes {
			if mesma(det.CPF, dominio.ChaveNomeNasc(det.NomeCivil, det.DataNascimento)) {
				p.deputado, p.detalhe = deputadoPorID[depID], det
				if p.cpf == "" {
					p.cpf = det.CPF
				}
			}
		}
	}
	if p.senador == nil {
		for i, s := range senadores {
			if st, ok := estat[s.Codigo]; ok && mesma("", dominio.ChaveNomeNasc(s.NomeCompleto, st.DataNascimento)) {
				p.senador, p.estatSenado = &senadores[i], &st
			}
		}
	}
	var candidaturas []*tse.Candidatura
	if cands != nil {
		candidaturas = cands.BuscarPor(p.cpf, p.nomeNasc)
		// Os dados pessoais do registo de 2026 são os mais recentes (ocupação, escolaridade, partido).
		if len(candidaturas) > 0 && (p.tse == nil || cands.PorSQ[p.tse.ID] == nil) {
			p.tse = candidaturas[0]
		}
	}
	if p.eleito == nil && g != nil {
		for _, c := range g.Lista {
			if mesma(c.Cpf, c.NomeNasc) {
				p.eleito, p.anoEleito = c, dominio.AnoGeral2022
			}
		}
	}
	if p.eleito == nil && m != nil && p.nomeNasc != "" {
		for _, lista := range [][]*tse.Candidatura{m.Prefeitos, m.Vereadores} {
			for _, c := range lista {
				if c.NomeNasc == p.nomeNasc {
					p.eleito, p.anoEleito = c, dominio.AnoMunicipal
				}
			}
		}
	}
	if p.tse == nil && p.eleito != nil {
		p.tse = p.eleito
	}

	perfil := &PerfilDTO{ID: id, AnoEleicao: dominio.AnoEleicao, Candidaturas2026: candidaturas, Avisos: []string{}, Indicadores: []Indicador{}}
	if perfil.Candidaturas2026 == nil {
		perfil.Candidaturas2026 = []*tse.Candidatura{}
	}
	perfil.Propostas = []PropostaRefDTO{}
	for _, c := range perfil.Candidaturas2026 {
		if c.ChaveCargo == "presidente" || c.ChaveCargo == "governador" {
			perfil.Propostas = append(perfil.Propostas, PropostaRefDTO{
				Rotulo: fmt.Sprintf("Candidatura a %s (%d)", c.Cargo, dominio.AnoEleicao), Ano: strconv.Itoa(dominio.AnoEleicao), UF: c.UF, SQ: c.ID,
			})
		}
	}
	if p.eleito != nil && p.eleito.ChaveCargo == "prefeito" {
		perfil.Propostas = append(perfil.Propostas, PropostaRefDTO{
			Rotulo: fmt.Sprintf("Prefeito(a) de %s, eleito(a) em %d", p.eleito.Municipio, dominio.AnoMunicipal),
			Ano:    strconv.Itoa(dominio.AnoMunicipal), UF: p.eleito.UF, SQ: p.eleito.ID,
		})
	}
	d.preencherIdentidade(perfil, p)
	d.preencherMandato(ctx, perfil, p, montados.Deputados)
	d.preencherHistoricoEPatrimonio(ctx, perfil, p)
	gerarIndicadores(perfil, p, montados.Deputados)
	d.preencherJustica(ctx, perfil, p)
	perfil.Campanha2026 = map[string]*CampanhaPerfilDTO{}
	camp := d.campanha.Obter(ctx)
	perfil.Apuracao2026 = map[string]*tse.ResultadoCandidato{}
	for _, c := range perfil.Candidaturas2026 {
		res := d.apuracaoDe(ctx, c.ChaveCargo, []string{c.UF})[c.ID]
		if res != nil {
			perfil.Apuracao2026[c.ID] = res
		}
		if camp != nil {
			if g := camp.PorSQ[c.ID]; g != nil || c.LimiteGastos > 0 {
				dto := &CampanhaPerfilDTO{GastosCampanha: g, Cargo: c.Cargo}
				if dto.GastosCampanha == nil {
					dto.GastosCampanha = &tse.GastosCampanha{PorTipoDespesa: []dominio.Valor{}, PorOrigemReceita: []dominio.Valor{}, Fornecedores: []dominio.Valor{}, Doadores: []dominio.Valor{}}
				}
				if c.LimiteGastos > 0 {
					l := c.LimiteGastos
					dto.Limite = &l
				}
				if res != nil && res.Votos > 0 && dto.Despesas > 0 {
					v := dto.Despesas / float64(res.Votos)
					dto.CustoPorVoto = &v
				}
				perfil.Campanha2026[c.ID] = dto
			}
		}
	}
	return perfil, nil
}

func (d *Servico) preencherJustica(ctx context.Context, perfil *PerfilDTO, p *pessoa) {
	perfil.Justica = []sancoes.RegistroJustica{}
	s := d.sancoes.Obter(ctx)
	if s == nil {
		perfil.Avisos = append(perfil.Avisos, "Cadastros de sanções (CGU) e decisões da Justiça Eleitoral ainda a carregar.")
		return
	}
	cpf := d.cpfDe(ctx, p.cpf, p.nomeNasc)
	sqs := []string{}
	for _, c := range perfil.Candidaturas2026 {
		sqs = append(sqs, c.ID)
	}
	if p.eleito != nil {
		sqs = append(sqs, p.eleito.ID)
	}
	for _, h := range perfil.Historico {
		sqs = append(sqs, h.Sq)
	}
	perfil.Justica = s.Buscar(cpf, perfil.Pessoal.Nome, sqs...)
	if perfil.Justica == nil {
		perfil.Justica = []sancoes.RegistroJustica{}
	}
	// Corrupção primeiro, depois as sanções em vigor.
	sort.SliceStable(perfil.Justica, func(i, j int) bool {
		a, b := perfil.Justica[i], perfil.Justica[j]
		if a.Corrupcao != b.Corrupcao {
			return a.Corrupcao
		}
		return a.Inicio > b.Inicio
	})
	perfil.JusticaVerificada = cpf != ""
	perfil.JusticaDataCGU = s.DataCGU
}

func (d *Servico) preencherIdentidade(perfil *PerfilDTO, p *pessoa) {
	ps := &perfil.Pessoal
	ps.Redes = []noticias.LinkDTO{}
	if t := p.tse; t != nil {
		ps.Nome, ps.NomeUrna = t.Nome, t.NomeUrna
		ps.Nascimento, ps.Idade = t.Nascimento, idadeEm(t.Nascimento)
		ps.Genero, ps.Escolaridade = generoExtenso(t.Genero), t.Escolaridade
		ps.EstadoCivil, ps.CorRaca, ps.Ocupacao, ps.Email = t.EstadoCivil, t.CorRaca, t.Ocupacao, t.Email
		ps.Naturalidade = strings.Trim(strings.Join([]string{t.MunicipioNasc, t.UfNascimento}, " - "), " -")
		perfil.Foto, perfil.Partido, perfil.UF, perfil.Municipio = t.Foto, t.Partido, t.UF, t.Municipio
	}
	if p.eleito != nil { // quem está no cargo é mostrado pelo local do mandato
		perfil.UF, perfil.Municipio = p.eleito.UF, p.eleito.Municipio
	}
	if dep := p.deputado; dep != nil {
		det := p.detalhe
		if ps.Nome == "" {
			ps.Nome, ps.NomeUrna = det.NomeCivil, dep.Nome
			ps.Nascimento, ps.Idade = det.DataNascimento, idadeEm(det.DataNascimento)
			ps.Genero, ps.Escolaridade = generoExtenso(dominio.GeneroCurto(det.Sexo)), det.Escolaridade
		}
		if ps.Naturalidade == "" && det.MunicipioNasc != "" {
			ps.Naturalidade = det.MunicipioNasc + " - " + det.UfNascimento
		}
		ps.Email = dep.Email
		for _, r := range det.RedeSocial {
			ps.Redes = append(ps.Redes, noticias.LinkDTO{Rotulo: nomeRede(r), URL: r})
		}
		if det.UrlWebsite != "" {
			ps.Redes = append(ps.Redes, noticias.LinkDTO{Rotulo: "Site", URL: det.UrlWebsite})
		}
		perfil.Foto, perfil.Partido, perfil.UF = dep.Foto, dep.Partido, dep.UF
		perfil.NomeBusca = dep.Nome
	}
	if s := p.senador; s != nil {
		if ps.Nome == "" {
			ps.Nome, ps.NomeUrna, ps.Genero = s.NomeCompleto, s.Nome, generoExtenso(dominio.GeneroCurto(s.Sexo))
			if p.estatSenado != nil {
				ps.Nascimento, ps.Idade = p.estatSenado.DataNascimento, idadeEm(p.estatSenado.DataNascimento)
			}
		}
		ps.Email = s.Email
		perfil.Foto, perfil.Partido, perfil.UF = s.Foto, s.Partido, s.UF
		perfil.NomeBusca = s.Nome
	}
	if perfil.NomeBusca == "" {
		// O nome de urna costuma ser como a imprensa os trata; se for curto demais, usa o nome completo.
		if len(strings.Fields(ps.NomeUrna)) >= 2 {
			perfil.NomeBusca = texto.TituloMunicipio(ps.NomeUrna)
		} else {
			perfil.NomeBusca = texto.TituloMunicipio(ps.Nome)
		}
	}
}

func (d *Servico) preencherMandato(ctx context.Context, perfil *PerfilDTO, p *pessoa, deputados []DeputadoDTO) {
	switch {
	case p.deputado != nil:
		dep := p.deputado
		mand := &MandatoDTO{
			Cargo: "Deputado(a) Federal", Local: dep.UF, Periodo: mandatoPorCargo["deputado-federal"], AnoEleicao: dominio.AnoGeral2022,
			Desempenho: &DesempenhoDTO{Rotulo: "Assiduidade", Percentual: dep.Assiduidade, Projetos: dep.TotalProjetos},
			Pagina:     &noticias.LinkDTO{Rotulo: "Página na Câmara dos Deputados", URL: fmt.Sprintf("https://www.camara.leg.br/deputados/%d", dep.ID)},
		}
		if dep.Assiduidade != nil {
			mand.Desempenho.Detalhe = fmt.Sprintf("%d de %d sessões deliberativas do Plenário", *dep.SessoesPresentes, *dep.SessoesTotal)
		}
		if g := d.geral2022.Obter(ctx); g != nil {
			mand.Votacao = g.VotacaoDe(p.detalhe.CPF, dominio.ChaveNomeNasc(p.detalhe.NomeCivil, p.detalhe.DataNascimento))
		}
		if v := d.votacoesCD.Obter(ctx); v != nil {
			if vd := v.PorDeputado[dep.ID]; vd != nil {
				mand.Votacoes = resumirVotos("Câmara dos Deputados", vd)
			}
		} else {
			perfil.Avisos = append(perfil.Avisos, "Votações da Câmara ainda a carregar; tente de novo daqui a pouco.")
		}
		if c := d.cota.Obter(ctx); c != nil {
			mand.Gastos = resumirGastos(c, dep.ID, deputados)
		} else {
			perfil.Avisos = append(perfil.Avisos, "Gastos da cota parlamentar ainda a carregar; tente de novo daqui a pouco.")
		}
		if projetos, err := d.projetosLista.Obter(ctx, dep.ID); err == nil {
			mand.Projetos = projetos
		}
		mand.Custo = d.custoDeputado(ctx, dep.ID, p.detalhe)
		perfil.Mandato = mand

	case p.senador != nil:
		s := p.senador
		mand := &MandatoDTO{
			Cargo: "Senador(a)", Local: s.UF, Periodo: anoDe(s.MandatoInicio) + "–" + anoDe(s.MandatoFim), Vices: s.Suplentes,
			Pagina: &noticias.LinkDTO{Rotulo: "Página no Senado Federal", URL: fmt.Sprintf("https://www25.senado.leg.br/web/senadores/senador/-/perfil/%d", s.Codigo)},
		}
		if s.Suplente {
			mand.Situacao = "Suplente em exercício"
		} else if strings.HasPrefix(s.MandatoInicio, "2023") {
			mand.AnoEleicao = dominio.AnoGeral2022
			if g := d.geral2022.Obter(ctx); g != nil {
				mand.Votacao = g.VotacaoDe("", p.nomeNasc)
			}
		} else if strings.HasPrefix(s.MandatoInicio, "2019") {
			mand.AnoEleicao = 2018
		}
		if st := p.estatSenado; st != nil {
			mand.Desempenho = &DesempenhoDTO{Rotulo: "Participação em votações", Percentual: st.Participacao(), Projetos: &st.Projetos,
				Detalhe: fmt.Sprintf("Votou em %d de %d votações nominais (%d ausências justificadas, %d sem comparecer)", st.Presente, st.Votacoes, st.AusenciasJustificadas, st.NaoCompareceu)}
			mand.Votacoes = &VotacoesDTO{Casa: "Senado Federal", Total: st.Votacoes, Recentes: st.VotosRecentes}
			mand.Projetos = st.ProjetosRecentes
		}
		mand.Custo = d.custoSenador(ctx, *s)
		perfil.Mandato = mand

	case p.eleito != nil:
		e := p.eleito
		chave := e.ChaveCargo
		local := e.UF
		if e.Municipio != "" {
			local = e.Municipio + " - " + e.UF
		} else if local == "BR" {
			local = "Brasil"
		}
		mand := &MandatoDTO{Cargo: e.Cargo, Local: local, Periodo: mandatoPorCargo[chave], AnoEleicao: p.anoEleito, Vices: e.Vices, Situacao: e.Situacao}
		if p.anoEleito == dominio.AnoGeral2022 {
			if g := d.geral2022.Obter(ctx); g != nil {
				mand.Votacao = g.Votos.PorCandidato[e.ID]
			}
		} else if v := d.votos2024.Obter(ctx); v != nil {
			mand.Votacao = v.PorCandidato[e.ID]
		}
		tmp := &EleitoDTO{Candidatura: e, Candidaturas2026: perfil.Candidaturas2026}
		tmp.anotarSaida()
		mand.Nota = tmp.Nota
		perfil.Mandato = mand
	}
}

func (d *Servico) preencherHistoricoEPatrimonio(ctx context.Context, perfil *PerfilDTO, p *pessoa) {
	perfil.Historico = []*tse.CandidaturaPassada{}
	var sq2022 string
	if h := d.historico.Obter(ctx); h != nil {
		for _, c := range h.Buscar(p.cpf, p.nomeNasc) {
			perfil.Historico = append(perfil.Historico, c)
			if c.Ano == dominio.AnoGeral2022 && (sq2022 == "" || c.Eleito) {
				sq2022 = c.Sq
			}
		}
	} else {
		perfil.Avisos = append(perfil.Avisos, "Histórico eleitoral ainda a carregar.")
	}
	if p.eleito != nil && p.anoEleito == dominio.AnoMunicipal {
		h := &tse.CandidaturaPassada{Ano: dominio.AnoMunicipal, Cargo: p.eleito.Cargo, Local: p.eleito.Municipio + " - " + p.eleito.UF,
			Partido: p.eleito.Partido, Numero: p.eleito.Numero, Resultado: p.eleito.Situacao, Eleito: true}
		if v := d.votos2024.Obter(ctx); v != nil {
			if vt := v.PorCandidato[p.eleito.ID]; vt != nil {
				h.Votos, h.Percentual = &vt.Votos, &vt.Percentual
			}
		}
		perfil.Historico = append(perfil.Historico, h)
	}
	sort.SliceStable(perfil.Historico, func(i, j int) bool { return perfil.Historico[i].Ano > perfil.Historico[j].Ano })

	bens := d.bens.Obter(ctx)
	if bens == nil {
		return
	}
	var sq2026 string
	for _, c := range perfil.Candidaturas2026 {
		if _, ok := bens.Bens2026[c.ID]; ok {
			sq2026 = c.ID
			break
		}
	}
	if sq2026 == "" && len(perfil.Candidaturas2026) == 0 {
		return
	}
	pat := &PatrimonioDTO{Bens: bens.Bens2026[sq2026]}
	if pat.Bens == nil {
		pat.Bens = []tse.Bem{}
	}
	sort.Slice(pat.Bens, func(i, j int) bool { return pat.Bens[i].Valor > pat.Bens[j].Valor })
	for _, b := range pat.Bens {
		pat.Total2026 += b.Valor
	}
	if t, ok := bens.Total2022[sq2022]; ok && sq2022 != "" {
		pat.Total2022 = &t
		if t > 0 {
			v := (pat.Total2026 - t) / t * 100
			pat.Variacao = &v
		}
	}
	perfil.Patrimonio = pat
}

// Perfil monta o perfil completo de uma pessoa a partir do identificador usado no site:
// SQ_CANDIDATO do TSE, "camara-<id>" ou "senado-<código>".
func (d *Servico) Perfil(ctx context.Context, id string) (*PerfilDTO, error) {
	if id == "" || len(id) > 40 {
		return nil, entradaInvalida("Identificador inválido")
	}
	perfil, err := d.montarPerfil(ctx, id)
	if err != nil {
		return nil, err
	}
	if perfil == nil {
		return nil, naoEncontrado("Perfil não encontrado")
	}
	return perfil, nil
}
