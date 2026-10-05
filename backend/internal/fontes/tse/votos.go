package tse

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/plataforma/cache"
	"quemeuvoto/internal/plataforma/planilha"
	"quemeuvoto/internal/plataforma/zipremoto"
)

const urlVotosTSE = "https://cdn.tse.jus.br/estatistica/sead/odsele/votacao_candidato_munzona/votacao_candidato_munzona_%d.zip"

// carregarVotos soma os votos por candidato a partir do ficheiro por município/zona do TSE.
// O de 2022 tem 4,3 GB descompactado: é lido em fluxo e o resultado agregado (alguns MB)
// fica guardado em disco durante validadeCache, para não repetir a leitura a cada arranque.
// Com somenteEleitos, o resultado só guarda quem foi eleito (as posições continuam a ser
// calculadas sobre todos): em 2024 são ~64 mil eleitos contra ~430 mil candidatos.
func carregarVotos(ctx context.Context, ano int, somenteEleitos bool, validadeCache time.Duration) (*dominio.ResultadoVotacao, error) {
	nomeCache := fmt.Sprintf("votos_%d.json", ano)
	if r, ok := cache.Ler[dominio.ResultadoVotacao](nomeCache, validadeCache); ok {
		log.Printf("votos %d: lidos da cache em disco (%d candidatos)", ano, len(r.PorCandidato))
		return r, nil
	}

	inicio := time.Now()
	zr, err := zipremoto.Abrir(ctx, fmt.Sprintf(urlVotosTSE, ano))
	if err != nil {
		return nil, err
	}
	fluxo, err := zr.AbrirFluxo(ctx, fmt.Sprintf("votacao_candidato_munzona_%d_BRASIL.csv", ano))
	if err != nil {
		return nil, err
	}
	defer fluxo.Close()

	type chaveDisputa struct {
		eleicao, cargo, ue string
		turno              int
	}
	type parcial struct {
		disputa        chaveDisputa
		votos, validos int64
	}
	porCandTurno := map[string]*parcial{} // SQ|turno
	totalValidos := map[chaveDisputa]int64{}
	eleitos := map[string]bool{}

	// Os campos usados são ASCII: dispensa a conversão de Latin-1, o que poupa muito tempo em 4 GB.
	err = planilha.Ler(fluxo, func(cab map[string]int, l []string) {
		turno, _ := strconv.Atoi(planilha.Campo(cab, l, "NR_TURNO"))
		votos, _ := strconv.ParseInt(planilha.Campo(cab, l, "QT_VOTOS_NOMINAIS"), 10, 64)
		validos, _ := strconv.ParseInt(planilha.Campo(cab, l, "QT_VOTOS_NOMINAIS_VALIDOS"), 10, 64)
		sq := planilha.Campo(cab, l, "SQ_CANDIDATO")
		if strings.HasPrefix(planilha.Campo(cab, l, "DS_SIT_TOT_TURNO"), "ELEITO") {
			eleitos[sq] = true
		}
		d := chaveDisputa{planilha.Campo(cab, l, "CD_ELEICAO"), planilha.Campo(cab, l, "CD_CARGO"), planilha.Campo(cab, l, "SG_UE"), turno}

		chave := sq + "|" + strconv.Itoa(turno)
		p, ok := porCandTurno[chave]
		if !ok {
			p = &parcial{disputa: d}
			porCandTurno[chave] = p
		}
		p.votos += votos
		p.validos += validos
		totalValidos[d] += validos
	})
	if err != nil {
		return nil, err
	}

	// Posição de cada candidato na sua disputa.
	porDisputa := map[chaveDisputa][]string{}
	for chave, p := range porCandTurno {
		porDisputa[p.disputa] = append(porDisputa[p.disputa], chave)
	}
	r := &dominio.ResultadoVotacao{Ano: ano, PorCandidato: map[string]*dominio.Votacao{}, GeradoEm: time.Now()}
	for d, chaves := range porDisputa {
		sort.Slice(chaves, func(i, j int) bool { return porCandTurno[chaves[i]].votos > porCandTurno[chaves[j]].votos })
		for i, chave := range chaves {
			p := porCandTurno[chave]
			sq := chave[:len(chave)-len(strconv.Itoa(d.turno))-1]
			if somenteEleitos && !eleitos[sq] {
				continue
			}
			// Fica o último turno disputado (2º turno, quando houve).
			if atual, ok := r.PorCandidato[sq]; ok && atual.Turno > d.turno {
				continue
			}
			v := &dominio.Votacao{Votos: p.votos, Turno: d.turno, Posicao: i + 1, Disputaram: len(chaves)}
			if total := totalValidos[d]; total > 0 {
				v.Percentual = float64(p.validos) / float64(total) * 100
			}
			r.PorCandidato[sq] = v
		}
	}
	if len(r.PorCandidato) == 0 {
		return nil, fmt.Errorf("ficheiro de votação de %d sem dados", ano)
	}
	log.Printf("votos %d: %d candidatos agregados em %s", ano, len(r.PorCandidato), time.Since(inicio).Round(time.Second))
	if err := cache.Gravar(nomeCache, r); err != nil {
		log.Printf("votos %d: não foi possível gravar a cache: %v", ano, err)
	}
	return r, nil
}
