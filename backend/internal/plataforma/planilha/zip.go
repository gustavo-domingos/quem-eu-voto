package planilha

import (
	"archive/zip"
	"bytes"
	"context"
	"io"

	"quemeuvoto/internal/plataforma/rede"
)

// LerDoZip descarrega um zip do TSE e lê um dos CSV (em Latin-1) que ele contém.
func LerDoZip(ctx context.Context, url, nomeCSV string, fn func(cab map[string]int, linha []string)) error {
	zr, err := BaixarZip(ctx, url)
	if err != nil {
		return err
	}
	f, err := zr.Open(nomeCSV)
	if err != nil {
		return err
	}
	defer f.Close()
	return Ler(&LeitorLatin1{Origem: f}, fn)
}

func BaixarZip(ctx context.Context, url string) (*zip.Reader, error) {
	resp, err := rede.Baixar(ctx, url, nil)
	if err != nil {
		return nil, err
	}
	conteudo, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return zip.NewReader(bytes.NewReader(conteudo), int64(len(conteudo)))
}
