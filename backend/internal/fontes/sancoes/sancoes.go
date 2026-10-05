package sancoes

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/planilha"
	"quemeuvoto/internal/plataforma/texto"
)

const (
	urlCGU             = "https://portaldatransparencia.gov.br/download-de-dados/%s/%s"
	urlMotivosTSE      = "https://cdn.tse.jus.br/estatistica/sead/odsele/motivo_cassacao/motivo_cassacao_%d.zip"
	csvMotivosTSE      = "motivo_cassacao_%d_BRASIL.csv"
	IntervaloSancoes   = 24 * time.Hour
	linkCNJImprobidade = "https://www.cnj.jus.br/improbidade_adm/consultar_requerido.php"
)

// RegistroJustica é uma decisão oficial (sanção, condenação, decisão da Justiça Eleitoral)
// encontrada para a pessoa. Não incluímos investigações nem processos sem decisão.
type RegistroJustica struct {
	Fonte     string `json:"fonte"`
	Titulo    string `json:"titulo"`
	Detalhe   string `json:"detalhe,omitempty"`
	Orgao     string `json:"orgao,omitempty"`
	Processo  string `json:"processo,omitempty"`
	Inicio    string `json:"inicio,omitempty"`   // AAAA-MM-DD
	Fim       string `json:"fim,omitempty"`      // AAAA-MM-DD; vazio = sem prazo
	Transito  string `json:"transito,omitempty"` // trânsito em julgado
	Vigente   *bool  `json:"vigente,omitempty"`  // a sanção ainda está em vigor?
	Corrupcao bool   `json:"corrupcao"`          // improbidade, compra de votos, abuso de poder, uso do cargo...
	Link      string `json:"link"`
}

type registroCEAF struct {
	digitos string // os 6 dígitos do meio do CPF, os únicos que a CGU publica no CEAF
	reg     RegistroJustica
}

// IndiceSancoes junta os cadastros da CGU (CEIS, CNEP, CEAF) e as decisões da Justiça
// Eleitoral sobre candidaturas (motivos de indeferimento e cassação).
type IndiceSancoes struct {
	porCPF      map[string][]RegistroJustica
	ceafPorNome map[string][]registroCEAF
	porSQ       map[string][]RegistroJustica
	DataCGU     string
}

// Buscar devolve os registros da pessoa. Os da CGU exigem o CPF: o CEIS/CNEP publicam-no
// inteiro; o CEAF só os 6 dígitos do meio, por isso aí exigimos nome igual E dígitos iguais
// (só o nome dá falsos positivos com homónimos). Os do TSE vêm pelo número da candidatura.
func (s *IndiceSancoes) Buscar(cpf, nomeCompleto string, sqs ...string) []RegistroJustica {
	var res []RegistroJustica
	if len(cpf) == 11 {
		res = append(res, s.porCPF[cpf]...)
		for _, r := range s.ceafPorNome[texto.Normalizar(nomeCompleto)] {
			if r.digitos == cpf[3:9] {
				res = append(res, r.reg)
			}
		}
	}
	vistos := map[string]bool{}
	for _, sq := range sqs {
		if sq == "" || vistos[sq] {
			continue
		}
		vistos[sq] = true
		res = append(res, s.porSQ[sq]...)
	}
	return res
}

// Motivos da Justiça Eleitoral ligados a conduta (e não a papelada, como documentos em
// falta, quitação eleitoral ou problemas do partido).
func motivoRelevante(m string) (titulo string, corrupcao, ok bool) {
	n := texto.Normalizar(m)
	switch {
	// A inelegibilidade da LC 64/90 tem muitas causas (ex.: contas rejeitadas) e o abuso de
	// poder é uma infração eleitoral: são mostrados, mas não marcados como corrupção.
	case strings.Contains(n, "FICHA LIMPA"), strings.Contains(n, "INELEGIBILIDADE INFRACONSTITUCIONAL"):
		return "Inelegível pela LC 64/90 (que inclui a Lei da Ficha Limpa)", false, true
	case strings.Contains(n, "COMPRA DE VOTO"), strings.Contains(n, "CAPTACAO ILICITA DE SUFRAGIO"):
		return "Compra de votos (captação ilícita de sufrágio)", true, true
	case strings.Contains(n, "ABUSO DE PODER"):
		return texto.FraseCapitalizada(strings.TrimSpace(strings.Split(m, "(")[0])), false, true
	case strings.Contains(n, "GASTO ILICITO"), strings.Contains(n, "CAPTACAO OU GASTO ILICITO"):
		return "Captação ou gasto ilícito de recursos de campanha", true, true
	case strings.Contains(n, "CONDUTA VEDADA"):
		return "Conduta vedada a agente público em campanha", false, true
	case strings.Contains(n, "USO INDEVIDO DE MEIOS DE COMUNICACAO"):
		return "Uso indevido dos meios de comunicação", false, true
	case strings.Contains(n, "OUTRAS FRAUDES"):
		return "Fraude eleitoral", true, true
	}
	return "", false, false
}

