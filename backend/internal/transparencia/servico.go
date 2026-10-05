package transparencia

import (
	"context"
	"log"
	"sync"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/fontes/camara"
	"quemeuvoto/internal/fontes/sancoes"
	"quemeuvoto/internal/fontes/senado"
	"quemeuvoto/internal/fontes/tse"
	"quemeuvoto/internal/plataforma/carga"
)

const (
	ttlDeputados      = 6 * time.Hour
	intervaloPresenca = 24 * time.Hour
	// O TSE atualiza o ficheiro de candidatos várias vezes por dia durante o período eleitoral.
	intervaloCandidatos = 6 * time.Hour
	maxConcorrencia     = 8 // pedidos simultâneos à API da Câmara
)

// Servico guarda em memória o que vem da Câmara e do TSE, para não repetir
// centenas de chamadas a cada pedido do frontend.
type Servico struct {
	muDeputados    sync.Mutex
	deputados      []camara.DeputadoResumo
	deputadosEm    time.Time
	buscaDeputados chan struct{} // não nil enquanto há uma busca à Câmara em curso
	falhaDeputados time.Time     // última falha sem dados, para não insistir sem pausa
	erroDeputados  error

	presenca   *carga.Periodico[camara.IndicePresenca]
	candidatos *carga.Periodico[tse.IndiceCandidatos]
	municipal  *carga.Periodico[tse.IndiceMunicipal]
	geral2022  *carga.Periodico[tse.IndiceGeral2022]
	votos2024  *carga.Periodico[dominio.ResultadoVotacao]
	senadores  *carga.Periodico[[]senado.Senador]
	historico  *carga.Periodico[tse.IndiceHistorico]
	bens       *carga.Periodico[tse.IndiceBens]
	votacoesCD *carga.Periodico[camara.VotacoesCamara]
	cota       *carga.Periodico[camara.IndiceCota]
	sancoes    *carga.Periodico[sancoes.IndiceSancoes]
	campanha   *carga.Periodico[tse.IndiceCampanha]
	ceaps      *carga.Periodico[senado.IndiceCEAPS]
	gabinetes  *carga.Periodico[camara.IndiceGabinetes]
	vagas      *carga.Periodico[tse.IndiceVagas]
	projetos   *carga.PorID[int]
	detalhes   *carga.PorID[camara.DetalheDeputado]

	estatSenado   *carga.PorID[senado.EstatSenado]
	apuracoes     *tse.Apuracoes
	projetosLista *carga.PorID[[]dominio.Projeto]
}

func Novo() *Servico {
	sem := make(chan struct{}, maxConcorrencia)
	return &Servico{
		presenca:   carga.NovoPeriodico("presença", intervaloPresenca, camara.CarregarPresenca),
		candidatos: carga.NovoPeriodico("candidatos TSE", intervaloCandidatos, tse.CarregarCandidatos),
		municipal:  carga.NovoPeriodico("eleitos municipais", tse.IntervaloMunicipal, tse.CarregarMunicipal),
		geral2022:  carga.NovoPeriodico("eleitos 2022", 24*time.Hour, tse.CarregarGeral2022),
		votos2024:  carga.NovoPeriodico("votos 2024", 24*time.Hour, tse.CarregarVotos2024),
		senadores: carga.NovoPeriodico("senadores", 6*time.Hour, func(ctx context.Context) (*[]senado.Senador, error) {
			s, err := senado.ListarSenadores(ctx)
			return &s, err
		}),
		estatSenado:   carga.NovoPorID("estatísticas do senador", sem, senado.EstatisticasSenador),
		apuracoes:     tse.NovasApuracoes(),
		historico:     carga.NovoPeriodico("histórico eleitoral", 24*time.Hour, tse.CarregarHistorico),
		bens:          carga.NovoPeriodico("bens declarados", 12*time.Hour, tse.CarregarBens),
		votacoesCD:    carga.NovoPeriodico("votações da Câmara", 12*time.Hour, camara.CarregarVotacoesCamara),
		cota:          carga.NovoPeriodico("cota parlamentar", 12*time.Hour, camara.CarregarCota),
		sancoes:       carga.NovoPeriodico("sanções e decisões judiciais", sancoes.IntervaloSancoes, sancoes.CarregarSancoes),
		campanha:      carga.NovoPeriodico("contas de campanha", 6*time.Hour, tse.CarregarCampanha),
		ceaps:         carga.NovoPeriodico("cota do Senado", 12*time.Hour, senado.CarregarCEAPS),
		gabinetes:     carga.NovoPeriodico("funcionários de gabinete", 24*time.Hour, camara.CarregarGabinetes),
		vagas:         carga.NovoPeriodico("vagas por cargo", 24*time.Hour, tse.CarregarVagas),
		projetosLista: carga.NovoPorID("projetos do deputado", sem, camara.ListarProjetos),
		projetos:      carga.NovoPorID("projetos", sem, camara.ContarProjetos),
		detalhes:      carga.NovoPorID("detalhes", sem, camara.DetalharDeputado),
	}
}

// Iniciar carrega presença, candidatos e eleitos municipais e pré-aquece os dados de todos os deputados.
func (d *Servico) Iniciar() {
	go d.presenca.Executar()
	go d.candidatos.Executar()
	go d.municipal.Executar()
	go d.geral2022.Executar()
	go d.votos2024.Executar()
	go d.senadores.Executar()
	go d.historico.Executar()
	go d.bens.Executar()
	go d.votacoesCD.Executar()
	go d.cota.Executar()
	go d.sancoes.Executar()
	go d.campanha.Executar()
	go d.ceaps.Executar()
	go d.gabinetes.Executar()
	go d.vagas.Executar()
	go func() {
		if s := d.senadores.Obter(context.Background()); s != nil {
			ids := make([]int, len(*s))
			for i, sen := range *s {
				ids[i] = sen.Codigo
			}
			d.estatSenado.ObterVarios(context.Background(), ids)
			log.Printf("aquecimento: estatísticas de %d senadores", len(ids))
		}
	}()
	go func() {
		ctx := context.Background()
		deps, err := d.deputadosEmExercicio(ctx)
		if err != nil {
			log.Printf("aquecimento: falha ao listar deputados: %v", err)
			return
		}
		inicio := time.Now()
		// O que foi guardado no último arranque evita repetir ~1.000 pedidos à Câmara.
		d.projetos.CarregarDisco("camara_projetos.json", carga.TTLPorID)
		d.detalhes.CarregarDisco("camara_detalhes.json", carga.TTLPorID)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); d.projetos.ObterVarios(ctx, idsDe(deps)) }()
		go func() { defer wg.Done(); d.detalhes.ObterVarios(ctx, idsDe(deps)) }()
		wg.Wait()
		d.projetos.SalvarDisco("camara_projetos.json")
		d.detalhes.SalvarDisco("camara_detalhes.json")
		log.Printf("aquecimento: %d deputados em %s", len(deps), time.Since(inicio).Round(time.Second))
	}()
}
