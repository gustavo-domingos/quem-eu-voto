package api

import (
	"net/http"
	"strconv"

	"quemeuvoto/internal/transparencia"
)

// filtroDe lê os parâmetros comuns às listagens; a validação fica no serviço.
func filtroDe(r *http.Request) transparencia.Filtro {
	q := r.URL.Query()
	pagina, _ := strconv.Atoi(q.Get("pagina"))
	return transparencia.Filtro{
		Cargo:          q.Get("cargo"),
		Uf:             q.Get("uf"),
		Municipio:      q.Get("municipio"),
		Termo:          q.Get("q"),
		Ordem:          q.Get("ordem"),
		Partido:        q.Get("partido"),
		Genero:         q.Get("genero"),
		Faixa:          q.Get("faixa"),
		Nivel:          q.Get("nivel"),
		Em2026:         q.Get("em2026"),
		IncluirInaptos: q.Get("inaptos") == "1",
		SoJustica:      q.Get("justica") == "1",
		Pagina:         pagina,
	}
}

// GET /api/candidatos?cargo=&uf=&q=&ordem=&partido=&genero=&faixa=&inaptos=0|1&justica=0|1&pagina=
func candidatosHandler(s *transparencia.Servico) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := s.ListarCandidatos(r.Context(), filtroDe(r))
		responder(w, r, res, err)
	}
}

// GET /api/eleitos?cargo=&uf=&municipio=&q=&em2026=&ordem=&partido=&genero=&faixa=&nivel=&justica=&pagina=
func eleitosHandler(s *transparencia.Servico) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := s.ListarEleitos(r.Context(), filtroDe(r))
		responder(w, r, res, err)
	}
}

// GET /api/deputados?uf={UF|TODOS}
func deputadosHandler(s *transparencia.Servico) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := s.ListarDeputados(r.Context(), r.URL.Query().Get("uf"))
		responder(w, r, res, err)
	}
}

// GET /api/perfil?id={SQ_CANDIDATO | camara-<id> | senado-<código>}
func perfilHandler(s *transparencia.Servico) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := s.Perfil(r.Context(), r.URL.Query().Get("id"))
		responder(w, r, res, err)
	}
}

// GET /api/partidos
func partidosHandler(s *transparencia.Servico) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := s.ListarPartidos(r.Context())
		responder(w, r, res, err)
	}
}

// GET /api/municipios?uf={UF}
func municipiosHandler(s *transparencia.Servico) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := s.Municipios(r.Context(), r.URL.Query().Get("uf"))
		responder(w, r, res, err)
	}
}

// GET /api/vagas?uf={UF|TODOS}&municipio={código}
func vagasHandler(s *transparencia.Servico) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		res, err := s.Vagas(r.Context(), q.Get("uf"), q.Get("municipio"))
		responder(w, r, res, err)
	}
}

// GET /api/apuracao?cargo=governador&uf=SC
func apuracaoHandler(s *transparencia.Servico) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		res, err := s.Apuracao(r.Context(), q.Get("cargo"), q.Get("uf"))
		if err == nil {
			// Os dados são públicos e mudam a cada minuto: nada de cache no navegador.
			w.Header().Set("Cache-Control", "no-store")
		}
		responder(w, r, res, err)
	}
}
