package texto

import (
	"strings"
	"unicode"
)

// ResumirTexto corta textos longos numa fronteira de palavra.
func ResumirTexto(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	corte := max
	for corte > max-40 && r[corte] != ' ' {
		corte--
	}
	return string(r[:corte]) + "…"
}

var semAcentos = strings.NewReplacer(
	"Á", "A", "À", "A", "Â", "A", "Ã", "A", "Ä", "A",
	"É", "E", "È", "E", "Ê", "E", "Ë", "E",
	"Í", "I", "Ì", "I", "Î", "I", "Ï", "I",
	"Ó", "O", "Ò", "O", "Ô", "O", "Õ", "O", "Ö", "O",
	"Ú", "U", "Ù", "U", "Û", "U", "Ü", "U",
	"Ç", "C", "Ñ", "N",
)

// Normalizar põe o texto em maiúsculas, sem acentos e com espaços simples.
func Normalizar(s string) string {
	return strings.Join(strings.Fields(semAcentos.Replace(strings.ToUpper(s))), " ")
}

// TituloCargo converte "DEPUTADO FEDERAL" em "Deputado Federal".
func TituloCargo(cargo string) string {
	palavras := strings.Fields(strings.ToLower(cargo))
	for i, p := range palavras {
		r := []rune(p)
		r[0] = unicode.ToUpper(r[0])
		palavras[i] = string(r)
	}
	return strings.Join(palavras, " ")
}

// FraseCapitalizada converte "INDEFERIDO COM RECURSO" em "Indeferido com recurso".
func FraseCapitalizada(s string) string {
	if strings.HasPrefix(s, "#") {
		return ""
	}
	r := []rune(strings.ToLower(s))
	if len(r) > 0 {
		r[0] = unicode.ToUpper(r[0])
	}
	return string(r)
}

// SemMarcador troca os marcadores de "sem informação" do TSE (#NULO, #NE) por "".
func SemMarcador(s string) string {
	if strings.HasPrefix(s, "#") {
		return ""
	}
	return s
}

var preposicoes = map[string]bool{"de": true, "da": true, "do": true, "das": true, "dos": true, "e": true, "d'": true}

// TituloMunicipio converte "SÃO JOSÉ DA VARGINHA" em "São José da Varginha".
func TituloMunicipio(nome string) string {
	palavras := strings.Fields(strings.ToLower(nome))
	for i, p := range palavras {
		if i > 0 && preposicoes[p] {
			continue
		}
		palavras[i] = FraseCapitalizada(p)
	}
	return strings.Join(palavras, " ")
}

func ApenasDigitos(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
}
