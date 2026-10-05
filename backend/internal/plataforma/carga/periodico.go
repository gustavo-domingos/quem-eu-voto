package carga

import (
	"context"
	"log"
	"sync"
	"time"
)

// Periodico mantém um valor recarregado em segundo plano a intervalos fixos.
type Periodico[T any] struct {
	nome      string
	intervalo time.Duration
	carregar  func(context.Context) (*T, error)

	mu     sync.RWMutex
	valor  *T
	pronto chan struct{} // fechado após a primeira tentativa de carga
}

func NovoPeriodico[T any](nome string, intervalo time.Duration, carregar func(context.Context) (*T, error)) *Periodico[T] {
	return &Periodico[T]{nome: nome, intervalo: intervalo, carregar: carregar, pronto: make(chan struct{})}
}

func (p *Periodico[T]) Executar() {
	primeira := true
	for {
		inicio := time.Now()
		v, err := p.carregar(context.Background())
		espera := p.intervalo
		if err != nil {
			log.Printf("%s: falha na carga: %v", p.nome, err)
			espera = 10 * time.Minute
		} else {
			p.mu.Lock()
			p.valor = v
			p.mu.Unlock()
			log.Printf("%s: carregado em %s", p.nome, time.Since(inicio).Round(time.Second))
		}
		if primeira {
			close(p.pronto)
			primeira = false
		}
		time.Sleep(espera)
	}
}

// Obter espera pela primeira carga (ou pelo cancelamento do pedido) e devolve
// o valor atual, que pode ser nil se a carga falhou.
func (p *Periodico[T]) Obter(ctx context.Context) *T {
	select {
	case <-p.pronto:
	case <-ctx.Done():
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.valor
}
