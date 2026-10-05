package tse

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/planilha"
	"quemeuvoto/internal/plataforma/texto"
)

const (
	urlCandidatosTSE   = "https://cdn.tse.jus.br/estatistica/sead/odsele/consulta_cand/consulta_cand_%d.zip"
	csvCandidatosTSE   = "consulta_cand_%d_BRASIL.csv"
	urlComplementarTSE = "https://cdn.tse.jus.br/estatistica/sead/odsele/consulta_cand_complementar/consulta_cand_complementar_%d.zip"
	csvComplementarTSE = "consulta_cand_complementar_%d_BRASIL.csv"
)

// Situações de totalização em que os votos no candidato são contabilizados
// (ainda que, nos casos com recurso, possam vir a ser anulados).
var situacoesAptas = map[string]bool{
	"DEFERIDO":             true,
	"DEFERIDO COM RECURSO": true,
	"INDEFERIDO EM PRAZO RECURSAL OU COM RECURSO": true,
	"PENDENTE DE JULGAMENTO":                      true,
}

// Candidatura é um registo de candidatura nas eleições gerais, segundo o TSE.
type Candidatura struct {
	ID        string         `json:"id"` // SQ_CANDIDATO
	Cargo     string         `json:"cargo"`
	Numero    string         `json:"numero"`
	NomeUrna  string         `json:"nomeUrna"`
	Nome      string         `json:"nome"`
	Partido   string         `json:"partido"`
	UF        string         `json:"uf"`
	Coligacao string         `json:"coligacao,omitempty"`
	Ocupacao  string         `json:"ocupacao"`
	Situacao  string         `json:"situacao"`
	Apto      bool           `json:"apto"`
	Foto      string         `json:"foto"`
	Vices     []dominio.Vice `json:"vices,omitempty"`
	// Só para mandatos municipais.
	Municipio  string `json:"municipio,omitempty"`
	Observacao string `json:"observacao,omitempty"`

	ChaveCargo     string  `json:"-"`
	ChaveNome      string  `json:"-"` // nome de urna normalizado, para ordenar
	ChaveBusca     string  `json:"-"` // nome de urna, nome, partido e município normalizados
	ChaveMunicipio string  `json:"-"`
	Ue             string  `json:"-"`
	Cpf            string  `json:"-"`
	NomeNasc       string  `json:"-"`
	NumeroPartido  string  `json:"-"`
	NomePartido    string  `json:"-"`
	Federacao      string  `json:"-"`
	Genero         string  `json:"-"` // "F" ou "M"
	Escolaridade   string  `json:"-"`
	Nascimento     string  `json:"-"` // AAAA-MM-DD
	EstadoCivil    string  `json:"-"`
	CorRaca        string  `json:"-"`
	UfNascimento   string  `json:"-"`
	MunicipioNasc  string  `json:"-"` // só 2026 (ficheiro complementar)
	Email          string  `json:"-"`
	LimiteGastos   float64 `json:"-"` // limite legal de gastos de campanha (só 2026)
}

