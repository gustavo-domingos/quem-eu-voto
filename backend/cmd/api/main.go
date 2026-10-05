// Comando api é o servidor HTTP do Quem eu voto.
//
// Configuração por variáveis de ambiente: PORT, CORS_ORIGINS e CACHE_DIR (ver internal/config).
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"quemeuvoto/internal/api"
	"quemeuvoto/internal/config"
	"quemeuvoto/internal/fontes/noticias"
	"quemeuvoto/internal/fontes/tse"
	"quemeuvoto/internal/plataforma/cache"
	"quemeuvoto/internal/propostas"
	"quemeuvoto/internal/transparencia"
)

func main() {
	cfg := config.Carregar()
	cache.DefinirPasta(cfg.PastaCache)

	servico := transparencia.Novo()
	servico.Iniciar()

	roteador := api.NovoRoteador(api.Dependencias{
		Servico:   servico,
		Fotos:     tse.NovasFotos(),
		Noticias:  noticias.Novo(),
		Propostas: propostas.Novo(),
	}, api.Config{OrigensPermitidas: cfg.OrigensCORS})

	srv := &http.Server{
		Addr:              ":" + cfg.Porta,
		Handler:           roteador,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
		// Sem WriteTimeout: os PDFs das propostas e a análise por OCR podem demorar.
	}

	ctx, parar := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer parar()

	go func() {
		log.Printf("servidor em http://localhost:%s", cfg.Porta)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("erro ao iniciar o servidor: %v", err)
		}
	}()

	<-ctx.Done()
	log.Print("a encerrar…")
	desligar, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	if err := srv.Shutdown(desligar); err != nil {
		log.Printf("encerramento: %v", err)
	}
}
