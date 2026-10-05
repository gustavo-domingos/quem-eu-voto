package senado

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/rede"
	"quemeuvoto/internal/plataforma/texto"
)

const apiSenado = "https://legis.senado.leg.br/dadosabertos"

// Senador em exercício, segundo a API de dados abertos do Senado.
type Senador struct {
	Codigo         int
	Nome           string
	NomeCompleto   string
	Partido        string
	UF             string
	Foto           string
	Email          string
	Sexo           string
	MandatoInicio  string // AAAA-MM-DD
	MandatoFim     string
	Suplente       bool              // suplente em exercício (o titular está afastado)
	ExercicioDesde string            // AAAA-MM-DD do início do exercício atual
	Exercicios     []dominio.Periodo // todos os períodos em exercício (sem os afastamentos, ex.: para ser ministro)
	Suplentes      []dominio.Vice
}

// EstatSenado é calculado por senador e fica em cache como os dados dos deputados.
type EstatSenado struct {
	DataNascimento        string
	Votacoes              int // votações nominais em que o senador estava em exercício
	Presente              int
	AusenciasJustificadas int // missão, atividade parlamentar, licenças
	NaoCompareceu         int
	Projetos              int // PL, PLP e PEC de autoria ou coautoria desde inicioLegislatura
	VotosRecentes         []dominio.VotoRecente
	ProjetosRecentes      []dominio.Projeto
}

func (e EstatSenado) Participacao() *float64 {
	if e.Votacoes == 0 {
		return nil
	}
	pct := float64(e.Presente) / float64(e.Votacoes) * 100
	return &pct
}

// umOuVarios lê campos que a API do Senado devolve como objeto quando há só um elemento.
type umOuVarios[T any] []T

func (u *umOuVarios[T]) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) > 0 && b[0] == '[' {
		var lista []T
		err := json.Unmarshal(b, &lista)
		*u = lista
		return err
	}
	if string(b) == "null" {
		return nil
	}
	var um T
	if err := json.Unmarshal(b, &um); err != nil {
		return err
	}
	*u = []T{um}
	return nil
}

