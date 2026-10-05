// Package api expõe o serviço por HTTP: rotas, leitura dos parâmetros e tradução de erros.
// A regra de negócio fica em internal/transparencia; aqui só se fala HTTP.
package api

import (
	"net/http"

	"quemeuvoto/internal/fontes/noticias"
	"quemeuvoto/internal/fontes/tse"
	"quemeuvoto/internal/propostas"
	"quemeuvoto/internal/transparencia"
)

// Dependencias são os serviços que a API publica.
type Dependencias struct {
	Servico   *transparencia.Servico
	Fotos     *tse.Fotos
	Noticias  *noticias.Noticias
	Propostas *propostas.Propostas
}

// Config ajusta o comportamento HTTP.
type Config struct {
	// OrigensPermitidas lista as origens aceites pelo CORS; vazia ou com "*" aceita qualquer uma.
	OrigensPermitidas []string
}

// NovoRoteador monta o roteador da API pública com os middlewares comuns.
func NovoRoteador(dep Dependencias, cfg Config) http.Handler {
	s := dep.Servico
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", saude)

	mux.HandleFunc("GET /api/candidatos", candidatosHandler(s))
	mux.HandleFunc("GET /api/eleitos", eleitosHandler(s))
	mux.HandleFunc("GET /api/deputados", deputadosHandler(s))
	mux.HandleFunc("GET /api/perfil", perfilHandler(s))
	mux.HandleFunc("GET /api/partidos", partidosHandler(s))
	mux.HandleFunc("GET /api/municipios", municipiosHandler(s))
	mux.HandleFunc("GET /api/vagas", vagasHandler(s))
	mux.HandleFunc("GET /api/apuracao", apuracaoHandler(s))

	mux.HandleFunc("GET /api/noticias", noticiasHandler(dep.Noticias))
	mux.HandleFunc("GET /api/propostas", propostaHandler(dep.Propostas))
	mux.HandleFunc("GET /api/propostas/arquivo/{ano}/{uf}/{nome}", arquivoPropostaHandler(dep.Propostas))
	mux.HandleFunc("GET /api/fotos/{ano}/{uf}/{sq}", fotoHandler(dep.Fotos))

	return recuperar(registrar(cors(cfg.OrigensPermitidas, mux)))
}

func saude(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("ok"))
}
