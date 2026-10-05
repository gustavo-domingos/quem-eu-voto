package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"quemeuvoto/internal/fontes/noticias"
)

// GET /api/noticias?nome={nome}
func noticiasHandler(n *noticias.Noticias) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nome := strings.TrimSpace(r.URL.Query().Get("nome"))
		if len([]rune(nome)) < 4 || len([]rune(nome)) > 80 {
			http.Error(w, "Nome inválido", http.StatusBadRequest)
			return
		}
		resposta := noticias.RespostaNoticiasDTO{Noticias: []noticias.Noticia{}, Pesquisa: noticias.LinksDePesquisa(nome)}
		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()
		noticias, err := n.Buscar(ctx, nome)
		if err != nil {
			resposta.Aviso = "Não foi possível consultar a base de notícias agora. Use as pesquisas abaixo."
		} else {
			resposta.Noticias = noticias
		}
		escreverJSON(w, resposta)
	}
}