// candidaturaDaLinha lê os campos comuns aos ficheiros consulta_cand de qualquer ano.
// fotoPacote é o pacote de fotos do TSE onde está a foto (UF, ou BR para Presidente).
func candidaturaDaLinha(ano int, fotoPacote string, cab map[string]int, l []string) *Candidatura {
	sq := planilha.Campo(cab, l, "SQ_CANDIDATO")
	federacao := planilha.Campo(cab, l, "NM_FEDERACAO")
	if strings.HasPrefix(federacao, "#") {
		federacao = ""
	}
	// Para partidos em federação sem coligação, NM_COLIGACAO vem só como "FEDERAÇÃO".
	coligacao := planilha.Campo(cab, l, "NM_COLIGACAO")
	if coligacao == "FEDERAÇÃO" {
		coligacao = federacao
	}
	if coligacao == "PARTIDO ISOLADO" || strings.HasPrefix(coligacao, "#") {
		coligacao = ""
	}
	c := &Candidatura{
		ID:            sq,
		Cargo:         texto.TituloCargo(planilha.Campo(cab, l, "DS_CARGO")),
		Numero:        planilha.Campo(cab, l, "NR_CANDIDATO"),
		NomeUrna:      planilha.Campo(cab, l, "NM_URNA_CANDIDATO"),
		Nome:          planilha.Campo(cab, l, "NM_CANDIDATO"),
		Partido:       planilha.Campo(cab, l, "SG_PARTIDO"),
		UF:            planilha.Campo(cab, l, "SG_UF"),
		Coligacao:     coligacao,
		Ocupacao:      texto.FraseCapitalizada(planilha.Campo(cab, l, "DS_OCUPACAO")),
		Foto:          caminhoFoto(ano, fotoPacote, sq),
		Ue:            planilha.Campo(cab, l, "SG_UE"),
		Cpf:           planilha.Campo(cab, l, "NR_CPF_CANDIDATO"),
		NomeNasc:      dominio.ChaveNomeNasc(planilha.Campo(cab, l, "NM_CANDIDATO"), texto.DataISO(planilha.Campo(cab, l, "DT_NASCIMENTO"))),
		NumeroPartido: planilha.Campo(cab, l, "NR_PARTIDO"),
		NomePartido:   planilha.Campo(cab, l, "NM_PARTIDO"),
		Federacao:     federacao,
		Genero:        dominio.GeneroCurto(planilha.Campo(cab, l, "DS_GENERO")),
		Escolaridade:  texto.FraseCapitalizada(planilha.Campo(cab, l, "DS_GRAU_INSTRUCAO")),
		Nascimento:    texto.DataISO(planilha.Campo(cab, l, "DT_NASCIMENTO")),
		EstadoCivil:   texto.FraseCapitalizada(planilha.Campo(cab, l, "DS_ESTADO_CIVIL")),
		CorRaca:       texto.FraseCapitalizada(planilha.Campo(cab, l, "DS_COR_RACA")),
		UfNascimento:  texto.SemMarcador(planilha.Campo(cab, l, "SG_UF_NASCIMENTO")),
		Email:         strings.ToLower(texto.SemMarcador(planilha.Campo(cab, l, "DS_EMAIL"))),
	}
	c.ChaveNome = texto.Normalizar(c.NomeUrna)
	c.ChaveBusca = texto.Normalizar(c.NomeUrna + " " + c.Nome + " " + c.Partido)
	return c
}

// IndiceCandidatos guarda os candidatos aos cargos principais e permite encontrar
// as candidaturas de um deputado pelo CPF ou, quando a Câmara não informa o CPF,
// pelo nome civil + data de nascimento.
type IndiceCandidatos struct {
	Lista        []*Candidatura
	PorSQ        map[string]*Candidatura // inclui vices e suplentes
	porCPF       map[string][]*Candidatura
	PorNomeNasc  map[string][]*Candidatura `json:"-"`
	AtualizadoEm time.Time
}

// BuscarPor procura pelo CPF e, se não houver (o TSE esconde o CPF em 2024), pela
// chave de nome completo + data de nascimento (ver dominio.ChaveNomeNasc). Se alguma
// candidatura estiver apta, devolve só as aptas, para não misturar registos substituídos
// ou renunciados.
func (idx *IndiceCandidatos) BuscarPor(cpf, nomeNasc string) []*Candidatura {
	var encontradas []*Candidatura
	if len(cpf) == 11 {
		encontradas = idx.porCPF[cpf]
	}
	if len(encontradas) == 0 {
		encontradas = idx.PorNomeNasc[nomeNasc]
	}

	var aptas []*Candidatura
	for _, c := range encontradas {
		if c.Apto {
			aptas = append(aptas, c)
		}
	}
	if len(aptas) > 0 {
		return aptas
	}
	return encontradas
}

