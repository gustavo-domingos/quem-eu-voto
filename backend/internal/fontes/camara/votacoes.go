package camara

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/cache"
	"quemeuvoto/internal/plataforma/planilha"
	"quemeuvoto/internal/plataforma/texto"
)

// VotosDeputado resume como o deputado votou no Plenário.
type VotosDeputado struct {
	Contagem    map[string]int        `json:"contagem"`    // "Sim", "Não", "Abstenção", "Obstrução", ...
	Alinhados   int                   `json:"alinhados"`   // votou como o Governo orientou
	Comparaveis int                   `json:"comparaveis"` // votações em que o Governo orientou Sim/Não e o deputado votou Sim/Não
	Recentes    []dominio.VotoRecente `json:"recentes"`    // mais recentes primeiro
}

func (v *VotosDeputado) juntar(o *VotosDeputado) {
	if v.Contagem == nil {
		v.Contagem = map[string]int{}
	}
	for k, n := range o.Contagem {
		v.Contagem[k] += n
	}
	v.Alinhados += o.Alinhados
	v.Comparaveis += o.Comparaveis
	v.Recentes = append(v.Recentes, o.Recentes...)
	sort.Slice(v.Recentes, func(i, j int) bool { return v.Recentes[i].Data > v.Recentes[j].Data })
	if len(v.Recentes) > dominio.MaxVotosRecentes {
		v.Recentes = v.Recentes[:dominio.MaxVotosRecentes]
	}
}

type VotacoesCamara struct {
	PorDeputado map[int]*VotosDeputado
}

// CarregarVotacoesCamara junta as votações nominais do Plenário desde o início da legislatura.
// Cada ano é agregado uma vez e guardado em disco (os anos fechados não mudam).
func CarregarVotacoesCamara(ctx context.Context) (*VotacoesCamara, error) {
	total := &VotacoesCamara{PorDeputado: map[int]*VotosDeputado{}}
	for ano := dominio.AnoInicioLeg; ano <= time.Now().Year(); ano++ {
		validade := 30 * 24 * time.Hour
		if ano == time.Now().Year() {
			validade = 20 * time.Hour
		}
		nome := fmt.Sprintf("camara_votacoes_%d.json", ano)
		porAno, ok := cache.Ler[map[int]*VotosDeputado](nome, validade)
		if !ok {
			inicio := time.Now()
			m, err := agregarVotacoesAno(ctx, ano)
			if err != nil {
				return nil, fmt.Errorf("votações %d: %w", ano, err)
			}
			porAno = &m
			if err := cache.Gravar(nome, m); err != nil {
				log.Printf("votações Câmara %d: não foi possível gravar a cache: %v", ano, err)
			}
			log.Printf("votações Câmara %d: %d deputados em %s", ano, len(m), time.Since(inicio).Round(time.Second))
		}
		for id, v := range *porAno {
			if total.PorDeputado[id] == nil {
				total.PorDeputado[id] = &VotosDeputado{Contagem: map[string]int{}}
			}
			total.PorDeputado[id].juntar(v)
		}
	}
	return total, nil
}

func agregarVotacoesAno(ctx context.Context, ano int) (map[int]*VotosDeputado, error) {
	type votacao struct{ data, descricao, proposicao, ementa, governo string }
	plenario := map[string]*votacao{}

	err := abrirCSV(ctx, fmt.Sprintf("votacoes/csv/votacoes-%d.csv", ano), func(cab map[string]int, l []string) {
		if planilha.Campo(cab, l, "siglaOrgao") == "PLEN" {
			plenario[planilha.Campo(cab, l, "id")] = &votacao{data: planilha.Campo(cab, l, "data"), descricao: texto.ResumirTexto(planilha.Campo(cab, l, "descricao"), 220)}
		}
	})
	if err != nil {
		return nil, err
	}
	err = abrirCSV(ctx, fmt.Sprintf("votacoesProposicoes/csv/votacoesProposicoes-%d.csv", ano), func(cab map[string]int, l []string) {
		if v, ok := plenario[planilha.Campo(cab, l, "idVotacao")]; ok && v.proposicao == "" {
			v.proposicao = planilha.Campo(cab, l, "proposicao_titulo")
			v.ementa = texto.ResumirTexto(planilha.Campo(cab, l, "proposicao_ementa"), 300)
		}
	})
	if err != nil {
		return nil, err
	}
	err = abrirCSV(ctx, fmt.Sprintf("votacoesOrientacoes/csv/votacoesOrientacoes-%d.csv", ano), func(cab map[string]int, l []string) {
		if v, ok := plenario[planilha.Campo(cab, l, "idVotacao")]; ok && planilha.Campo(cab, l, "siglaBancada") == "Governo" {
			v.governo = planilha.Campo(cab, l, "orientacao")
		}
	})
	if err != nil {
		return nil, err
	}

	res := map[int]*VotosDeputado{}
	err = abrirCSV(ctx, fmt.Sprintf("votacoesVotos/csv/votacoesVotos-%d.csv", ano), func(cab map[string]int, l []string) {
		v, ok := plenario[planilha.Campo(cab, l, "idVotacao")]
		if !ok {
			return
		}
		id, err := strconv.Atoi(planilha.Campo(cab, l, "deputado_id"))
		if err != nil {
			return
		}
		d := res[id]
		if d == nil {
			d = &VotosDeputado{Contagem: map[string]int{}}
			res[id] = d
		}
		voto := planilha.Campo(cab, l, "voto")
		d.Contagem[voto]++
		if (v.governo == "Sim" || v.governo == "Não") && (voto == "Sim" || voto == "Não") {
			d.Comparaveis++
			if voto == v.governo {
				d.Alinhados++
			}
		}
		d.Recentes = append(d.Recentes, dominio.VotoRecente{
			Data: v.data, Proposicao: v.proposicao, Ementa: v.ementa, Descricao: v.descricao,
			Voto: voto, OrientacaoGoverno: v.governo,
		})
	})
	if err != nil {
		return nil, err
	}
	for _, d := range res {
		sort.SliceStable(d.Recentes, func(i, j int) bool { return d.Recentes[i].Data > d.Recentes[j].Data })
		if len(d.Recentes) > dominio.MaxVotosRecentes {
			d.Recentes = d.Recentes[:dominio.MaxVotosRecentes]
		}
	}
	return res, nil
}
