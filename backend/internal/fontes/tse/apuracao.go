package tse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/carga"
	"quemeuvoto/internal/plataforma/rede"
	"quemeuvoto/internal/plataforma/texto"
)

// Apuração em tempo real, a partir dos ficheiros que o TSE publica em resultados.tse.jus.br
// (os mesmos que alimentam o site e a app "Resultados" do TSE).
const (
	urlResultadosTSE = "https://resultados.tse.jus.br/oficial/ele%d/%d/dados/%s/%s-c%04d-e%06d-u.json"
	eleicaoFederal   = 6257 // Presidente
	eleicaoEstadual  = 6259 // Governador, Senador, Deputados
	ttlApuracao      = 60 * time.Second
)

// Código do cargo nos ficheiros de resultados do TSE.
var CargoResultados = map[string]int{
	"presidente": 1, "governador": 3, "senador": 5, "deputado-federal": 6, "deputado-estadual": 7,
}

// ResultadoCandidato é a situação de um candidato na apuração.
type ResultadoCandidato struct {
	Votos      int64   `json:"votos"`
	Percentual float64 `json:"percentual"` // dos votos válidos
	Posicao    int     `json:"posicao"`
	Situacao   string  `json:"situacao,omitempty"` // "Eleito", "2º turno", "Suplente"… (vazio até haver definição)
	Eleito     bool    `json:"eleito"`
	Destino    string  `json:"destino,omitempty"` // "Válido", "Anulado"… (votos de candidaturas sub judice)
}

type LinhaApuracao struct {
	SQ       string         `json:"sq"`
	Numero   string         `json:"numero"`
	NomeUrna string         `json:"nomeUrna"`
	Partido  string         `json:"partido"`
	Vices    []dominio.Vice `json:"vices,omitempty"`
	ResultadoCandidato
}

// Apuracao é o estado da contagem de um cargo numa UF (ou no país, para Presidente).
type Apuracao struct {
	Cargo          string          `json:"cargo"`
	Abrangencia    string          `json:"abrangencia"` // "BR" ou a UF
	Turno          string          `json:"turno"`
	AtualizadoEm   string          `json:"atualizadoEm"`   // data e hora do TSE
	SecoesApuradas float64         `json:"secoesApuradas"` // % de seções totalizadas
	Totalizada     bool            `json:"totalizada"`     // contagem encerrada
	Eleitores      int64           `json:"eleitores"`
	Comparecimento float64         `json:"comparecimento"` // % dos eleitores das seções apuradas
	Abstencao      float64         `json:"abstencao"`
	VotosValidos   int64           `json:"votosValidos"`
	Brancos        float64         `json:"brancos"` // % dos votos
	Nulos          float64         `json:"nulos"`
	Candidatos     []LinhaApuracao `json:"candidatos"`

	porSQ map[string]*LinhaApuracao
}

func (a *Apuracao) De(sq string) *ResultadoCandidato {
	if a == nil {
		return nil
	}
	if l, ok := a.porSQ[sq]; ok {
		return &l.ResultadoCandidato
	}
	return nil
}

type Apuracoes struct {
	mu       sync.Mutex
	entradas map[string]*entradaApuracao
}

type entradaApuracao struct {
	pronto chan struct{}
	valor  *Apuracao
	err    error
	expira time.Time
}

func NovasApuracoes() *Apuracoes {
	return &Apuracoes{entradas: map[string]*entradaApuracao{}}
}

