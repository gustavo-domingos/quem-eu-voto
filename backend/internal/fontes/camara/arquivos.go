package camara

import (
	"context"

	"quemeuvoto/internal/plataforma/planilha"
	"quemeuvoto/internal/plataforma/rede"
)

func abrirCSV(ctx context.Context, caminho string, fn func(cab map[string]int, linha []string)) error {
	resp, err := rede.Baixar(ctx, arquivosCamara+"/"+caminho, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return planilha.Ler(resp.Body, fn)
}
