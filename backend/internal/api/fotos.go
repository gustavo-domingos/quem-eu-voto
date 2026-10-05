package api

import (
	"fmt"
	"log"
	"mime"
	"net/http"

	"quemeuvoto/internal/fontes/tse"
)

// Handler responde a GET /api/fotos/{ano}/{uf}/{sq}.
func fotoHandler(f *tse.Fotos) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ano, uf, sq := r.PathValue("ano"), r.PathValue("uf"), r.PathValue("sq")
		if !tse.AnosFotos[ano] || !tse.PadraoUE.MatchString(uf) || !tse.PadraoSQ.MatchString(sq) {
			http.NotFound(w, r)
			return
		}

		zr, err := f.Pacote(r.Context(), ano, uf)
		if err != nil {
			http.Error(w, "Fotos indisponíveis", http.StatusBadGateway)
			return
		}
		conteudo, extensao, existe, err := zr.Ler(fmt.Sprintf("F%s%s_div", uf, sq))
		if !existe {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			log.Printf("foto %s/%s/%s: %v", ano, uf, sq, err)
			http.Error(w, "Foto indisponível", http.StatusBadGateway)
			return
		}

		tipo := mime.TypeByExtension(extensao)
		if tipo == "" {
			tipo = "image/jpeg"
		}
		w.Header().Set("Content-Type", tipo)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(conteudo)
	}
}
