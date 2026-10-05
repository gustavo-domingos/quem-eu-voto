// Package dominio reúne os conceitos partilhados por todas as fontes de dados: o calendário
// eleitoral, os cargos, as UFs e os tipos simples (votação, período de mandato, contagens).
// Não depende de nenhuma fonte nem da camada HTTP.
package dominio

// Calendário eleitoral considerado pela aplicação.
const (
	// AnoEleicao é a eleição geral em curso (presidente, governadores, Congresso e assembleias).
	AnoEleicao = 2026
	// AnoGeral2022 é a eleição geral anterior: quem exerce hoje os cargos gerais foi eleito nela.
	AnoGeral2022 = 2022
	// AnoMunicipal: não há eleições municipais em 2026; mostramos quem foi eleito em 2024 (mandato 2025–2028).
	AnoMunicipal = 2024

	// InicioLegislatura é o início da 57ª legislatura (2023–2027). As métricas consideram apenas este período.
	InicioLegislatura = "2023-02-01"
	AnoInicioLeg      = 2023
)

// UFsValidas são as 26 UFs e o Distrito Federal.
var UFsValidas = map[string]bool{
	"AC": true, "AL": true, "AP": true, "AM": true, "BA": true, "CE": true, "DF": true,
	"ES": true, "GO": true, "MA": true, "MT": true, "MS": true, "MG": true, "PA": true,
	"PB": true, "PR": true, "PE": true, "PI": true, "RJ": true, "RN": true, "RS": true,
	"RO": true, "RR": true, "SC": true, "SP": true, "SE": true, "TO": true,
}

// Vice é o vice ou suplente que concorre na chapa do titular.
type Vice struct {
	Cargo    string `json:"cargo"`
	NomeUrna string `json:"nomeUrna"`
	Partido  string `json:"partido"`
}
