package transparencia

import (
	"context"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/fontes/sancoes"
	"quemeuvoto/internal/fontes/tse"
)

// CandidatoDTO é um candidato do TSE; se for deputado federal em exercício, leva o seu desempenho.
type CandidatoDTO struct {
	*tse.Candidatura
	DeputadoFederal *DeputadoAtualDTO       `json:"deputadoFederal"`
	Justica         ResumoJusticaDTO        `json:"justica"`
	Patrimonio      *float64                `json:"patrimonio"` // bens declarados ao TSE em 2026
	Apuracao        *tse.ResultadoCandidato `json:"apuracao"`   // votos na apuração de 2026 (em tempo real)
	Campanha        *tse.ResumoCampanhaDTO  `json:"campanha"`   // gastos declarados ao TSE
}

type DeputadoAtualDTO struct {
	ID   int    `json:"id"`
	Nome string `json:"nome"`
	MetricasDTO
}

type RespostaCandidatosDTO struct {
	Candidatos              []CandidatoDTO `json:"candidatos"`
	Total                   int            `json:"total"`
	Pagina                  int            `json:"pagina"`
	TotalPaginas            int            `json:"totalPaginas"`
	AnoEleicao              int            `json:"anoEleicao"`
	InicioPeriodo           string         `json:"inicioPeriodo"`
	CandidatosAtualizadosEm *time.Time     `json:"candidatosAtualizadosEm"`
	Avisos                  []string       `json:"avisos"`
	Partidos                []string       `json:"partidos"` // opções para o filtro
}

const porPagina = 48

func cargoValido(chave string) bool {
	for _, c := range dominio.CargosPrincipais {
		if c == chave {
			return true
		}
	}
	return false
}

// ListarCandidatos devolve uma página das candidaturas de 2026 com o que se sabe de cada uma:
// registos na Justiça, patrimônio, apuração, contas de campanha e, para quem já é deputado
// federal, o desempenho na Câmara. Se a Câmara estiver lenta ou fora do ar, a lista sai na
// mesma, sem essas métricas e com um aviso.
func (d *Servico) ListarCandidatos(ctx context.Context, f Filtro) (*RespostaCandidatosDTO, error) {
	if err := f.normalizar(); err != nil {
		return nil, err
	}
	if !cargoValido(f.Cargo) {
		return nil, entradaInvalida("Cargo inválido")
	}
	if f.Cargo == "presidente" {
		f.Uf = "TODOS"
	}
	f.Municipio = "TODOS"

	indice := d.candidatos.Obter(ctx)
	if indice == nil {
		return nil, indisponivel("Dados de candidaturas do TSE indisponíveis no momento")
	}
	cadastros := d.sancoes.Obter(ctx)
	registros := func(c *tse.Candidatura) []sancoes.RegistroJustica {
		if cadastros == nil {
			return nil
		}
		return cadastros.Buscar(c.Cpf, c.Nome, c.ID)
	}
	var aceitar func(*tse.Candidatura) bool
	if f.SoJustica {
		aceitar = func(c *tse.Candidatura) bool { return len(registros(c)) > 0 }
	}
	var patrimonio map[string]float64
	if bens := d.bens.Obter(ctx); bens != nil {
		patrimonio = bens.Total2026
	}
	// Apuração de 2026: busca as UFs do cargo (ou só a escolhida) para mostrar e ordenar por votos.
	var ufsApuracao []string
	if f.Uf != "TODOS" {
		ufsApuracao = []string{f.Uf}
	} else {
		for uf := range dominio.UFsValidas {
			ufsApuracao = append(ufsApuracao, uf)
		}
	}
	apuracao := d.apuracaoDe(ctx, f.Cargo, ufsApuracao)
	votos := make(map[string]float64, len(apuracao))
	for sq, res := range apuracao {
		votos[sq] = float64(res.Votos)
	}
	campanha := d.campanha.Obter(ctx)
	gastos := campanha.DespesasPorSQ()
	custoVoto := map[string]float64{}
	for sq, g := range gastos {
		if v := votos[sq]; v > 0 && g > 0 {
			custoVoto[sq] = g / v
		}
	}
	metricas := map[string]map[string]float64{
		"patrimonio": patrimonio, "apuracao": votos, "campanha": gastos, "custo-voto": custoVoto,
	}
	pag, total := filtrarEPaginar(indice.Lista, f, aceitar, metricas)

	resposta := &RespostaCandidatosDTO{
		Total:                   total,
		Pagina:                  f.Pagina,
		TotalPaginas:            totalPaginas(total),
		AnoEleicao:              dominio.AnoEleicao,
		InicioPeriodo:           dominio.InicioLegislatura,
		CandidatosAtualizadosEm: &indice.AtualizadoEm,
		Candidatos:              []CandidatoDTO{},
		Avisos:                  []string{},
	}
	if cadastros == nil {
		resposta.Avisos = append(resposta.Avisos, "Cadastros de sanções (CGU) e decisões da Justiça Eleitoral indisponíveis no momento.")
	}
	if campanha == nil {
		resposta.Avisos = append(resposta.Avisos, "Contas de campanha do TSE indisponíveis no momento.")
	}
	var doCargo []*tse.Candidatura
	for _, c := range indice.Lista {
		if c.ChaveCargo == f.Cargo && (f.Uf == "TODOS" || c.UF == f.Uf) && (c.Apto || f.IncluirInaptos) {
			doCargo = append(doCargo, c)
		}
	}
	resposta.Partidos = partidosDe(doCargo)

	deputadoPorCandidatura, aviso := d.deputadosPorCandidatura(ctx)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if aviso != "" {
		resposta.Avisos = append(resposta.Avisos, aviso)
	}
	for _, c := range pag {
		dto := CandidatoDTO{
			Candidatura: c, DeputadoFederal: deputadoPorCandidatura[c.ID],
			Justica: resumoJustica(registros(c), len(c.Cpf) == 11),
		}
		if v, ok := patrimonio[c.ID]; ok {
			dto.Patrimonio = &v
		}
		dto.Apuracao = apuracao[c.ID]
		dto.Campanha = campanha.Resumo(c)
		resposta.Candidatos = append(resposta.Candidatos, dto)
	}
	return resposta, nil
}

// deputadosPorCandidatura liga cada candidatura de 2026 ao deputado federal em exercício que a
// fez. Espera pela Câmara no máximo 8 s; sem resposta, devolve um aviso para mostrar na página.
func (d *Servico) deputadosPorCandidatura(ctx context.Context) (map[string]*DeputadoAtualDTO, string) {
	res := map[string]*DeputadoAtualDTO{}
	ctxCamara, cancelar := context.WithTimeout(ctx, 8*time.Second)
	defer cancelar()
	todos, err := d.deputadosEmExercicio(ctxCamara)
	if err != nil {
		return res, "Não foi possível consultar os deputados federais em exercício."
	}
	montados, ok := d.montarDeputados(ctxCamara, todos)
	if !ok {
		return res, "A Câmara dos Deputados não respondeu a tempo: as métricas de quem já é deputado não aparecem agora."
	}
	for _, dep := range montados.Deputados {
		for _, c := range dep.Candidaturas {
			res[c.ID] = &DeputadoAtualDTO{ID: dep.ID, Nome: dep.Nome, MetricasDTO: dep.MetricasDTO}
		}
	}
	return res, ""
}
