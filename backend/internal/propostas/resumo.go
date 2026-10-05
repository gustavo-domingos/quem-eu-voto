package propostas

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/texto"
)

// Resumo de propostas de governo sem IA: as frases são sempre transcritas do documento,
// nunca reescritas; o programa só escolhe quais mostrar e em que tema as arruma.

// TemaResumo agrupa os compromissos de um tema.
type TemaResumo struct {
	Tema         string   `json:"tema"`
	Total        int      `json:"total"`        // quantos compromissos foram encontrados neste tema
	Compromissos []string `json:"compromissos"` // os mais concretos (com números primeiro)
}

type compromisso struct {
	texto  string
	tema   string
	numero bool
	forte  bool // começa por verbo ("Criar…") ou forma nominal ("Criação de…"); fraco = verbo no futuro
}

const temaOutros = "Outros compromissos"

var (
	verbosCompromisso = regexp.MustCompile(`(?i)^(vamos\s+)?(criar|implantar|ampliar|construir|garantir|investir|reduzir|implementar|fortalecer|expandir|modernizar|valorizar|promover|combater|instituir|zerar|duplicar|contratar|universalizar|retomar|recuperar|entregar|estruturar|reformar|apoiar|incentivar|estimular|assegurar|aumentar|melhorar|desenvolver|eliminar|acabar|oferecer|disponibilizar|estabelecer|viabilizar|priorizar|revisar|regularizar|integrar|levar|equipar|qualificar|capacitar|aprimorar|democratizar|descentralizar|informatizar|digitalizar|requalificar|pavimentar|duplicar|concluir|triplicar|dobrar)\b`)
	// "Criação de…", "Ampliação do…": muitas propostas listam as ações como substantivos.
	inicioNominal = regexp.MustCompile(`(?i)^(criação|implantação|ampliação|construção|garantia|redução|implementação|fortalecimento|expansão|modernização|valorização|promoção|instituição|contratação|universalização|retomada|recuperação|estruturação|reforma|incentivo|estímulo|aumento|melhoria|eliminação|oferta|disponibilização|regularização|integração|qualificação|capacitação|aprimoramento|pavimentação|duplicação|conclusão|requalificação|digitalização|informatização|descentralização|revitalização|instalação|aquisição)\s+(de|da|do|das|dos)\s`)
	// Compromissos no futuro ("O Estado investirá…", "Vamos criar…"). "será"/"terá" ficam de fora
	// (raiz curta), porque aparecem em frases genéricas. O \b do Go só vale para ASCII, por isso
	// a fronteira depois do "á" é escrita como "não é letra" (\P{L}).
	verboFuturo     = regexp.MustCompile(`(?i)(?:^|\P{L})(vamos|iremos)\s+\p{L}+|\p{L}{3,}(ará|arão|erá|erão|irá|irão)(?:\P{L}|$)`)
	fimDeFrase      = regexp.MustCompile(`[.;!?•●▪■◦]\s+`)
	tituloNumerado  = regexp.MustCompile(`^\d+(\.\d+)*\.?\s+\p{Lu}`)
	naoAlfanumerico = regexp.MustCompile(`[^A-Z0-9]+`)
)

// textoParaContagem normaliza e troca pontuação por espaços, para contar palavras inteiras.
func textoParaContagem(s string) string {
	return " " + naoAlfanumerico.ReplaceAllString(texto.Normalizar(s), " ") + " "
}

// contarPalavra conta uma palavra-chave. As curtas ("SUS", "UPA") só contam como palavra
// inteira, para "SUS" não contar dentro de "SUSTENTAVEL"; as longas valem como prefixo.
func contarPalavra(texto, palavra string) int {
	if len(palavra) <= 4 {
		return strings.Count(texto, " "+palavra+" ")
	}
	return strings.Count(texto, " "+palavra)
}

// contarTemas devolve quantas vezes cada tema aparece no texto, do mais citado ao menos.
func contarTemas(conteudo string) []dominio.Contagem {
	norm := textoParaContagem(conteudo)
	res := []dominio.Contagem{}
	for _, t := range temasPropostas {
		n := 0
		for _, p := range t.palavras {
			n += contarPalavra(norm, p)
		}
		if n > 0 {
			res = append(res, dominio.Contagem{Rotulo: t.tema, Total: n})
		}
	}
	sort.SliceStable(res, func(i, j int) bool { return res[i].Total > res[j].Total })
	return res
}

// temaDe escolhe o tema com mais palavras-chave no trecho ("" se nenhum).
func temaDe(trecho string) string {
	norm := textoParaContagem(trecho)
	melhor, maior := "", 0
	for _, t := range temasPropostas {
		n := 0
		for _, p := range t.palavras {
			n += contarPalavra(norm, p)
		}
		if n > maior {
			melhor, maior = t.tema, n
		}
	}
	return melhor
}

