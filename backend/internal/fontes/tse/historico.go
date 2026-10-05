package tse

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/planilha"
	"quemeuvoto/internal/plataforma/texto"
)

// CandidaturaPassada é uma candidatura numa eleição anterior.
type CandidaturaPassada struct {
	Ano        int      `json:"ano"`
	Cargo      string   `json:"cargo"`
	Local      string   `json:"local"`
	Partido    string   `json:"partido"`
	Numero     string   `json:"numero"`
	Resultado  string   `json:"resultado"`
	Eleito     bool     `json:"eleito"`
	Votos      *int64   `json:"votos"`
	Percentual *float64 `json:"percentual"`

	Sq    string `json:"-"`
	turno int
}

// IndiceHistorico guarda as candidaturas gerais de 2018 e 2022 por CPF e por nome + nascimento.
type IndiceHistorico struct {
	porCPF      map[string][]*CandidaturaPassada
	porNomeNasc map[string][]*CandidaturaPassada
}

func (h *IndiceHistorico) Buscar(cpf, nomeNasc string) []*CandidaturaPassada {
	if len(cpf) == 11 {
		if l := h.porCPF[cpf]; len(l) > 0 {
			return l
		}
	}
	return h.porNomeNasc[nomeNasc]
}

var resultadosTSE = map[string]string{
	"ELEITO":           "Eleito",
	"ELEITO POR QP":    "Eleito por quociente partidário",
	"ELEITO POR MÉDIA": "Eleito por média",
	"SUPLENTE":         "Suplente",
	"NÃO ELEITO":       "Não eleito",
	"2º TURNO":         "Foi ao 2º turno",
}

// Anos gerais anteriores com consulta_cand pequena (~5 MB). As municipais de 2020 têm
// mais de meio milhão de registos e ficam de fora.
var anosHistorico = []int{2018, 2022}

func CarregarHistorico(ctx context.Context) (*IndiceHistorico, error) {
	h := &IndiceHistorico{porCPF: map[string][]*CandidaturaPassada{}, porNomeNasc: map[string][]*CandidaturaPassada{}}
	for _, ano := range anosHistorico {
		var votos *dominio.ResultadoVotacao
		if ano == dominio.AnoGeral2022 {
			v, err := carregarVotos(ctx, ano, false, validadeVotos2022)
			if err != nil {
				log.Printf("histórico %d: sem votos: %v", ano, err)
			}
			votos = v
		}

		porSQ := map[string]*CandidaturaPassada{}
		err := planilha.LerDoZip(ctx, fmt.Sprintf(urlCandidatosTSE, ano), fmt.Sprintf(csvCandidatosTSE, ano), func(cab map[string]int, l []string) {
			sq := planilha.Campo(cab, l, "SQ_CANDIDATO")
			turno, _ := strconv.Atoi(planilha.Campo(cab, l, "NR_TURNO"))
			sit := planilha.Campo(cab, l, "DS_SIT_TOT_TURNO")
			resultado, ok := resultadosTSE[sit]
			if !ok {
				resultado = "Candidatura não apreciada ou inapta"
			}

			if c, existe := porSQ[sq]; existe {
				if turno > c.turno { // o resultado final é o do último turno
					c.turno, c.Resultado, c.Eleito = turno, resultado, strings.HasPrefix(sit, "ELEITO")
				}
				return
			}
			local := planilha.Campo(cab, l, "SG_UF")
			if local == "BR" {
				local = "Brasil"
			}
			c := &CandidaturaPassada{
				Ano: ano, Cargo: texto.TituloCargo(planilha.Campo(cab, l, "DS_CARGO")), Local: local,
				Partido: planilha.Campo(cab, l, "SG_PARTIDO"), Numero: planilha.Campo(cab, l, "NR_CANDIDATO"),
				Resultado: resultado, Eleito: strings.HasPrefix(sit, "ELEITO"), Sq: sq, turno: turno,
			}
			if votos != nil {
				if v := votos.PorCandidato[sq]; v != nil {
					c.Votos, c.Percentual = &v.Votos, &v.Percentual
				}
			}
			porSQ[sq] = c
			if cpf := planilha.Campo(cab, l, "NR_CPF_CANDIDATO"); len(cpf) == 11 {
				h.porCPF[cpf] = append(h.porCPF[cpf], c)
			}
			chave := dominio.ChaveNomeNasc(planilha.Campo(cab, l, "NM_CANDIDATO"), texto.DataISO(planilha.Campo(cab, l, "DT_NASCIMENTO")))
			h.porNomeNasc[chave] = append(h.porNomeNasc[chave], c)
		})
		if err != nil {
			return nil, fmt.Errorf("candidatos %d: %w", ano, err)
		}
		log.Printf("histórico %d: %d candidaturas", ano, len(porSQ))
	}
	return h, nil
}
