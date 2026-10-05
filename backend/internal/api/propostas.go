package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"quemeuvoto/internal/fontes/tse"
	"quemeuvoto/internal/propostas"
)

// GET /api/propostas?ano=2026&uf=SC&sq=240002544118
func propostaHandler(p *propostas.Propostas) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		ano, uf, sq := q.Get("ano"), strings.ToUpper(q.Get("uf")), q.Get("sq")
		if !propostas.ValidarAnoUF(ano, uf) || !tse.PadraoSQ.MatchString(sq) {
			escreverErro(w, "Parâmetros inválidos", http.StatusBadRequest)
			return
		}
		// Espera até 15 s; se for OCR e ainda não acabou, devolve a versão parcial ("a processar").
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		a, err := p.Analisar(ctx, ano, uf, sq)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) && r.Context().Err() == nil {
				escreverJSON(w, propostas.EmProcessamento())
				return
			}
			if r.Context().Err() == nil {
				escreverErro(w, "Não foi possível obter a proposta no TSE agora", http.StatusBadGateway)
			}
			return
		}
		escreverJSON(w, a)
	}
}

// GET /api/propostas/arquivo/{ano}/{uf}/{nome} — devolve o PDF tal como o TSE o publica.
func arquivoPropostaHandler(p *propostas.Propostas) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ano, uf := r.PathValue("ano"), strings.ToUpper(r.PathValue("uf"))
		nome := uf + "/" + strings.TrimSuffix(r.PathValue("nome"), ".pdf")
		if !propostas.ValidarAnoUF(ano, uf) || !propostas.PadraoArquivo.MatchString(nome) {
			http.NotFound(w, r)
			return
		}
		z, err := p.Pacote(r.Context(), ano, uf)
		if err != nil {
			escreverErro(w, "Propostas indisponíveis", http.StatusBadGateway)
			return
		}
		f, ok := z.Arquivos[nome]
		if !ok {
			http.NotFound(w, r)
			return
		}
		fluxo, err := z.AbrirFluxo(r.Context(), nome)
		if err != nil {
			escreverErro(w, "Proposta indisponível", http.StatusBadGateway)
			return
		}
		defer fluxo.Close()
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Length", strconv.FormatUint(f.UncompressedSize64, 10))
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="proposta-%s-%s.pdf"`, ano, strings.ReplaceAll(nome, "/", "-")))
		w.Header().Set("Cache-Control", "public, max-age=86400")
		io.Copy(w, fluxo)
	}
}
