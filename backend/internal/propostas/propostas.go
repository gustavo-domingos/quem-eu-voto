package propostas

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/ledongthuc/pdf"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/fontes/tse"
	"quemeuvoto/internal/plataforma/cache"
	"quemeuvoto/internal/plataforma/carga"
	"quemeuvoto/internal/plataforma/ocr"
	"quemeuvoto/internal/plataforma/zipremoto"
)

// Só os candidatos a Presidente, Governador e Prefeito são obrigados a entregar ao TSE
// uma proposta de governo (Lei 9.504/97, art. 11, §1º, IX). O TSE publica os PDFs em
// pacotes por UF; os das municipais passam de 1 GB, por isso lemos só o PDF pedido.
const (
	urlPropostasTSE = "https://cdn.tse.jus.br/estatistica/sead/odsele/proposta_governo/proposta_governo_%s_%s.zip"
	maxPDF          = 60 << 20
	ttlPropostas    = 24 * time.Hour
)

var anosPropostas = map[string]bool{"2024": true, "2026": true}

type ArquivoProposta struct {
	Nome    string `json:"nome"`
	URL     string `json:"url"`
	Tamanho int64  `json:"tamanho"`
}

// AnaliseProposta resume o PDF sem IA: temas mais citados, compromissos transcritos do
// texto e os mesmos compromissos arrumados por tema.
type AnaliseProposta struct {
	Arquivos        []ArquivoProposta  `json:"arquivos"`
	TextoDisponivel bool               `json:"textoDisponivel"`  // false para PDFs digitalizados sem OCR
	Origem          string             `json:"origem,omitempty"` // "texto" (PDF com texto) ou "ocr" (lido da imagem)
	OCRDisponivel   bool               `json:"ocrDisponivel"`    // Tesseract e Poppler instalados no servidor
	Processando     bool               `json:"processando,omitempty"`
	Palavras        int                `json:"palavras"`
	Temas           []dominio.Contagem `json:"temas"`
	Compromissos    []string           `json:"compromissos"`
	ResumoPorTema   []TemaResumo       `json:"resumoPorTema"`
}

const (
	validadeAnalise     = 7 * 24 * time.Hour
	compromissosPorTema = 5
)

// Temas e palavras-chave (sem acentos, maiúsculas). Conta-se quantas vezes aparecem.
var temasPropostas = []struct {
	tema     string
	palavras []string
}{
	{"Saúde", []string{"SAUDE", "HOSPITA", "SUS", "MEDIC", "UPA", "CIRURGIA", "VACINA"}},
	{"Educação", []string{"EDUCACAO", "ESCOLA", "ENSINO", "PROFESSOR", "CRECHE", "ALUNO", "UNIVERSIDADE"}},
	{"Segurança pública", []string{"SEGURANCA PUBLICA", "POLICIA", "CRIME", "CRIMINALIDADE", "VIOLENCIA", "PRESIDIO", "FACCAO"}},
	{"Economia e emprego", []string{"EMPREGO", "ECONOMIA", "EMPRESA", "EMPREENDED", "INDUSTRIA", "RENDA", "INVESTIMENTO PRIVADO"}},
	{"Infraestrutura e mobilidade", []string{"RODOVIA", "ESTRADA", "INFRAESTRUTURA", "MOBILIDADE", "TRANSPORTE", "PORTO", "AEROPORTO", "PAVIMENTA"}},
	{"Meio ambiente e clima", []string{"MEIO AMBIENTE", "AMBIENTAL", "SUSTENTAB", "CLIMA", "ENCHENTE", "DESMATAMENTO", "DESASTRE"}},
	{"Assistência social", []string{"ASSISTENCIA SOCIAL", "POBREZA", "FOME", "VULNERAB", "TRANSFERENCIA DE RENDA"}},
	{"Moradia", []string{"HABITACAO", "MORADIA", "HABITACIONAL", "CASA PROPRIA"}},
	{"Agricultura", []string{"AGRICULT", "AGRONEGOCIO", "RURAL", "PRODUTOR", "AGROPECUAR"}},
	{"Cultura, esporte e turismo", []string{"CULTURA", "ESPORTE", "LAZER", "TURISMO"}},
	{"Gestão e contas públicas", []string{"IMPOSTO", "TRIBUT", "GESTAO PUBLICA", "TRANSPARENCIA", "CORRUPCAO", "FISCAL", "PRIVATIZ", "CONCESS"}},
	{"Direitos e igualdade", []string{"MULHER", "IGUALDADE", "RACISMO", "LGBT", "INDIGENA", "DIREITOS HUMANOS", "PESSOA COM DEFICIENCIA", "IDOSO"}},
	{"Tecnologia e inovação", []string{"INOVACAO", "TECNOLOGIA", "DIGITAL", "INTERNET", "CIENCIA", "INTELIGENCIA ARTIFICIAL"}},
	{"Saneamento e água", []string{"SANEAMENTO", "ESGOTO", "AGUA POTAVEL", "ABASTECIMENTO DE AGUA"}},
}