func CarregarSancoes(ctx context.Context) (*IndiceSancoes, error) {
	s := &IndiceSancoes{porCPF: map[string][]RegistroJustica{}, ceafPorNome: map[string][]registroCEAF{}, porSQ: map[string][]RegistroJustica{}}

	// A CGU publica um ficheiro por dia; o de hoje pode ainda não existir.
	var erroCGU error
	for dias := 0; dias < 7; dias++ {
		data := time.Now().AddDate(0, 0, -dias).Format("20060102")
		if erroCGU = s.lerCGU(ctx, data); erroCGU == nil {
			s.DataCGU = data[:4] + "-" + data[4:6] + "-" + data[6:]
			break
		}
	}
	if erroCGU != nil {
		return nil, fmt.Errorf("cadastros da CGU: %w", erroCGU)
	}

	for _, ano := range []int{dominio.AnoGeral2022, dominio.AnoMunicipal, dominio.AnoEleicao} {
		err := planilha.LerDoZip(ctx, fmt.Sprintf(urlMotivosTSE, ano), fmt.Sprintf(csvMotivosTSE, ano), func(cab map[string]int, l []string) {
			titulo, corrupcao, ok := motivoRelevante(planilha.Campo(cab, l, "DS_MOTIVO"))
			if !ok {
				return
			}
			tipo := "Candidatura indeferida pela Justiça Eleitoral"
			if strings.Contains(planilha.Campo(cab, l, "DS_TP_MOTIVO"), "cassa") {
				tipo = "Cassação do registro ou diploma decidida pela Justiça Eleitoral"
			}
			sq := planilha.Campo(cab, l, "SQ_CANDIDATO")
			processo := texto.SemMarcador(planilha.Campo(cab, l, "NR_PROCESSO"))
			if strings.HasPrefix(processo, "-") {
				processo = ""
			}
			// Vários motivos do mesmo processo viram um só registo.
			for i, r := range s.porSQ[sq] {
				if r.Processo == processo && strings.HasPrefix(r.Detalhe, tipo) {
					s.porSQ[sq][i].Titulo += "; " + strings.ToLower(titulo[:1]) + titulo[1:]
					s.porSQ[sq][i].Corrupcao = r.Corrupcao || corrupcao
					return
				}
			}
			s.porSQ[sq] = append(s.porSQ[sq], RegistroJustica{
				Fonte: "Justiça Eleitoral (TSE)", Titulo: titulo, Corrupcao: corrupcao,
				Detalhe: fmt.Sprintf("%s na eleição de %d (%s). Pode ter havido recurso: confira a situação atual no TSE.",
					tipo, ano, planilha.Campo(cab, l, "DS_ELEICAO")),
				Processo: processo,
				Link:     "https://divulgacandcontas.tse.jus.br/",
			})
		})
		if err != nil {
			return nil, fmt.Errorf("motivos TSE %d: %w", ano, err)
		}
	}
	log.Printf("sanções: %d CPF na CGU (CEIS/CNEP), %d nomes no CEAF, %d candidaturas com decisão do TSE", len(s.porCPF), len(s.ceafPorNome), len(s.porSQ))
	return s, nil
}

