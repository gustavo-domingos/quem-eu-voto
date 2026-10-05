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

// ErroDTO é o corpo de todas as respostas de erro: o frontend mostra Erro a quem usa o site.
type ErroDTO struct {
	Erro   string `json:"erro"`
	Status int    `json:"status"`
}

// escreverErro responde com o código HTTP e a mensagem em JSON.
func escreverErro(w http.ResponseWriter, mensagem string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErroDTO{Erro: mensagem, Status: status})
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
		escreverErro(w, e.Mensagem, statusPorTipo[e.Tipo])
		return
	}
	log.Printf("%s %s: %v", r.Method, r.URL.RequestURI(), err)
	escreverErro(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
}

// responder escreve o resultado de uma chamada ao serviço, ou o erro.
func responder[T any](w http.ResponseWriter, r *http.Request, v T, err error) {
	if err != nil {
		responderErro(w, r, err)
		return
	}
	escreverJSON(w, v)
}
