package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"quemeuvoto/internal/transparencia"
)

func escreverJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("escrever resposta: %v", err)
	}
}

var statusPorTipo = map[transparencia.TipoErro]int{
	transparencia.EntradaInvalida: http.StatusBadRequest,
	transparencia.NaoEncontrado:   http.StatusNotFound,
	transparencia.Indisponivel:    http.StatusBadGateway,
}

// responderErro traduz um erro do serviço na resposta HTTP certa.
func responderErro(w http.ResponseWriter, r *http.Request, err error) {
	if r.Context().Err() != nil {
		return // quem pediu já desistiu: não há a quem responder
	}
	var e *transparencia.Erro
	if errors.As(err, &e) {
		http.Error(w, e.Mensagem, statusPorTipo[e.Tipo])
		return
	}
	log.Printf("%s %s: %v", r.Method, r.URL.RequestURI(), err)
	http.Error(w, "Erro interno", http.StatusInternalServerError)
}

// responder escreve o resultado de uma chamada ao serviço, ou o erro.
func responder[T any](w http.ResponseWriter, r *http.Request, v T, err error) {
	if err != nil {
		responderErro(w, r, err)
		return
	}
	escreverJSON(w, v)
}
