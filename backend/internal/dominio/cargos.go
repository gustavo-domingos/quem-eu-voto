package dominio

// Cargos que a aplicação lista, agrupados pela chave usada no filtro.
// Deputado Distrital (DF) é o equivalente a Deputado Estadual.
var CargosPrincipais = map[string]string{
	"Presidente":         "presidente",
	"Governador":         "governador",
	"Senador":            "senador",
	"Deputado Federal":   "deputado-federal",
	"Deputado Estadual":  "deputado-estadual",
	"Deputado Distrital": "deputado-estadual",
}

// Vices e suplentes são mostrados junto do titular com o mesmo número na mesma circunscrição.
var CargoDoTitular = map[string]string{
	"Vice-presidente": "Presidente",
	"Vice-governador": "Governador",
	"1º Suplente":     "Senador",
	"2º Suplente":     "Senador",
}

// Cada estado tem sempre 3 senadores (Constituição, art. 46); em cada eleição renova-se 1 ou 2.
const CadeirasSenadoPorUF = 3