func getSenado(ctx context.Context, caminho string, destino any) error {
	resp, err := rede.Get(ctx, apiSenado+caminho, "application/json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(destino)
}

func ListarSenadores(ctx context.Context) ([]Senador, error) {
	var res struct {
		ListaParlamentarEmExercicio struct {
			Parlamentares struct {
				Parlamentar umOuVarios[struct {
					IdentificacaoParlamentar struct {
						CodigoParlamentar       string
						NomeParlamentar         string
						NomeCompletoParlamentar string
						SexoParlamentar         string
						UrlFotoParlamentar      string
						EmailParlamentar        string
						SiglaPartidoParlamentar string
						UfParlamentar           string
					}
					Mandato struct {
						DescricaoParticipacao        string
						PrimeiraLegislaturaDoMandato struct{ DataInicio string }
						SegundaLegislaturaDoMandato  struct{ DataFim string }
						Suplentes                    struct {
							Suplente umOuVarios[struct{ NomeParlamentar string }]
						}
						Exercicios struct {
							Exercicio umOuVarios[struct{ DataInicio, DataFim string }]
						}
					}
				}]
			}
		}
	}
	if err := getSenado(ctx, "/senador/lista/atual", &res); err != nil {
		return nil, err
	}

	var senadores []Senador
	for _, p := range res.ListaParlamentarEmExercicio.Parlamentares.Parlamentar {
		id := p.IdentificacaoParlamentar
		codigo, err := strconv.Atoi(id.CodigoParlamentar)
		if err != nil {
			continue
		}
		s := Senador{
			Codigo:        codigo,
			Nome:          id.NomeParlamentar,
			NomeCompleto:  id.NomeCompletoParlamentar,
			Partido:       id.SiglaPartidoParlamentar,
			UF:            id.UfParlamentar,
			Foto:          strings.Replace(id.UrlFotoParlamentar, "http://", "https://", 1),
			Email:         id.EmailParlamentar,
			Sexo:          id.SexoParlamentar,
			MandatoInicio: p.Mandato.PrimeiraLegislaturaDoMandato.DataInicio,
			MandatoFim:    p.Mandato.SegundaLegislaturaDoMandato.DataFim,
			// A API traz "1º Suplente" com o "º" corrompido; basta saber se não é titular.
			Suplente: !strings.HasPrefix(p.Mandato.DescricaoParticipacao, "Titular"),
		}
		for _, ex := range p.Mandato.Exercicios.Exercicio {
			s.Exercicios = append(s.Exercicios, dominio.Periodo{Inicio: ex.DataInicio, Fim: ex.DataFim})
			if ex.DataFim == "" && ex.DataInicio > s.ExercicioDesde {
				s.ExercicioDesde = ex.DataInicio
			}
		}
		if s.ExercicioDesde == "" {
			s.ExercicioDesde = s.MandatoInicio
		}
		if len(s.Exercicios) == 0 {
			s.Exercicios = []dominio.Periodo{{Inicio: s.MandatoInicio}}
		}
		for i, sup := range p.Mandato.Suplentes.Suplente {
			s.Suplentes = append(s.Suplentes, dominio.Vice{Cargo: fmt.Sprintf("%dº Suplente", i+1), NomeUrna: sup.NomeParlamentar})
		}
		senadores = append(senadores, s)
	}
	if len(senadores) == 0 {
		return nil, fmt.Errorf("API do Senado sem senadores em exercício")
	}
	return senadores, nil
}

// EstatisticasSenador junta data de nascimento, participação nas votações nominais e
// projetos apresentados desde o início da legislatura.
func EstatisticasSenador(ctx context.Context, codigo int) (EstatSenado, error) {
	var e EstatSenado

	var det struct {
		DetalheParlamentar struct {
			Parlamentar struct {
				DadosBasicosParlamentar struct{ DataNascimento string }
			}
		}
	}
	if err := getSenado(ctx, fmt.Sprintf("/senador/%d", codigo), &det); err != nil {
		return e, err
	}
	e.DataNascimento = det.DetalheParlamentar.Parlamentar.DadosBasicosParlamentar.DataNascimento

	var vot struct {
		VotacaoParlamentar struct {
			Parlamentar struct {
				Votacoes struct {
					Votacao umOuVarios[struct {
						SiglaDescricaoVoto string
						DescricaoVotacao   string
						SessaoPlenaria     struct{ DataSessao string }
						Materia            struct{ DescricaoIdentificacao, Ementa string }
					}]
				}
			}
		}
	}
	inicio := strings.ReplaceAll(dominio.InicioLegislatura, "-", "")
	fim := time.Now().Format("20060102")
	if err := getSenado(ctx, fmt.Sprintf("/senador/%d/votacoes?dataInicio=%s&dataFim=%s", codigo, inicio, fim), &vot); err != nil {
		return e, err
	}
	for _, v := range vot.VotacaoParlamentar.Parlamentar.Votacoes.Votacao {
		e.Votacoes++
		switch voto := v.SiglaDescricaoVoto; {
		case voto == "NCom":
			e.NaoCompareceu++
		case voto == "MIS" || voto == "AP" || (strings.HasPrefix(voto, "L") && len(voto) <= 4):
			e.AusenciasJustificadas++
		default: // Sim, Não, Abstenção, Obstrução, voto secreto ("Votou"), presente sem voto (P-NRV)...
			e.Presente++
		}
		e.VotosRecentes = append(e.VotosRecentes, dominio.VotoRecente{
			Data: v.SessaoPlenaria.DataSessao, Proposicao: v.Materia.DescricaoIdentificacao,
			Ementa: texto.ResumirTexto(v.Materia.Ementa, 300), Descricao: texto.ResumirTexto(v.DescricaoVotacao, 220),
			Voto: v.SiglaDescricaoVoto,
		})
	}
	sort.SliceStable(e.VotosRecentes, func(i, j int) bool { return e.VotosRecentes[i].Data > e.VotosRecentes[j].Data })
	if len(e.VotosRecentes) > dominio.MaxVotosRecentes {
		e.VotosRecentes = e.VotosRecentes[:dominio.MaxVotosRecentes]
	}

	var aut struct {
		MateriasAutoriaParlamentar struct {
			Parlamentar struct {
				Autorias struct {
					Autoria umOuVarios[struct {
						Materia struct{ Codigo, Sigla, Ano, Data, DescricaoIdentificacao, Ementa string }
					}]
				}
			}
		}
	}
	if err := getSenado(ctx, fmt.Sprintf("/senador/%d/autorias", codigo), &aut); err != nil {
		return e, err
	}
	for _, a := range aut.MateriasAutoriaParlamentar.Parlamentar.Autorias.Autoria {
		ano, _ := strconv.Atoi(a.Materia.Ano)
		if ano >= dominio.AnoInicioLeg && (a.Materia.Sigla == "PL" || a.Materia.Sigla == "PLP" || a.Materia.Sigla == "PEC") {
			e.Projetos++
			e.ProjetosRecentes = append(e.ProjetosRecentes, dominio.Projeto{
				Titulo: a.Materia.DescricaoIdentificacao, Ementa: texto.ResumirTexto(a.Materia.Ementa, 300), Data: a.Materia.Data,
				URL: "https://www25.senado.leg.br/web/atividade/materias/-/materia/" + a.Materia.Codigo,
			})
		}
	}
	sort.SliceStable(e.ProjetosRecentes, func(i, j int) bool { return e.ProjetosRecentes[i].Data > e.ProjetosRecentes[j].Data })
	if len(e.ProjetosRecentes) > 10 {
		e.ProjetosRecentes = e.ProjetosRecentes[:10]
	}
	return e, nil
}