var (
	PadraoArquivo = regexp.MustCompile(`^[A-Z]{2}/[0-9]{4}[A-Z]{2}[0-9]{1,15}_[0-9]{2}$`)
)

type Propostas struct {
	mu       sync.Mutex
	pacotes  map[string]*tse.PacoteFotos
	analises map[string]*entradaAnalise
}

type entradaAnalise struct {
	pronto  chan struct{}
	valor   *AnaliseProposta
	err     error
	expira  time.Time
	parcial atomic.Pointer[AnaliseProposta] // o que já se sabe enquanto o OCR corre
}

func Novo() *Propostas {
	return &Propostas{pacotes: map[string]*tse.PacoteFotos{}, analises: map[string]*entradaAnalise{}}
}

func (p *Propostas) Pacote(ctx context.Context, ano, uf string) (*zipremoto.Zip, error) {
	chave := ano + "/" + uf
	p.mu.Lock()
	pc, ok := p.pacotes[chave]
	if !ok || (carga.Fechado(pc.Pronto) && time.Now().After(pc.Expira)) {
		pc = &tse.PacoteFotos{Pronto: make(chan struct{})}
		p.pacotes[chave] = pc
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			pc.Zip, pc.Err = zipremoto.Abrir(ctx, fmt.Sprintf(urlPropostasTSE, ano, uf))
			pc.Expira = time.Now().Add(ttlPropostas)
			if pc.Err != nil {
				log.Printf("propostas %s: %v", chave, pc.Err)
				pc.Expira = time.Now().Add(carga.TTLErro)
			}
			close(pc.Pronto)
		}()
	}
	p.mu.Unlock()
	select {
	case <-pc.Pronto:
		return pc.Zip, pc.Err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// arquivosDe lista as partes do PDF de uma candidatura (…_01, …_02), em ordem.
func arquivosDe(z *zipremoto.Zip, ano, uf, sq string) []string {
	prefixo := fmt.Sprintf("%s/%s%s%s_", uf, ano, uf, sq)
	var nomes []string
	for nome := range z.Arquivos {
		if strings.HasPrefix(nome, prefixo) {
			nomes = append(nomes, nome)
		}
	}
	sort.Strings(nomes)
	return nomes
}

// Analisar devolve a análise (da memória, do disco ou calculada agora). Se demorar mais do
// que o ctx permite (OCR), devolve a versão parcial com Processando = true.
func (p *Propostas) Analisar(ctx context.Context, ano, uf, sq string) (*AnaliseProposta, error) {
	chave := ano + "/" + uf + "/" + sq
	p.mu.Lock()
	e, ok := p.analises[chave]
	if !ok || (carga.Fechado(e.pronto) && time.Now().After(e.expira)) {
		e = &entradaAnalise{pronto: make(chan struct{})}
		p.analises[chave] = e
		go func() {
			nomeCache := fmt.Sprintf("proposta_%s_%s_%s.json", ano, uf, sq)
			// Uma análise sem texto em cache é refeita se entretanto o OCR passou a estar disponível.
			if a, ok := cache.Ler[AnaliseProposta](nomeCache, validadeAnalise); ok && (a.TextoDisponivel || !ocr.Disponivel()) {
				a.OCRDisponivel = ocr.Disponivel()
				e.valor, e.expira = a, time.Now().Add(ttlPropostas)
				close(e.pronto)
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
			defer cancel()
			e.valor, e.err = p.fazerAnalise(ctx, ano, uf, sq, func(a *AnaliseProposta) { e.parcial.Store(a) })
			e.expira = time.Now().Add(ttlPropostas)
			if e.err != nil {
				log.Printf("proposta %s: %v", chave, e.err)
				e.expira = time.Now().Add(carga.TTLErro)
			} else if len(e.valor.Arquivos) > 0 {
				if err := cache.Gravar(nomeCache, e.valor); err != nil {
					log.Printf("proposta %s: não foi possível gravar a cache: %v", chave, err)
				}
			}
			close(e.pronto)
		}()
	}
	p.mu.Unlock()
	select {
	case <-e.pronto:
		return e.valor, e.err
	case <-ctx.Done():
		if parcial := e.parcial.Load(); parcial != nil {
			return parcial, nil
		}
		return nil, ctx.Err()
	}
}

func (p *Propostas) fazerAnalise(ctx context.Context, ano, uf, sq string, aoParcial func(*AnaliseProposta)) (*AnaliseProposta, error) {
	z, err := p.Pacote(ctx, ano, uf)
	if err != nil {
		return nil, err
	}
	a := &AnaliseProposta{
		Arquivos: []ArquivoProposta{}, Temas: []dominio.Contagem{}, Compromissos: []string{}, ResumoPorTema: []TemaResumo{},
		OCRDisponivel: ocr.Disponivel(),
	}
	var pdfs [][]byte
	var texto strings.Builder
	for i, nome := range arquivosDe(z, ano, uf, sq) {
		f := z.Arquivos[nome]
		a.Arquivos = append(a.Arquivos, ArquivoProposta{
			Nome:    fmt.Sprintf("Parte %d", i+1),
			URL:     fmt.Sprintf("/api/propostas/arquivo/%s/%s/%s", ano, uf, strings.TrimPrefix(nome, uf+"/")),
			Tamanho: int64(f.UncompressedSize64),
		})
		if f.UncompressedSize64 > maxPDF {
			continue
		}
		conteudo, err := lerInteiro(ctx, z, nome)
		if err != nil {
			return nil, err
		}
		pdfs = append(pdfs, conteudo)
		texto.WriteString(textoDoPDF(conteudo))
		texto.WriteString("\n")
	}
	if len(a.Arquivos) == 0 {
		return a, nil
	}

	bruto := texto.String()
	a.Origem = "texto"
	if len(strings.Fields(bruto)) < 150 {
		a.Origem = ""
		if !a.OCRDisponivel {
			return a, nil
		}
		// PDF digitalizado: avisa que está a processar e lê as imagens com OCR.
		parcial := *a
		parcial.Processando = true
		aoParcial(&parcial)
		var lido strings.Builder
		for _, conteudo := range pdfs {
			t, err := ocr.TextoDoPDF(ctx, conteudo)
			if err != nil {
				return nil, err
			}
			lido.WriteString(t)
		}
		bruto = lido.String()
		a.Origem = "ocr"
	}

	a.Palavras = len(strings.Fields(bruto))
	a.TextoDisponivel = a.Palavras >= 150
	if !a.TextoDisponivel {
		a.Origem = ""
		return a, nil
	}
	a.Temas = contarTemas(bruto)
	frases := extrairFrases(bruto)
	a.Compromissos = maisConcretos(frases, 12)
	a.ResumoPorTema = resumirPorTema(frases, compromissosPorTema)
	return a, nil
}

func lerInteiro(ctx context.Context, z *zipremoto.Zip, nome string) ([]byte, error) {
	fluxo, err := z.AbrirFluxo(ctx, nome)
	if err != nil {
		return nil, err
	}
	defer fluxo.Close()
	return io.ReadAll(io.LimitReader(fluxo, maxPDF))
}

// textoDoPDF extrai o texto por linhas. PDFs digitalizados (só imagem) devolvem "".
// A biblioteca entra em pânico com alguns PDFs malformados; nesse caso fica o que já leu.
func textoDoPDF(conteudo []byte) (texto string) {
	var b strings.Builder
	defer func() {
		if recover() != nil {
			texto = b.String()
		}
	}()
	r, err := pdf.NewReader(bytes.NewReader(conteudo), int64(len(conteudo)))
	if err != nil {
		return ""
	}
	for i := 1; i <= r.NumPage(); i++ {
		pg := r.Page(i)
		if pg.V.IsNull() {
			continue
		}
		linhas, err := pg.GetTextByRow()
		if err != nil {
			continue
		}
		for _, l := range linhas {
			for _, t := range l.Content {
				b.WriteString(t.S)
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

var palavrasDeLigacao = map[string]bool{
	"de": true, "da": true, "do": true, "das": true, "dos": true, "e": true, "a": true, "o": true, "as": true, "os": true,
	"para": true, "com": true, "em": true, "no": true, "na": true, "nos": true, "nas": true, "por": true, "ao": true, "à": true, "que": true,
}

// terminaCortada deteta frases partidas pela quebra de página/linha ("…concessões do estado de").
func terminaCortada(frase string) bool {
	palavras := strings.Fields(frase)
	return len(palavras) > 0 && palavrasDeLigacao[strings.ToLower(palavras[len(palavras)-1])]
}

// maiusculas identifica títulos escritos todos em maiúsculas.
func maiusculas(s string) bool {
	letras, altas := 0, 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			letras++
			if unicode.IsUpper(r) {
				altas++
			}
		}
	}
	return letras > 0 && altas*10 > letras*7
}

func ValidarAnoUF(ano, uf string) bool {
	return anosPropostas[ano] && (uf == "BR" || dominio.UFsValidas[uf])
}

// EmProcessamento é a resposta enquanto um PDF digitalizado ainda passa por OCR.
func EmProcessamento() *AnaliseProposta {
	return &AnaliseProposta{Processando: true, Arquivos: []ArquivoProposta{}, OCRDisponivel: ocr.Disponivel()}
}
