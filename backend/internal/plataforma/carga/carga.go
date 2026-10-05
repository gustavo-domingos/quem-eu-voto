package carga

import (
	"time"
)

const (
	// TTLPorID é a validade de um valor obtido com sucesso por PorID.
	TTLPorID = 24 * time.Hour
	// TTLErro é quanto tempo se espera antes de repetir uma busca que falhou.
	TTLErro = 2 * time.Minute
)

// Fechado diz, sem bloquear, se o canal já foi fechado.
func Fechado(c chan struct{}) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}