// Obter devolve a apuração (em cache por 60 s). uf é ignorada para Presidente.
func (a *Apuracoes) Obter(ctx context.Context, chaveCargo, uf string) (*Apuracao, error) {
	cod, ok := CargoResultados[chaveCargo]
	if !ok {
		return nil, fmt.Errorf("cargo sem apuração: %s", chaveCargo)
	}
	eleicao, abr, abrangencia := eleicaoEstadual, strings.ToLower(uf), uf
	if chaveCargo == "presidente" {
		eleicao, abr, abrangencia = eleicaoFederal, "br", "BR"
	} else if uf == "DF" && chaveCargo == "deputado-estadual" {
		cod = 8 // Deputado Distrital
	}
	chave := fmt.Sprintf("%d/%s/%d", eleicao, abr, cod)

	a.mu.Lock()
	e, existe := a.entradas[chave]
	if !existe || (carga.Fechado(e.pronto) && time.Now().After(e.expira)) {
		anterior := e
		e = &entradaApuracao{pronto: make(chan struct{})}
		a.entradas[chave] = e
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			url := fmt.Sprintf(urlResultadosTSE, dominio.AnoEleicao, eleicao, abr, abr, cod, eleicao)
			e.valor, e.err = baixarApuracao(ctx, url, chaveCargo, abrangencia)
			e.expira = time.Now().Add(ttlApuracao)
			if e.err != nil {
				log.Printf("apuração %s: %v", chave, e.err)
				// Se o TSE falhar, mantém a última apuração conhecida em vez de não mostrar nada.
				if anterior != nil && anterior.valor != nil {
					e.valor, e.err = anterior.valor, nil
				}
				e.expira = time.Now().Add(15 * time.Second)
			}
			close(e.pronto)
		}()
	}
	a.mu.Unlock()
	select {
	case <-e.pronto:
		return e.valor, e.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Os números do TSE vêm como texto, com vírgula decimal ("64,81").
func numeroTSE(s string) float64 {
	v, _ := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	return v
}

func inteiroTSE(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func baixarApuracao(ctx context.Context, url, chaveCargo, abrangencia string) (*Apuracao, error) {
	resp, err := rede.Baixar(ctx, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	corpo, err := io.ReadAll(io.LimitReader(resp.Body, 50<<20))
	if err != nil {
		return nil, err
	}
	// Os ficheiros de dados vêm em Latin-1; os de configuração em UTF-8.
	if !utf8.Valid(corpo) {
		var b strings.Builder
		for _, c := range corpo {
			b.WriteRune(rune(c))
		}
		corpo = []byte(b.String())
	}

	var r struct {
		T   string                         `json:"t"`
		Tf  string                         `json:"tf"`
		Dg  string                         `json:"dg"`
		Hg  string                         `json:"hg"`
		S   struct{ Pst string }           `json:"s"`
		E   struct{ Te, Pc, Pa string }    `json:"e"`
		V   struct{ Vv, Pvb, Ptvn string } `json:"v"`
		Car []struct {
			Agr []struct {
				Par []struct {
					Sg   string `json:"sg"`
					Cand []struct {
						N     string `json:"n"`
						SQ    string `json:"sqcand"`
						Nmu   string `json:"nmu"`
						Dvt   string `json:"dvt"`
						Seq   string `json:"seq"`
						E     string `json:"e"`
						St    string `json:"st"`
						Vap   string `json:"vap"`
						Pvapn string `json:"pvapn"`
						Vices []struct {
							Tp, Nmu, Sgp string
						} `json:"vs"`
					} `json:"cand"`
				} `json:"par"`
			} `json:"agr"`
		} `json:"carg"`
	}
	if err := json.Unmarshal(corpo, &r); err != nil {
		return nil, fmt.Errorf("resposta inesperada do TSE: %w", err)
	}

	a := &Apuracao{
		Cargo: chaveCargo, Abrangencia: abrangencia, Turno: r.T, AtualizadoEm: r.Dg + " " + r.Hg, SecoesApuradas: numeroTSE(r.S.Pst),
		Totalizada: r.Tf == "s", Eleitores: inteiroTSE(r.E.Te), Comparecimento: numeroTSE(r.E.Pc), Abstencao: numeroTSE(r.E.Pa),
		VotosValidos: inteiroTSE(r.V.Vv), Brancos: numeroTSE(r.V.Pvb), Nulos: numeroTSE(r.V.Ptvn),
		Candidatos: []LinhaApuracao{}, porSQ: map[string]*LinhaApuracao{},
	}
	for _, carg := range r.Car {
		for _, agr := range carg.Agr {
			for _, par := range agr.Par {
				for _, c := range par.Cand {
					l := LinhaApuracao{SQ: c.SQ, Numero: c.N, NomeUrna: c.Nmu, Partido: par.Sg}
					l.Votos, l.Percentual = inteiroTSE(c.Vap), numeroTSE(c.Pvapn)
					l.Posicao, _ = strconv.Atoi(c.Seq)
					l.Eleito, l.Situacao = c.E == "s", texto.FraseCapitalizada(c.St)
					if c.Dvt != "" && !strings.EqualFold(c.Dvt, "Válido") {
						l.Destino = c.Dvt
					}
					for _, v := range c.Vices {
						cargo := "Vice"
						if chaveCargo == "senador" {
							cargo = "Suplente"
						}
						l.Vices = append(l.Vices, dominio.Vice{Cargo: cargo, NomeUrna: v.Nmu, Partido: v.Sgp})
					}
					a.Candidatos = append(a.Candidatos, l)
				}
			}
		}
	}
	sort.SliceStable(a.Candidatos, func(i, j int) bool {
		pi, pj := a.Candidatos[i].Posicao, a.Candidatos[j].Posicao
		if (pi == 0) != (pj == 0) {
			return pj == 0
		}
		if pi != pj {
			return pi < pj
		}
		return a.Candidatos[i].Votos > a.Candidatos[j].Votos
	})
	for i := range a.Candidatos {
		a.porSQ[a.Candidatos[i].SQ] = &a.Candidatos[i]
	}
	return a, nil
}
