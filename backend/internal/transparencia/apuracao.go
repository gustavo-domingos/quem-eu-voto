package transparencia

import (
	"context"
	"strings"
	"sync"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/fontes/tse"
)

// apuracaoDe devolve os resultados da apuração de 2026 por SQ_CANDIDATO, para as UFs pedidas
// (todas em paralelo; cada ficheiro do TSE fica 60 s em cache). UFs sem resposta ficam de fora.
func (d *Servico) apuracaoDe(ctx context.Context, chaveCargo string, ufs []string) map[string]*tse.ResultadoCandidato {
	res := map[string]*tse.ResultadoCandidato{}
	if _, ok := tse.CargoResultados[chaveCargo]; !ok {
		return res
	}
	if chaveCargo == "presidente" {
		ufs = []string{"BR"}
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, uf := range ufs {
		wg.Add(1)
		go func(uf string) {
			defer wg.Done()
			ap, err := d.apuracoes.Obter(ctx, chaveCargo, uf)
			if err != nil || ap == nil {
				return
			}
			mu.Lock()
			for i := range ap.Candidatos {
				res[ap.Candidatos[i].SQ] = &ap.Candidatos[i].ResultadoCandidato
			}
			mu.Unlock()
		}(uf)
	}
	wg.Wait()
	return res
}

// Apuracao devolve a apuração ao vivo de um cargo numa UF (ou no país, para presidente).
func (d *Servico) Apuracao(ctx context.Context, cargo, uf string) (*tse.Apuracao, error) {
	uf = strings.ToUpper(uf)
	if _, ok := tse.CargoResultados[cargo]; !ok {
		return nil, entradaInvalida("Cargo inválido")
	}
	if cargo != "presidente" && !dominio.UFsValidas[uf] {
		return nil, entradaInvalida("Escolha um estado")
	}
	ap, err := d.apuracoes.Obter(ctx, cargo, uf)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, indisponivel("Apuração indisponível no TSE neste momento")
	}
	return ap, nil
}