func CarregarCandidatos(ctx context.Context) (*IndiceCandidatos, error) {
	// Situação de cada candidatura, do ficheiro complementar.
	type situacao struct {
		texto         string
		apto          bool
		municipioNasc string
		limite        float64
	}
	situacoes := map[string]situacao{}
	err := planilha.LerDoZip(ctx, fmt.Sprintf(urlComplementarTSE, dominio.AnoEleicao), fmt.Sprintf(csvComplementarTSE, dominio.AnoEleicao), func(cab map[string]int, l []string) {
		tot := planilha.Campo(cab, l, "DS_SITUACAO_CANDIDATO_TOT")
		naUrna := planilha.Campo(cab, l, "ST_CANDIDATO_INSERIDO_URNA") == "SIM"
		descricao := tot
		if strings.HasPrefix(descricao, "#") {
			descricao = planilha.Campo(cab, l, "DS_SITUACAO_JULGAMENTO")
		}
		if !naUrna && (strings.HasPrefix(descricao, "#") || descricao == "DEFERIDO") {
			descricao = "NÃO CONSTA NA URNA"
		}
		situacoes[planilha.Campo(cab, l, "SQ_CANDIDATO")] = situacao{
			texto: texto.FraseCapitalizada(descricao), apto: naUrna && situacoesAptas[tot],
			municipioNasc: texto.TituloMunicipio(texto.SemMarcador(planilha.Campo(cab, l, "NM_MUNICIPIO_NASCIMENTO"))),
			limite:        texto.ValorDecimal(planilha.Campo(cab, l, "VR_DESPESA_MAX_CAMPANHA")),
		}
	})
	if err != nil {
		return nil, fmt.Errorf("ficheiro complementar: %w", err)
	}

	idx := &IndiceCandidatos{
		porCPF:       map[string][]*Candidatura{},
		PorNomeNasc:  map[string][]*Candidatura{},
		PorSQ:        map[string]*Candidatura{},
		AtualizadoEm: time.Now(),
	}
	var vices []*Candidatura
	err = planilha.LerDoZip(ctx, fmt.Sprintf(urlCandidatosTSE, dominio.AnoEleicao), fmt.Sprintf(csvCandidatosTSE, dominio.AnoEleicao), func(cab map[string]int, l []string) {
		c := candidaturaDaLinha(dominio.AnoEleicao, planilha.Campo(cab, l, "SG_UE"), cab, l)
		c.Situacao = situacoes[c.ID].texto
		c.Apto = situacoes[c.ID].apto
		c.MunicipioNasc = situacoes[c.ID].municipioNasc
		c.LimiteGastos = situacoes[c.ID].limite
		idx.PorSQ[c.ID] = c

		// Vices e suplentes não entram na listagem, mas são indexados para que um
		// deputado candidato a vice (ex.: vice-governador) seja reconhecido como candidato.
		if _, ehVice := dominio.CargoDoTitular[c.Cargo]; ehVice {
			vices = append(vices, c)
		} else if chave, ok := dominio.CargosPrincipais[c.Cargo]; ok {
			c.ChaveCargo = chave
			idx.Lista = append(idx.Lista, c)
		} else {
			return
		}
		if len(c.Cpf) == 11 {
			idx.porCPF[c.Cpf] = append(idx.porCPF[c.Cpf], c)
		}
		idx.PorNomeNasc[c.NomeNasc] = append(idx.PorNomeNasc[c.NomeNasc], c)
	})
	if err != nil {
		return nil, fmt.Errorf("ficheiro de candidatos: %w", err)
	}
	if len(idx.Lista) == 0 {
		return nil, fmt.Errorf("ficheiro do TSE sem candidatos")
	}

	// Liga vices e suplentes aptos ao titular apto com o mesmo número.
	titulares := map[string]*Candidatura{}
	for _, c := range idx.Lista {
		if c.Apto {
			titulares[c.Ue+"|"+c.Cargo+"|"+c.Numero] = c
		}
	}
	for _, v := range vices {
		if t, ok := titulares[v.Ue+"|"+dominio.CargoDoTitular[v.Cargo]+"|"+v.Numero]; ok && v.Apto {
			t.Vices = append(t.Vices, dominio.Vice{Cargo: v.Cargo, NomeUrna: v.NomeUrna, Partido: v.Partido})
		}
	}
	for _, c := range idx.Lista {
		// "1º Suplente" antes de "2º Suplente".
		sort.Slice(c.Vices, func(i, j int) bool { return c.Vices[i].Cargo < c.Vices[j].Cargo })
	}

	log.Printf("candidatos TSE: %d candidatos aos cargos principais", len(idx.Lista))
	return idx, nil
}
