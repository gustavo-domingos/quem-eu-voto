package api

import (
	"log"
	"net/http"
	"slices"
	"time"
)

// cors libera a API para o frontend. Com a lista vazia (ou "*"), aceita qualquer origem.
func cors(origens []string, h http.Handler) http.Handler {
	todas := len(origens) == 0 || slices.Contains(origens, "*")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if todas {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if o := r.Header.Get("Origin"); slices.Contains(origens, o) {
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// recuperar transforma um panic num 500, em vez de derrubar a ligação sem resposta.
func recuperar(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				if p == http.ErrAbortHandler {
					panic(p)
				}
				log.Printf("panic em %s %s: %v", r.Method, r.URL.Path, p)
				escreverErro(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
			}
		}()
		h.ServeHTTP(w, r)
	})
}

type gravadorStatus struct {
	http.ResponseWriter
	status int
}

func (g *gravadorStatus) WriteHeader(s int) {
	g.status = s
	g.ResponseWriter.WriteHeader(s)
}

// Flush deixa passar o streaming (ex.: PDFs grandes das propostas).
func (g *gravadorStatus) Flush() {
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// registrar põe no log os pedidos com erro do servidor ou lentos (as fontes oficiais às vezes demoram).
func registrar(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		g := &gravadorStatus{ResponseWriter: w, status: http.StatusOK}
		h.ServeHTTP(g, r)
		if d := time.Since(inicio); g.status >= 500 || d > 10*time.Second {
			log.Printf("%s %s -> %d em %s", r.Method, r.URL.RequestURI(), g.status, d.Round(time.Millisecond))
		}
	})
}
