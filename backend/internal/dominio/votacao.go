package dominio

import (
	"time"
)

// Votacao é o resultado de um candidato no último turno que disputou.
type Votacao struct {
	Votos      int64   `json:"v"`
	Turno      int     `json:"t"`
	Posicao    int     `json:"p"`  // posição entre os candidatos ao mesmo cargo, na mesma circunscrição e eleição
	Disputaram int     `json:"d"`  // quantos candidatos tiveram votos nessa disputa
	Percentual float64 `json:"pc"` // % dos votos nominais válidos da disputa
}

type ResultadoVotacao struct {
	Ano          int                 `json:"ano"`
	PorCandidato map[string]*Votacao `json:"porCandidato"` // por SQ_CANDIDATO
	GeradoEm     time.Time           `json:"geradoEm"`
}
