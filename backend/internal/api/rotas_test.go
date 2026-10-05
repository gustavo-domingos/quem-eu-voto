package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"quemeuvoto/internal/transparencia"
)

// roteador sem Iniciar(): nenhuma fonte é carregada, por isso os testes não usam a rede.
func roteador(origens ...string) http.Handler {
	return NovoRoteador(Dependencias{Servico: transparencia.Novo()}, Config{OrigensPermitidas: origens})
}

func pedir(h http.Handler, metodo, url string, cabecalhos ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(metodo, url, nil)
	for i := 0; i+1 < len(cabecalhos); i += 2 {
		req.Header.Set(cabecalhos[i], cabecalhos[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSaude(t *testing.T) {
	rec := pedir(roteador(), http.MethodGet, "/healthz")
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("healthz = %d %q", rec.Code, rec.Body.String())
	}
}

func TestParametrosInvalidos(t *testing.T) {
	casos := map[string]string{
		"/api/candidatos?cargo=senador&uf=XX": "UF inválida",
		"/api/candidatos?cargo=xx&uf=SC":      "Cargo inválido",
		"/api/eleitos?cargo=xx":               "Cargo inválido",
		"/api/vagas?uf=XX":                    "UF inválida",
		"/api/municipios?uf=":                 "UF inválida",
		"/api/apuracao?cargo=governador":      "Escolha um estado",
		"/api/perfil?id=":                     "Identificador inválido",
	}
	h := roteador()
	for url, msg := range casos {
		rec := pedir(h, http.MethodGet, url)
		var corpo ErroDTO
		if err := json.Unmarshal(rec.Body.Bytes(), &corpo); err != nil {
			t.Errorf("%s: corpo não é JSON: %q", url, rec.Body.String())
			continue
		}
		if rec.Code != http.StatusBadRequest || corpo.Erro != msg || corpo.Status != http.StatusBadRequest {
			t.Errorf("%s = %d %+v; esperado 400 %q", url, rec.Code, corpo, msg)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("%s: Content-Type = %q", url, ct)
		}
	}
}

func TestMetodoNaoPermitido(t *testing.T) {
	if rec := pedir(roteador(), http.MethodPost, "/api/candidatos"); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST = %d, esperado 405", rec.Code)
	}
}

func TestCORS(t *testing.T) {
	rec := pedir(roteador(), http.MethodOptions, "/api/candidatos")
	if rec.Code != http.StatusOK || rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("preflight sem lista = %d, origem %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}

	h := roteador("https://quemeuvoto.example")
	rec = pedir(h, http.MethodGet, "/healthz", "Origin", "https://quemeuvoto.example")
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://quemeuvoto.example" {
		t.Errorf("origem permitida: cabeçalho = %q", got)
	}
	rec = pedir(h, http.MethodGet, "/healthz", "Origin", "https://outro.example")
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("origem não permitida recebeu %q", got)
	}
}

func TestResponderErro(t *testing.T) {
	casos := []struct {
		err    error
		status int
	}{
		{&transparencia.Erro{Tipo: transparencia.NaoEncontrado, Mensagem: "x"}, http.StatusNotFound},
		{&transparencia.Erro{Tipo: transparencia.Indisponivel, Mensagem: "x"}, http.StatusBadGateway},
		{errors.New("inesperado"), http.StatusInternalServerError},
	}
	for _, c := range casos {
		rec := httptest.NewRecorder()
		responderErro(rec, httptest.NewRequest(http.MethodGet, "/", nil), c.err)
		if rec.Code != c.status {
			t.Errorf("%v = %d, esperado %d", c.err, rec.Code, c.status)
		}
	}
}