// ehTitulo reconhece títulos de secção: linhas curtas em maiúsculas ou numeradas ("4. Saúde").
func ehTitulo(p string) bool {
	n := len([]rune(p))
	if n == 0 || n > 90 || strings.HasSuffix(p, ".") || verbosCompromisso.MatchString(p) {
		return false
	}
	return maiusculas(p) || tituloNumerado.MatchString(p)
}

// paragrafosDe junta as linhas que continuam a mesma frase (a seguinte começa em minúscula).
func paragrafosDe(conteudo string) []string {
	var paragrafos []string
	var atual strings.Builder
	for _, linha := range strings.Split(conteudo, "\n") {
		linha = strings.TrimSpace(linha)
		if linha == "" {
			continue
		}
		if atual.Len() > 0 && unicode.IsLower([]rune(linha)[0]) {
			atual.WriteString(" " + linha)
			continue
		}
		if atual.Len() > 0 {
			paragrafos = append(paragrafos, atual.String())
		}
		atual.Reset()
		atual.WriteString(linha)
	}
	if atual.Len() > 0 {
		paragrafos = append(paragrafos, atual.String())
	}
	return paragrafos
}

// extrairFrases devolve, pela ordem do documento, as frases que começam por um verbo de ação.
// O tema vem das palavras da própria frase ou, se não houver, do título da secção onde está.
func extrairFrases(conteudo string) []compromisso {
	var res []compromisso
	vistos := map[string]bool{}
	secao := ""
	for _, p := range paragrafosDe(conteudo) {
		if ehTitulo(p) {
			secao = temaDe(p)
			continue
		}
		for _, frase := range fimDeFrase.Split(p, -1) {
			frase = strings.TrimLeft(strings.Join(strings.Fields(frase), " "), "-–—•●▪■◦*0123456789.)( ")
			frase = strings.TrimRight(frase, ",;:.- ")
			n := len([]rune(frase))
			if n < 40 || n > 260 || maiusculas(frase) || terminaCortada(frase) {
				continue
			}
			forte := verbosCompromisso.MatchString(frase) || inicioNominal.MatchString(frase+" ")
			// Frases no futuro só contam se falarem de um tema concreto (evita generalidades).
			if !forte && (!verboFuturo.MatchString(frase) || temaDe(frase) == "") {
				continue
			}
			chave := texto.Normalizar(frase)
			if vistos[chave] {
				continue
			}
			vistos[chave] = true
			tema := temaDe(frase)
			if tema == "" {
				tema = secao
			}
			if tema == "" {
				tema = temaOutros
			}
			// Só a primeira letra passa a maiúscula (listas costumam começar em minúscula); o resto fica igual.
			letras := []rune(frase)
			letras[0] = unicode.ToUpper(letras[0])
			res = append(res, compromisso{texto: string(letras), tema: tema, forte: forte, numero: strings.IndexFunc(frase, unicode.IsDigit) >= 0})
		}
	}
	return res
}

// maisConcretos ordena com as frases que têm números primeiro, mantendo a ordem do documento.
func maisConcretos(lista []compromisso, max int) []string {
	ordenada := append([]compromisso{}, lista...)
	// Primeiro as que têm números; depois as que começam por verbo/ação; por fim as do futuro.
	peso := func(c compromisso) int {
		p := 0
		if c.numero {
			p += 2
		}
		if c.forte {
			p++
		}
		return p
	}
	sort.SliceStable(ordenada, func(i, j int) bool { return peso(ordenada[i]) > peso(ordenada[j]) })
	res := []string{}
	for _, c := range ordenada {
		if len(res) == max {
			break
		}
		res = append(res, c.texto)
	}
	return res
}

// extrairCompromissos devolve os compromissos mais concretos do documento inteiro.
func extrairCompromissos(conteudo string, max int) []string {
	return maisConcretos(extrairFrases(conteudo), max)
}

// resumirPorTema agrupa os compromissos por tema (temas com mais compromissos primeiro;
// "Outros" fica no fim) e guarda até porTema frases de cada um.
func resumirPorTema(frases []compromisso, porTema int) []TemaResumo {
	grupos := map[string][]compromisso{}
	var ordem []string
	for _, c := range frases {
		if _, ok := grupos[c.tema]; !ok {
			ordem = append(ordem, c.tema)
		}
		grupos[c.tema] = append(grupos[c.tema], c)
	}
	res := []TemaResumo{}
	for _, tema := range ordem {
		res = append(res, TemaResumo{Tema: tema, Total: len(grupos[tema]), Compromissos: maisConcretos(grupos[tema], porTema)})
	}
	sort.SliceStable(res, func(i, j int) bool {
		if (res[i].Tema == temaOutros) != (res[j].Tema == temaOutros) {
			return res[j].Tema == temaOutros
		}
		return res[i].Total > res[j].Total
	})
	return res
}
