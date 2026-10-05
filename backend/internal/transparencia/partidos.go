package transparencia

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/fontes/tse"
	"quemeuvoto/internal/plataforma/texto"
)

// PartidoDTO resume a presença de um partido nas eleições de 2026, na Câmara e nas prefeituras/câmaras municipais.
type PartidoDTO struct {
	Sigla     string `json:"sigla"`
	Numero    string `json:"numero"`
	Nome      string `json:"nome"`
	Federacao string `json:"federacao,omitempty"`

	// Candidaturas aptas em 2026, por cargo (presidente, governador, senador, deputado-federal, deputado-estadual).
	Candidatos2026      map[string]int `json:"candidatos2026"`
	TotalCandidatos2026 int            `json:"totalCandidatos2026"`

	DeputadosFederais int      `json:"deputadosFederais"`
	MediaAssiduidade  *float64 `json:"mediaAssiduidade"`
	TotalProjetos     int      `json:"totalProjetos"`

	Prefeitos2024  int `json:"prefeitos2024"`
	Vereadores2024 int `json:"vereadores2024"`
}

type RespostaPartidosDTO struct {
	Partidos     []*PartidoDTO `json:"partidos"`
	AnoEleicao   int           `json:"anoEleicao"`
	AnoMunicipal int           `json:"anoMunicipal"`
	AtualizadoEm time.Time     `json:"atualizadoEm"`
	Avisos       []string      `json:"avisos"`
}

// ListarPartidos resume cada partido: candidaturas de 2026, bancada na Câmara (com a média de
// assiduidade e os projetos) e prefeitos e vereadores eleitos em 2024.
func (d *Servico) ListarPartidos(ctx context.Context) (*RespostaPartidosDTO, error) {
	// Os partidos são agrupados pelo número no TSE, que se mantém quando o partido muda
	// de nome (ex.: PMB → DEMOCRATA, ambos 35). Como 2026 é lido primeiro, a sigla e o
	// nome mostrados são os atuais. A Câmara não informa o número: os deputados são
	// associados pela sigla, sem espaços nem acentos ("PCdoB" = "PC do B").
	porChave := map[string]*PartidoDTO{}
	numeroPorSigla := map[string]string{}
	chaveSigla := func(sigla string) string { return strings.ReplaceAll(texto.Normalizar(sigla), " ", "") }
	obter := func(chave, sigla string) *PartidoDTO {
		p, ok := porChave[chave]
		if !ok {
			p = &PartidoDTO{Sigla: sigla, Candidatos2026: map[string]int{}}
			porChave[chave] = p
		}
		return p
	}
	doTSE := func(c *tse.Candidatura) *PartidoDTO {
		numeroPorSigla[chaveSigla(c.Partido)] = c.NumeroPartido
		p := obter("n"+c.NumeroPartido, c.Partido)
		if p.Numero == "" {
			p.Numero, p.Nome = c.NumeroPartido, c.NomePartido
		}
		if p.Federacao == "" {
			p.Federacao = c.Federacao
		}
		return p
	}
	daCamara := func(sigla string) *PartidoDTO {
		if numero, ok := numeroPorSigla[chaveSigla(sigla)]; ok {
			return obter("n"+numero, sigla)
		}
		return obter("s"+chaveSigla(sigla), sigla)
	}
	resposta := &RespostaPartidosDTO{AnoEleicao: dominio.AnoEleicao, AnoMunicipal: dominio.AnoMunicipal, AtualizadoEm: time.Now(), Avisos: []string{}}

	if candidatos := d.candidatos.Obter(ctx); candidatos != nil {
		for _, c := range candidatos.Lista {
			p := doTSE(c)
			if c.Apto {
				p.Candidatos2026[c.ChaveCargo]++
				p.TotalCandidatos2026++
			}
		}
	} else {
		resposta.Avisos = append(resposta.Avisos, "Candidaturas de 2026 indisponíveis no momento.")
	}

	if municipal := d.municipal.Obter(ctx); municipal != nil {
		for _, c := range municipal.Prefeitos {
			doTSE(c).Prefeitos2024++
		}
		for _, c := range municipal.Vereadores {
			doTSE(c).Vereadores2024++
		}
	} else {
		resposta.Avisos = append(resposta.Avisos, "Eleitos municipais indisponíveis no momento.")
	}

	if todos, err := d.deputadosEmExercicio(ctx); err != nil {
		resposta.Avisos = append(resposta.Avisos, "Deputados federais indisponíveis no momento.")
	} else {
		montados, ok := d.montarDeputados(ctx, todos)
		if !ok {
			return nil, ctx.Err()
		}
		somaAssid := map[*PartidoDTO]float64{}
		comAssid := map[*PartidoDTO]int{}
		for _, dep := range montados.Deputados {
			p := daCamara(dep.Partido)
			p.DeputadosFederais++
			if dep.TotalProjetos != nil {
				p.TotalProjetos += *dep.TotalProjetos
			}
			if dep.Assiduidade != nil {
				somaAssid[p] += *dep.Assiduidade
				comAssid[p]++
			}
		}
		for p, n := range comAssid {
			media := math.Round(somaAssid[p]/float64(n)*10) / 10
			p.MediaAssiduidade = &media
		}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	for _, p := range porChave {
		resposta.Partidos = append(resposta.Partidos, p)
	}
	sort.Slice(resposta.Partidos, func(i, j int) bool {
		a, b := resposta.Partidos[i], resposta.Partidos[j]
		if a.TotalCandidatos2026 != b.TotalCandidatos2026 {
			return a.TotalCandidatos2026 > b.TotalCandidatos2026
		}
		return a.Sigla < b.Sigla
	})
	return resposta, nil
}
