package dominio

import (
	"strings"

	"quemeuvoto/internal/plataforma/texto"
)

// ChaveNomeNasc junta o nome normalizado e a data de nascimento.
func ChaveNomeNasc(nome, nascimento string) string {
	return texto.Normalizar(nome) + "|" + nascimento
}

// GeneroCurto converte "FEMININO"/"Feminino"/"F" em "F" (e o mesmo para masculino).
func GeneroCurto(g string) string {
	switch strings.ToUpper(strings.TrimSpace(g)) {
	case "F", "FEMININO":
		return "F"
	case "M", "MASCULINO":
		return "M"
	}
	return ""
}
