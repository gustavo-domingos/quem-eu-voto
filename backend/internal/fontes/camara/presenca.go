package camara

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/planilha"
)

// Presenca resume a assiduidade de um deputado nas sessões deliberativas do Plenário.
type Presenca struct {
	Presentes int
	Sessoes   int
	Desde     string // AAAA-MM-DD da primeira presença na legislatura
}

func (p Presenca) Percentual() float64 {
	if p.Sessoes == 0 {
		return 0
	}
	return float64(p.Presentes) / float64(p.Sessoes) * 100
}

// IndicePresenca é calculado a partir dos ficheiros anuais de dados abertos da Câmara:
//   - eventos-{ano}.csv: identifica as Sessões Deliberativas do Plenário já encerradas;
//   - eventosPresencaDeputados-{ano}.csv: quem registou presença em cada evento.
//
// O denominador de cada deputado é o número de sessões (com registo de presença)
// desde a sua primeira presença na legislatura. Assim, suplentes que assumiram a
// meio do mandato não são penalizados pelas sessões anteriores à posse.
// Limitação: licenças e faltas justificadas não constam dos ficheiros e contam como ausência.
type IndicePresenca struct {
	PorDeputado  map[int]Presenca
	TotalSessoes int
	AtualizadoEm time.Time
}

type sessao struct {
	data      string // AAAA-MM-DD, ordenável como texto
	presentes map[int]struct{}
}

func CarregarPresenca(ctx context.Context) (*IndicePresenca, error) {
	sessoes := map[string]*sessao{}

	for ano := dominio.AnoInicioLeg; ano <= time.Now().Year(); ano++ {
		if err := lerSessoesPlenario(ctx, ano, sessoes); err != nil {
			return nil, fmt.Errorf("eventos %d: %w", ano, err)
		}
		if err := lerPresencas(ctx, ano, sessoes); err != nil {
			return nil, fmt.Errorf("presenças %d: %w", ano, err)
		}
	}

	// Apenas sessões com registo de presença entram na conta.
	var datas []string
	primeira := map[int]string{}
	presentes := map[int]int{}
	for _, s := range sessoes {
		if len(s.presentes) == 0 {
			continue
		}
		datas = append(datas, s.data)
		for id := range s.presentes {
			presentes[id]++
			if d, ok := primeira[id]; !ok || s.data < d {
				primeira[id] = s.data
			}
		}
	}
	sort.Strings(datas)

	idx := &IndicePresenca{
		PorDeputado:  make(map[int]Presenca, len(presentes)),
		TotalSessoes: len(datas),
		AtualizadoEm: time.Now(),
	}
	for id, n := range presentes {
		desde := sort.SearchStrings(datas, primeira[id])
		idx.PorDeputado[id] = Presenca{Presentes: n, Sessoes: len(datas) - desde, Desde: primeira[id]}
	}
	log.Printf("presença: %d sessões, %d deputados", idx.TotalSessoes, len(idx.PorDeputado))
	return idx, nil
}

func lerSessoesPlenario(ctx context.Context, ano int, sessoes map[string]*sessao) error {
	return abrirCSV(ctx, fmt.Sprintf("eventos/csv/eventos-%d.csv", ano), func(cab map[string]int, l []string) {
		if planilha.Campo(cab, l, "descricaoTipo") != "Sessão Deliberativa" ||
			planilha.Campo(cab, l, "localCamara.nome") != "Plenário da Câmara dos Deputados" ||
			!strings.HasPrefix(planilha.Campo(cab, l, "situacao"), "Encerrada") {
			return
		}
		inicio := planilha.Campo(cab, l, "dataHoraInicio")
		if len(inicio) < 10 || inicio[:10] < dominio.InicioLegislatura {
			return
		}
		sessoes[planilha.Campo(cab, l, "id")] = &sessao{data: inicio[:10]}
	})
}

func lerPresencas(ctx context.Context, ano int, sessoes map[string]*sessao) error {
	return abrirCSV(ctx, fmt.Sprintf("eventosPresencaDeputados/csv/eventosPresencaDeputados-%d.csv", ano), func(cab map[string]int, l []string) {
		s, ok := sessoes[planilha.Campo(cab, l, "idEvento")]
		if !ok {
			return
		}
		idDep, err := strconv.Atoi(planilha.Campo(cab, l, "idDeputado"))
		if err != nil {
			return
		}
		if s.presentes == nil {
			s.presentes = map[int]struct{}{}
		}
		s.presentes[idDep] = struct{}{}
	})
}
