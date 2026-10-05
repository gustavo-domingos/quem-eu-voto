package transparencia

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/fontes/camara"
	"quemeuvoto/internal/fontes/tse"
	"quemeuvoto/internal/plataforma/cache"
)

// projetosDe devolve o total de projetos por ID; deputados cuja contagem falhou ficam de fora do mapa.
func (d *Servico) projetosDe(ctx context.Context, deps []camara.DeputadoResumo) map[int]int {
	return d.projetos.ObterVarios(ctx, idsDe(deps))
}

func (d *Servico) detalhesDe(ctx context.Context, deps []camara.DeputadoResumo) map[int]camara.DetalheDeputado {
	return d.detalhes.ObterVarios(ctx, idsDe(deps))
}

func (d *Servico) deputadosEmExercicio(ctx context.Context) ([]camara.DeputadoResumo, error) {
	for {
		d.muDeputados.Lock()
		if d.deputados != nil && time.Since(d.deputadosEm) < ttlDeputados {
			deps := d.deputados
			d.muDeputados.Unlock()
			return deps, nil
		}
		// Ao arrancar, usa a lista guardada em disco se for recente (poupa a API da Câmara).
		if d.deputados == nil {
			if salvos, ok := cache.Ler[[]camara.DeputadoResumo]("camara_deputados.json", ttlDeputados); ok {
				d.deputados, d.deputadosEm = *salvos, time.Now()
				d.muDeputados.Unlock()
				return *salvos, nil
			}
		}
		antigos := d.deputados
		if antigos == nil && time.Since(d.falhaDeputados) < time.Minute {
			err := d.erroDeputados
			d.muDeputados.Unlock()
			return nil, err
		}
		busca := d.buscaDeputados
		if busca == nil {
			busca = make(chan struct{})
			d.buscaDeputados = busca
			go d.buscarDeputados(busca)
		}
		d.muDeputados.Unlock()

		// Espera pela busca (partilhada por todos os pedidos) só até ao prazo de quem pediu.
		select {
		case <-busca:
			continue
		case <-ctx.Done():
			if antigos != nil {
				return antigos, nil
			}
			return nil, ctx.Err()
		}
	}
}

func (d *Servico) buscarDeputados(feito chan struct{}) {
	ctx, cancelar := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancelar()
	deps, err := camara.ListarDeputados(ctx)

	d.muDeputados.Lock()
	defer d.muDeputados.Unlock()
	defer close(feito)
	d.buscaDeputados = nil
	if err == nil {
		d.deputados, d.deputadosEm = deps, time.Now()
		if err := cache.Gravar("camara_deputados.json", deps); err != nil {
			log.Printf("deputados: não foi possível gravar a cache: %v", err)
		}
		return
	}
	if d.deputados != nil {
		log.Printf("deputados: a usar a lista anterior após erro: %v", err)
		d.deputadosEm = time.Now() // volta a tentar só depois do TTL
		return
	}
	// Com a Câmara fora do ar, aceita uma lista em disco de até uma semana.
	if salvos, ok := cache.Ler[[]camara.DeputadoResumo]("camara_deputados.json", 7*24*time.Hour); ok {
		log.Printf("deputados: Câmara indisponível (%v); a usar a lista guardada em disco", err)
		d.deputados, d.deputadosEm = *salvos, time.Now()
		return
	}
	log.Printf("deputados: Câmara indisponível: %v", err)
	d.falhaDeputados, d.erroDeputados = time.Now(), err
}

func idsDe(deps []camara.DeputadoResumo) []int {
	ids := make([]int, len(deps))
	for i, d := range deps {
		ids[i] = d.ID
	}
	return ids
}

// MetricasDTO reúne o desempenho de um deputado federal. Métricas indisponíveis vão como null.
type MetricasDTO struct {
	Assiduidade      *float64 `json:"assiduidade"`
	SessoesPresentes *int     `json:"sessoesPresentes"`
	SessoesTotal     *int     `json:"sessoesTotal"`
	TotalProjetos    *int     `json:"totalProjetos"`
}

type DeputadoDTO struct {
	ID      int    `json:"id"`
	Nome    string `json:"nome"`
	Partido string `json:"partido"`
	UF      string `json:"uf"`
	Foto    string `json:"foto"`
	Email   string `json:"email"`
	MetricasDTO
	// Candidaturas nas eleições de anoEleicao: [] se não é candidato, null se não foi possível verificar.
	Candidaturas []*tse.Candidatura `json:"candidaturas"`
}

