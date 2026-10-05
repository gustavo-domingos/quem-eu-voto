package dominio

// Projeto é uma proposição (PL, PLP, PEC) apresentada pelo parlamentar.
type Projeto struct {
	Titulo string `json:"titulo"`
	Ementa string `json:"ementa"`
	Data   string `json:"data,omitempty"`
	URL    string `json:"url"`
}

const MaxVotosRecentes = 15

// VotoRecente é um voto do deputado numa votação nominal do Plenário.
type VotoRecente struct {
	Data              string `json:"data"`
	Proposicao        string `json:"proposicao"`
	Ementa            string `json:"ementa"`
	Descricao         string `json:"descricao"`
	Voto              string `json:"voto"`
	OrientacaoGoverno string `json:"orientacaoGoverno,omitempty"`
}

// Periodo é um intervalo de exercício do mandato (fim vazio = até hoje).
type Periodo struct{ Inicio, Fim string }
