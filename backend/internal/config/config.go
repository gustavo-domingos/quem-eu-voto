// Package config lê a configuração do servidor a partir de variáveis de ambiente.
package config

import (
	"os"
	"strings"
)

// Config é a configuração do processo. Todos os campos têm valores por omissão para desenvolvimento.
type Config struct {
	// Porta HTTP (PORT). Por omissão, 8080.
	Porta string
	// Origens aceites pelo CORS (CORS_ORIGINS, separadas por vírgula). Vazio = qualquer origem.
	OrigensCORS []string
	// Pasta da cache em disco (CACHE_DIR). Vazio = pasta de cache do utilizador.
	PastaCache string
}

// Carregar lê as variáveis de ambiente.
func Carregar() Config {
	c := Config{
		Porta:      valor("PORT", "8080"),
		PastaCache: os.Getenv("CACHE_DIR"),
	}
	for _, o := range strings.Split(os.Getenv("CORS_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			c.OrigensCORS = append(c.OrigensCORS, strings.TrimSuffix(o, "/"))
		}
	}
	return c
}

func valor(nome, padrao string) string {
	if v := strings.TrimSpace(os.Getenv(nome)); v != "" {
		return v
	}
	return padrao
}
