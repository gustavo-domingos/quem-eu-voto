package noticias

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"quemeuvoto/internal/plataforma/rede"
	"quemeuvoto/internal/plataforma/texto"
)

// A API do GDELT (base aberta de notícias) pede no máximo 1 pedido a cada 5 segundos.
const (
	urlGDELT        = "https://api.gdeltproject.org/api/v2/doc/doc"
	intervaloGDELT  = 6 * time.Second
	ttlNoticias     = 6 * time.Hour
	ttlNoticiasErro = 10 * time.Minute
)

type Noticia struct {
	Titulo string `json:"titulo"`
	URL    string `json:"url"`
	Fonte  string `json:"fonte"`
	Data   string `json:"data"` // AAAA-MM-DD
}

type LinkDTO struct {
	Rotulo string `json:"rotulo"`
	URL    string `json:"url"`
}

type RespostaNoticiasDTO struct {
	Noticias []Noticia `json:"noticias"`
	Pesquisa []LinkDTO `json:"pesquisa"` // pesquisas que o próprio utilizador pode abrir
	Aviso    string    `json:"aviso,omitempty"`
}

type Noticias struct {
	mu       sync.Mutex
	ultimo   time.Time // último pedido feito ao GDELT
	fila     sync.Mutex
	entradas map[string]*entradaNoticias
}

type entradaNoticias struct {
	noticias []Noticia
	err      error
	expira   time.Time
}

func Novo() *Noticias {
	return &Noticias{entradas: map[string]*entradaNoticias{}}
}

func LinksDePesquisa(nome string) []LinkDTO {
	q := url.QueryEscape(`"` + nome + `"`)
	return []LinkDTO{
		{"Google Notícias", "https://news.google.com/search?hl=pt-BR&gl=BR&ceid=BR:pt-419&q=" + q},
		{"Processos públicos (Jusbrasil)", "https://www.jusbrasil.com.br/busca?q=" + q},
		{"Portal da Transparência", "https://portaldatransparencia.gov.br/busca?termo=" + url.QueryEscape(nome)},
	}
}

func (n *Noticias) Buscar(ctx context.Context, nome string) ([]Noticia, error) {
	chave := texto.Normalizar(nome)
	n.mu.Lock()
	if e, ok := n.entradas[chave]; ok && time.Now().Before(e.expira) {
		n.mu.Unlock()
		return e.noticias, e.err
	}
	n.mu.Unlock()

	// Um pedido de cada vez, respeitando o intervalo pedido pelo GDELT.
	n.fila.Lock()
	defer n.fila.Unlock()
	if espera := intervaloGDELT - time.Since(n.ultimo); espera > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(espera):
		}
	}
	n.ultimo = time.Now()

	q := url.Values{}
	q.Set("query", fmt.Sprintf(`"%s" sourcecountry:brazil`, nome))
	q.Set("mode", "artlist")
	q.Set("format", "json")
	q.Set("maxrecords", "15")
	q.Set("sort", "datedesc")
	q.Set("timespan", "12months")

	noticias, err := pedirGDELT(ctx, urlGDELT+"?"+q.Encode())
	e := &entradaNoticias{noticias: noticias, err: err, expira: time.Now().Add(ttlNoticias)}
	if err != nil {
		e.expira = time.Now().Add(ttlNoticiasErro)
	}
	n.mu.Lock()
	n.entradas[chave] = e
	n.mu.Unlock()
	return noticias, err
}

func pedirGDELT(ctx context.Context, u string) ([]Noticia, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := rede.ClienteAPI.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, rede.ErroStatus{Status: resp.StatusCode}
	}
	var res struct {
		Articles []struct {
			URL      string `json:"url"`
			Title    string `json:"title"`
			SeenDate string `json:"seendate"` // 20261004T120000Z
			Domain   string `json:"domain"`
		} `json:"articles"`
	}
	// Sem resultados, o GDELT devolve um corpo vazio em vez de JSON.
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("resposta inesperada do GDELT: %w", err)
	}
	noticias := []Noticia{}
	vistos := map[string]bool{}
	for _, a := range res.Articles {
		titulo := strings.TrimSpace(a.Title)
		if titulo == "" || vistos[strings.ToLower(titulo)] {
			continue
		}
		vistos[strings.ToLower(titulo)] = true
		data := ""
		if len(a.SeenDate) >= 8 {
			data = a.SeenDate[:4] + "-" + a.SeenDate[4:6] + "-" + a.SeenDate[6:8]
		}
		noticias = append(noticias, Noticia{Titulo: titulo, URL: a.URL, Fonte: a.Domain, Data: data})
	}
	return noticias, nil
}