type RespostaDeputadosDTO struct {
	Deputados               []DeputadoDTO `json:"deputados"`
	InicioPeriodo           string        `json:"inicioPeriodo"`
	AnoEleicao              int           `json:"anoEleicao"`
	PresencaAtualizadaEm    *time.Time    `json:"presencaAtualizadaEm"`
	CandidatosAtualizadosEm *time.Time    `json:"candidatosAtualizadosEm"`
	Avisos                  []string      `json:"avisos"`
}

// montarDeputados junta lista, métricas e candidaturas dos deputados indicados.
// Devolve ok=false se o pedido foi cancelado entretanto.
func (dados *Servico) montarDeputados(ctx context.Context, deps []camara.DeputadoResumo) (RespostaDeputadosDTO, bool) {
	var (
		wg       sync.WaitGroup
		projetos map[int]int
		detalhes map[int]camara.DetalheDeputado
	)
	wg.Add(2)
	go func() { defer wg.Done(); projetos = dados.projetosDe(ctx, deps) }()
	go func() { defer wg.Done(); detalhes = dados.detalhesDe(ctx, deps) }()
	wg.Wait()
	presenca := dados.presenca.Obter(ctx)
	candidatos := dados.candidatos.Obter(ctx)
	if ctx.Err() != nil {
		return RespostaDeputadosDTO{}, false
	}

	resposta := RespostaDeputadosDTO{
		Deputados:     make([]DeputadoDTO, 0, len(deps)),
		InicioPeriodo: dominio.InicioLegislatura,
		AnoEleicao:    dominio.AnoEleicao,
		Avisos:        []string{},
	}
	if presenca == nil {
		resposta.Avisos = append(resposta.Avisos, "Dados de presença indisponíveis no momento.")
	} else {
		resposta.PresencaAtualizadaEm = &presenca.AtualizadoEm
	}
	if candidatos == nil {
		resposta.Avisos = append(resposta.Avisos, "Dados de candidaturas do TSE indisponíveis no momento.")
	} else {
		resposta.CandidatosAtualizadosEm = &candidatos.AtualizadoEm
		if falhas := len(deps) - len(detalhes); falhas > 0 {
			resposta.Avisos = append(resposta.Avisos, fmt.Sprintf("Não foi possível verificar a candidatura de %d deputado(s).", falhas))
		}
	}
	if falhas := len(deps) - len(projetos); falhas > 0 {
		resposta.Avisos = append(resposta.Avisos, fmt.Sprintf("Não foi possível obter a contagem de projetos de %d deputado(s).", falhas))
	}

	for _, d := range deps {
		dto := DeputadoDTO{
			ID:      d.ID,
			Nome:    d.Nome,
			Partido: d.SiglaPartido,
			UF:      d.SiglaUf,
			Foto:    d.UrlFoto,
			Email:   d.Email,
		}
		if total, ok := projetos[d.ID]; ok {
			dto.TotalProjetos = &total
		}
		if det, ok := detalhes[d.ID]; ok && candidatos != nil {
			dto.Candidaturas = candidatos.BuscarPor(det.CPF, dominio.ChaveNomeNasc(det.NomeCivil, det.DataNascimento))
			if dto.Candidaturas == nil {
				dto.Candidaturas = []*tse.Candidatura{}
			}
		}
		if presenca != nil {
			if p, ok := presenca.PorDeputado[d.ID]; ok && p.Sessoes > 0 {
				pct := math.Round(p.Percentual()*10) / 10
				dto.Assiduidade = &pct
				dto.SessoesPresentes = &p.Presentes
				dto.SessoesTotal = &p.Sessoes
			}
		}
		resposta.Deputados = append(resposta.Deputados, dto)
	}
	return resposta, true
}

// ListarDeputados devolve os deputados federais em exercício (todos ou de uma UF) com as suas métricas.
func (d *Servico) ListarDeputados(ctx context.Context, uf string) (*RespostaDeputadosDTO, error) {
	uf = strings.ToUpper(uf)
	if uf == "" {
		uf = "TODOS"
	}
	if uf != "TODOS" && !dominio.UFsValidas[uf] {
		return nil, entradaInvalida("UF inválida")
	}
	todos, err := d.deputadosEmExercicio(ctx)
	if err != nil {
		log.Printf("listar deputados: %v", err)
		return nil, indisponivel("Erro ao consultar API da Câmara")
	}
	deps := make([]camara.DeputadoResumo, 0, len(todos))
	for _, dep := range todos {
		if uf == "TODOS" || dep.SiglaUf == uf {
			deps = append(deps, dep)
		}
	}
	resposta, ok := d.montarDeputados(ctx, deps)
	if !ok {
		return nil, ctx.Err()
	}
	return &resposta, nil
}
