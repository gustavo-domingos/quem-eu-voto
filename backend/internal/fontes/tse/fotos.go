package tse

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"

	"quemeuvoto/internal/plataforma/carga"
	"quemeuvoto/internal/plataforma/zipremoto"
)

const (
	urlFotosTSE = "https://cdn.tse.jus.br/estatistica/sead/eleicoes/eleicoes%s/fotos/foto_cand%s_%s_div.zip"
	ttlFotos    = 24 * time.Hour
)

var (
	AnosFotos = map[string]bool{"2022": true, "2024": true, "2026": true}
	PadraoUE  = regexp.MustCompile(`^[A-Z]{2}$`)
	PadraoSQ  = regexp.MustCompile(`^[0-9]{1,15}$`)
)

// caminhoFoto monta o URL da foto servida pelo backend. O pacote do TSE é por
// UF (ou "BR" para a eleição presidencial).
func caminhoFoto(ano int, pacote, sq string) string {
	return fmt.Sprintf("/api/fotos/%d/%s/%s", ano, pacote, sq)
}

// Fotos serve as fotos de candidatos a partir dos pacotes por UF do TSE, lidos
// remotamente (ver zipRemoto). Só o diretório de cada pacote fica em memória.
type Fotos struct {
	mu      sync.Mutex
	pacotes map[string]*PacoteFotos
}

type PacoteFotos struct {
	Pronto chan struct{}  `json:"-"`
	Zip    *zipremoto.Zip `json:"-"`
	Err    error          `json:"-"`
	Expira time.Time      `json:"-"`
}

func NovasFotos() *Fotos {
	return &Fotos{pacotes: map[string]*PacoteFotos{}}
}

func (f *Fotos) Pacote(ctx context.Context, ano, uf string) (*zipremoto.Zip, error) {
	chave := ano + "/" + uf
	f.mu.Lock()
	p, ok := f.pacotes[chave]
	if !ok || (carga.Fechado(p.Pronto) && time.Now().After(p.Expira)) {
		p = &PacoteFotos{Pronto: make(chan struct{})}
		f.pacotes[chave] = p
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			inicio := time.Now()
			p.Zip, p.Err = zipremoto.Abrir(ctx, fmt.Sprintf(urlFotosTSE, ano, ano, uf))
			if p.Err != nil {
				log.Printf("fotos %s: %v", chave, p.Err)
				p.Expira = time.Now().Add(carga.TTLErro)
			} else {
				log.Printf("fotos %s: %d fotos indexadas em %s", chave, len(p.Zip.Arquivos), time.Since(inicio).Round(time.Millisecond))
				p.Expira = time.Now().Add(ttlFotos)
			}
			close(p.Pronto)
		}()
	}
	f.mu.Unlock()

	select {
	case <-p.Pronto:
		return p.Zip, p.Err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
