package transparencia

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/texto"
)

// Indicador é um facto objetivo, com fonte, destacado no perfil.
type Indicador struct {
	Tipo  string `json:"tipo"` // "atencao", "destaque" ou "info"
	Texto string `json:"texto"`
	Fonte string `json:"fonte"`
}

// ---------- Indicadores (factos objetivos, sempre com fonte) ----------

func reais(v float64) string {
	inteiro := strconv.FormatInt(int64(math.Round(v)), 10)
	var b strings.Builder
	for i, r := range inteiro {
		if i > 0 && (len(inteiro)-i)%3 == 0 {
			b.WriteRune('.')
		}
		b.WriteRune(r)
	}
	return "R$ " + b.String()
}

func porcento(v float64) string {
	return strings.Replace(strconv.FormatFloat(v, 'f', 1, 64), ".", ",", 1) + "%"
}

func gerarIndicadores(perfil *PerfilDTO, p *pessoa, deputados []DeputadoDTO) {
	add := func(tipo, fonte, formato string, args ...any) {
		perfil.Indicadores = append(perfil.Indicadores, Indicador{Tipo: tipo, Texto: fmt.Sprintf(formato, args...), Fonte: fonte})
	}

	for _, c := range perfil.Candidaturas2026 {
		switch {
		case !c.Apto:
			add("atencao", "TSE", "Candidatura a %s em %d: %s.", c.Cargo, dominio.AnoEleicao, strings.ToLower(c.Situacao))
		case c.Situacao != "" && c.Situacao != "Deferido":
			add("atencao", "TSE", "Candidatura a %s sub judice: %s.", c.Cargo, strings.ToLower(c.Situacao))
		}
	}

	if mand := perfil.Mandato; mand != nil {
		if des := mand.Desempenho; des != nil && des.Percentual != nil {
			media := 0.0
			if p.deputado != nil {
				soma, n := 0.0, 0
				for _, d := range deputados {
					if d.Assiduidade != nil {
						soma += *d.Assiduidade
						n++
					}
				}
				if n > 0 {
					media = soma / float64(n)
				}
			}
			fonte := "Câmara dos Deputados"
			if p.senador != nil && p.deputado == nil {
				fonte = "Senado Federal"
			}
			comparacao := ""
			if media > 0 {
				comparacao = fmt.Sprintf(" (média da Câmara: %s)", porcento(media))
			}
			switch {
			case *des.Percentual < 80:
				add("atencao", fonte, "%s de %s no mandato%s.", des.Rotulo, porcento(*des.Percentual), comparacao)
			case *des.Percentual >= 95:
				add("destaque", fonte, "%s de %s no mandato%s.", des.Rotulo, porcento(*des.Percentual), comparacao)
			}
		}
		if p.deputado != nil && p.deputado.TotalProjetos != nil {
			soma, n := 0, 0
			for _, d := range deputados {
				if d.TotalProjetos != nil {
					soma += *d.TotalProjetos
					n++
				}
			}
			if n > 0 {
				media := float64(soma) / float64(n)
				if float64(*p.deputado.TotalProjetos) >= 1.5*media {
					add("destaque", "Câmara dos Deputados", "Apresentou %d projetos (PL, PLP, PEC) desde 2023, acima da média de %.0f por deputado.", *p.deputado.TotalProjetos, media)
				} else if float64(*p.deputado.TotalProjetos) <= 0.25*media {
					add("atencao", "Câmara dos Deputados", "Apresentou %d projetos (PL, PLP, PEC) desde 2023; a média por deputado é %.0f.", *p.deputado.TotalProjetos, media)
				}
			}
		}
		if gs := mand.Gastos; gs != nil && gs.MediaDeputados > 0 {
			dif := (gs.Total - gs.MediaDeputados) / gs.MediaDeputados * 100
			switch {
			case dif >= 20:
				add("atencao", "Câmara dos Deputados (cota parlamentar)", "Gastou %s da cota parlamentar desde 2023, %s acima da média dos deputados (%dº de %d que mais gastaram).", reais(gs.Total), porcento(dif), gs.Posicao, gs.De)
			case dif <= -20:
				add("destaque", "Câmara dos Deputados (cota parlamentar)", "Gastou %s da cota parlamentar desde 2023, %s abaixo da média dos deputados.", reais(gs.Total), porcento(-dif))
			}
		}
		if v := mand.Votacoes; v != nil && v.AlinhamentoGoverno != nil {
			add("info", v.Casa, "Votou de acordo com a orientação do Governo em %s das %d votações em que o Governo orientou Sim ou Não.", porcento(*v.AlinhamentoGoverno), v.Comparaveis)
		}
		if mand.Votacao != nil && mand.Votacao.Posicao <= 3 && mand.Votacao.Disputaram > 3 {
			add("destaque", "TSE", "Foi o %dº mais votado da disputa em %d, com %s dos votos válidos.", mand.Votacao.Posicao, mand.AnoEleicao, porcento(mand.Votacao.Percentual))
		}
		if mand.Nota != "" {
			add("info", "TSE", "%s", mand.Nota)
		}
	}

	if pat := perfil.Patrimonio; pat != nil {
		if pat.Total2022 != nil && pat.Variacao != nil && *pat.Variacao >= 100 && pat.Total2026-*pat.Total2022 >= 500_000 {
			add("atencao", "TSE (bens declarados)", "Patrimônio declarado passou de %s (2022) para %s (2026): aumento de %s.", reais(*pat.Total2022), reais(pat.Total2026), porcento(*pat.Variacao))
		}
		if len(pat.Bens) == 0 {
			add("info", "TSE (bens declarados)", "Não declarou bens ao TSE em %d.", dominio.AnoEleicao)
		}
	}

	// Mudança de partido entre a eleição anterior e a candidatura atual.
	if len(perfil.Candidaturas2026) > 0 {
		atual := perfil.Candidaturas2026[0].Partido
		for _, h := range perfil.Historico {
			if h.Ano < dominio.AnoEleicao && h.Partido != "" && texto.Normalizar(h.Partido) != texto.Normalizar(atual) {
				add("info", "TSE", "Mudou de partido: concorreu pelo %s em %d e pelo %s em %d.", h.Partido, h.Ano, atual, dominio.AnoEleicao)
				break
			}
			if h.Ano < dominio.AnoEleicao {
				break
			}
		}
	}
}