func (s *IndiceSancoes) lerCGU(ctx context.Context, data string) error {
	for _, cadastro := range []string{"ceis", "cnep", "ceaf"} {
		zr, err := planilha.BaixarZip(ctx, fmt.Sprintf(urlCGU, cadastro, data))
		if err != nil {
			return err
		}
		for _, f := range zr.File {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			err = planilha.Ler(&planilha.LeitorLatin1{Origem: rc}, func(cab map[string]int, l []string) {
				if planilha.Campo(cab, l, "TIPO DE PESSOA") != "F" {
					return
				}
				r := registroCGU(cadastro, cab, l)
				doc := planilha.Campo(cab, l, "CPF OU CNPJ DO SANCIONADO")
				if cadastro == "ceaf" {
					digitos := texto.ApenasDigitos(doc)
					if len(digitos) == 6 {
						nome := texto.Normalizar(planilha.Campo(cab, l, "NOME DO SANCIONADO"))
						s.ceafPorNome[nome] = append(s.ceafPorNome[nome], registroCEAF{digitos: digitos, reg: r})
					}
					return
				}
				if cpf := texto.ApenasDigitos(doc); len(cpf) == 11 {
					s.porCPF[cpf] = append(s.porCPF[cpf], r)
				}
			})
			rc.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func registroCGU(cadastro string, cab map[string]int, l []string) RegistroJustica {
	categoria := planilha.Campo(cab, l, "CATEGORIA DA SANÇÃO")
	fundamento := planilha.Campo(cab, l, "FUNDAMENTAÇÃO LEGAL")
	origem := planilha.Campo(cab, l, "ORIGEM INFORMAÇÕES")
	fn := texto.Normalizar(fundamento)
	r := RegistroJustica{
		Orgao:    texto.SemMarcador(planilha.Campo(cab, l, "ÓRGÃO SANCIONADOR")),
		Processo: planilha.Campo(cab, l, "NÚMERO DO PROCESSO"),
		Inicio:   texto.DataISO(planilha.Campo(cab, l, "DATA INÍCIO SANÇÃO")),
		Fim:      texto.DataISO(planilha.Campo(cab, l, "DATA FINAL SANÇÃO")),
		Transito: texto.DataISO(planilha.Campo(cab, l, "DATA DO TRÂNSITO EM JULGADO")),
		Corrupcao: strings.Contains(fn, "IMPROBIDADE") || strings.Contains(fn, "8429") || strings.Contains(fn, "VALER-SE DO CARGO") ||
			strings.Contains(fn, "CORRUPCAO") || strings.Contains(fn, "PROPINA") || strings.Contains(fn, "12846") || strings.Contains(fn, "LESAO AOS COFRES"),
	}
	vigente := r.Fim == "" || r.Fim >= time.Now().Format("2006-01-02")
	r.Vigente = &vigente

	switch cadastro {
	case "ceis":
		r.Fonte = "CGU – Cadastro de Empresas e Pessoas Inidôneas e Suspensas (CEIS)"
		r.Link = "https://portaldatransparencia.gov.br/sancoes/ceis"
		if strings.Contains(fn, "8429") || strings.Contains(fn, "IMPROBIDADE") {
			r.Titulo = "Condenação por improbidade administrativa"
			if strings.Contains(texto.Normalizar(origem), "CONSELHO NACIONAL DE JUSTICA") {
				r.Link = linkCNJImprobidade
			}
		} else {
			r.Titulo = "Sanção administrativa: " + strings.ToLower(categoria)
		}
		r.Detalhe = "Sanção aplicada: " + strings.ToLower(categoria) + "."
	case "cnep":
		r.Fonte = "CGU – Cadastro Nacional de Empresas Punidas (CNEP, Lei Anticorrupção)"
		r.Link = "https://portaldatransparencia.gov.br/sancoes/cnep"
		r.Titulo = "Punição com base na Lei Anticorrupção"
		r.Corrupcao = true
		r.Detalhe = "Sanção aplicada: " + strings.ToLower(categoria) + "."
	case "ceaf":
		r.Fonte = "CGU – Cadastro de Expulsões da Administração Federal (CEAF)"
		r.Link = "https://portaldatransparencia.gov.br/sancoes/ceaf"
		r.Titulo = "Expulso(a) do serviço público federal (" + strings.ToLower(categoria) + ")"
		if lot := texto.SemMarcador(planilha.Campo(cab, l, "ÓRGÃO DE LOTAÇÃO")); lot != "" && lot != "Sem Informação" {
			r.Detalhe = "Órgão onde trabalhava: " + texto.TituloMunicipio(lot) + ". "
		}
		r.Detalhe += "Motivo: " + motivosCEAF(fundamento) + "."
	}
	return r
}

// motivosCEAF extrai a infração de cada artigo citado ("ART. 132, IV - IMPROBIDADE ADMINISTRATIVA").
func motivosCEAF(fundamento string) string {
	var motivos []string
	vistos := map[string]bool{}
	for _, parte := range strings.Split(fundamento, ";") {
		trechos := strings.Split(parte, " - ")
		if len(trechos) < 2 {
			continue
		}
		m := strings.TrimSpace(trechos[len(trechos)-1])
		m = strings.TrimRight(strings.Split(m, ":")[0], ".")
		if m == "" || vistos[m] {
			continue
		}
		vistos[m] = true
		motivos = append(motivos, strings.ToLower(texto.ResumirTexto(m, 120)))
	}
	if len(motivos) == 0 {
		return strings.ToLower(texto.ResumirTexto(fundamento, 160))
	}
	if len(motivos) > 3 {
		motivos = motivos[:3]
	}
	return strings.Join(motivos, "; ")
}
