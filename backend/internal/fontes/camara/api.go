package camara

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/rede"
	"quemeuvoto/internal/plataforma/texto"
)

const (
	apiCamara      = "https://dadosabertos.camara.leg.br/api/v2"
	arquivosCamara = "https://dadosabertos.camara.leg.br/arquivos"
)

// Tipos de proposição considerados "projetos": Projeto de Lei, Projeto de Lei
// Complementar e Proposta de Emenda à Constituição. Requerimentos, emendas,
// indicações etc. ficam de fora.
var tiposProjeto = []string{"PL", "PLP", "PEC"}

type DeputadoResumo struct {
	ID           int    `json:"id"`
	Nome         string `json:"nome"`
	SiglaPartido string `json:"siglaPartido"`
	SiglaUf      string `json:"siglaUf"`
	UrlFoto      string `json:"urlFoto"`
	Email        string `json:"email"`
}

// ListarDeputados devolve todos os deputados em exercício (a API devolve os 513 numa única página).
func ListarDeputados(ctx context.Context) ([]DeputadoResumo, error) {
	resp, err := rede.Get(ctx, apiCamara+"/deputados?ordem=ASC&ordenarPor=nome", "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Dados []DeputadoResumo `json:"dados"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Dados, nil
}

// DetalheDeputado traz os campos usados para cruzar o deputado com o registo de candidatura no TSE.
type DetalheDeputado struct {
	CPF            string   `json:"cpf"`
	NomeCivil      string   `json:"nomeCivil"`
	DataNascimento string   `json:"dataNascimento"` // AAAA-MM-DD
	Sexo           string   `json:"sexo"`
	Escolaridade   string   `json:"escolaridade"`
	RedeSocial     []string `json:"redeSocial"`
	UrlWebsite     string   `json:"urlWebsite"`
	MunicipioNasc  string   `json:"municipioNascimento"`
	UfNascimento   string   `json:"ufNascimento"`
	UltimoStatus   struct {
		Gabinete struct {
			Predio string `json:"predio"`
			Sala   string `json:"sala"`
		} `json:"gabinete"`
	} `json:"ultimoStatus"`
}

// Gabinete devolve "prédio/sala" (ex.: "4/206"), como aparece na lotação dos funcionários.
func (d DetalheDeputado) Gabinete() string {
	g := d.UltimoStatus.Gabinete
	if g.Predio == "" || g.Sala == "" {
		return ""
	}
	return g.Predio + "/" + g.Sala
}

// ListarProjetos devolve os PL, PLP e PEC mais recentes de que o deputado é autor ou coautor.
func ListarProjetos(ctx context.Context, deputadoID int) ([]dominio.Projeto, error) {
	anos := make([]string, 0, 4)
	for a := dominio.AnoInicioLeg; a <= time.Now().Year(); a++ {
		anos = append(anos, strconv.Itoa(a))
	}
	q := url.Values{}
	q.Set("idDeputadoAutor", strconv.Itoa(deputadoID))
	q.Set("siglaTipo", strings.Join(tiposProjeto, ","))
	q.Set("ano", strings.Join(anos, ","))
	q.Set("ordem", "DESC")
	q.Set("ordenarPor", "id")
	q.Set("itens", "10")

	resp, err := rede.Get(ctx, apiCamara+"/proposicoes?"+q.Encode(), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var res struct {
		Dados []struct {
			ID        int    `json:"id"`
			SiglaTipo string `json:"siglaTipo"`
			Numero    int    `json:"numero"`
			Ano       int    `json:"ano"`
			Ementa    string `json:"ementa"`
		} `json:"dados"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	projetos := []dominio.Projeto{}
	for _, p := range res.Dados {
		projetos = append(projetos, dominio.Projeto{
			Titulo: fmt.Sprintf("%s %d/%d", p.SiglaTipo, p.Numero, p.Ano),
			Ementa: texto.ResumirTexto(p.Ementa, 300),
			URL:    fmt.Sprintf("https://www.camara.leg.br/proposicoesWeb/fichadetramitacao?idProposicao=%d", p.ID),
		})
	}
	return projetos, nil
}

func DetalharDeputado(ctx context.Context, deputadoID int) (DetalheDeputado, error) {
	resp, err := rede.Get(ctx, fmt.Sprintf("%s/deputados/%d", apiCamara, deputadoID), "application/json")
	if err != nil {
		return DetalheDeputado{}, err
	}
	defer resp.Body.Close()

	var res struct {
		Dados DetalheDeputado `json:"dados"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return DetalheDeputado{}, err
	}
	return res.Dados, nil
}

// ContarProjetos devolve quantos PL/PLP/PEC o deputado (co)assinou na legislatura atual.
// Usa o cabeçalho X-Total-Count, por isso basta pedir 1 item.
func ContarProjetos(ctx context.Context, deputadoID int) (int, error) {
	anos := make([]string, 0, 4)
	for a := dominio.AnoInicioLeg; a <= time.Now().Year() && a < dominio.AnoInicioLeg+4; a++ {
		anos = append(anos, strconv.Itoa(a))
	}

	q := url.Values{}
	q.Set("idDeputadoAutor", strconv.Itoa(deputadoID))
	q.Set("siglaTipo", strings.Join(tiposProjeto, ","))
	q.Set("ano", strings.Join(anos, ","))
	q.Set("itens", "1")

	resp, err := rede.Get(ctx, apiCamara+"/proposicoes?"+q.Encode(), "application/json")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	total, err := strconv.Atoi(resp.Header.Get("X-Total-Count"))
	if err != nil {
		return 0, fmt.Errorf("X-Total-Count ausente ou inválido: %w", err)
	}
	return total, nil
}
