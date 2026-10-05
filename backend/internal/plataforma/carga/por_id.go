package carga

import (
	"context"
	"log"
	"sync"
	"time"

	"quemeuvoto/internal/plataforma/cache"
)

// PorID é uma cache com TTL de um valor obtido por deputado, que junta
// pedidos simultâneos para o mesmo ID numa única chamada à Câmara.
type PorID[T any] struct {
	nome   string
	sem    chan struct{}
	buscar func(context.Context, int) (T, error)

	mu       sync.Mutex
	entradas map[int]*entrada[T]
}

type entrada[T any] struct {
	pronto chan struct{}
	valor  T
	err    error
	expira time.Time
}

func NovoPorID[T any](nome string, sem chan struct{}, buscar func(context.Context, int) (T, error)) *PorID[T] {
	return &PorID[T]{nome: nome, sem: sem, buscar: buscar, entradas: map[int]*entrada[T]{}}
}

// CarregarDisco põe na cache os valores guardados num arranque anterior (se recentes).
func (p *PorID[T]) CarregarDisco(nomeArquivo string, validade time.Duration) {
	salvos, ok := cache.Ler[map[int]T](nomeArquivo, validade)
	if !ok {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, v := range *salvos {
		if _, existe := p.entradas[id]; existe {
			continue
		}
		e := &entrada[T]{pronto: make(chan struct{}), valor: v, expira: time.Now().Add(validade)}
		close(e.pronto)
		p.entradas[id] = e
	}
	log.Printf("%s: %d valores lidos da cache em disco", p.nome, len(*salvos))
}

// SalvarDisco guarda os valores obtidos com sucesso.
func (p *PorID[T]) SalvarDisco(nomeArquivo string) {
	p.mu.Lock()
	valores := map[int]T{}
	for id, e := range p.entradas {
		if Fechado(e.pronto) && e.err == nil {
			valores[id] = e.valor
		}
	}
	p.mu.Unlock()
	if len(valores) == 0 {
		return
	}
	if err := cache.Gravar(nomeArquivo, valores); err != nil {
		log.Printf("%s: não foi possível gravar a cache: %v", p.nome, err)
	}
}

// ObterVarios devolve os valores por ID; IDs cuja busca falhou ficam de fora do mapa.
func (p *PorID[T]) ObterVarios(ctx context.Context, ids []int) map[int]T {
	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		res = make(map[int]T, len(ids))
	)
	for _, id := range ids {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if v, err := p.Obter(ctx, id); err == nil {
				mu.Lock()
				res[id] = v
				mu.Unlock()
			}
		}(id)
	}
	wg.Wait()
	return res
}

func (p *PorID[T]) Obter(ctx context.Context, id int) (T, error) {
	p.mu.Lock()
	e, ok := p.entradas[id]
	emCurso := ok && !Fechado(e.pronto)
	if !ok || (!emCurso && time.Now().After(e.expira)) {
		e = &entrada[T]{pronto: make(chan struct{})}
		p.entradas[id] = e
		go p.preencher(id, e)
	}
	p.mu.Unlock()

	select {
	case <-e.pronto:
		return e.valor, e.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// preencher corre desligado do pedido HTTP, para que um pedido cancelado
// não desperdice o trabalho que outros pedidos estão à espera de reutilizar.
func (p *PorID[T]) preencher(id int, e *entrada[T]) {
	p.sem <- struct{}{}
	defer func() { <-p.sem }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	e.valor, e.err = p.buscar(ctx, id)
	if e.err != nil {
		log.Printf("%s do deputado %d: %v", p.nome, id, e.err)
		e.expira = time.Now().Add(TTLErro)
	} else {
		e.expira = time.Now().Add(TTLPorID)
	}
	close(e.pronto)
}
