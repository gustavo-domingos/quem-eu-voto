package planilha

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"io"
	"strings"
)

// Ler lê um CSV separado por ";" com cabeçalho, chamando fn para cada linha.
func Ler(origem io.Reader, fn func(cab map[string]int, linha []string)) error {
	// Os ficheiros da Câmara começam com BOM UTF-8 antes da primeira aspa; sem o remover,
	// a primeira coluna seria lida como BOM+"id" e nunca seria encontrada.
	corpo := bufio.NewReader(origem)
	if bom, _ := corpo.Peek(3); bytes.Equal(bom, []byte{0xEF, 0xBB, 0xBF}) {
		corpo.Discard(3)
	}

	r := csv.NewReader(corpo)
	r.Comma = ';'
	r.LazyQuotes = true
	r.ReuseRecord = true
	r.FieldsPerRecord = -1

	cabecalho, err := r.Read()
	if err != nil {
		return err
	}
	cab := make(map[string]int, len(cabecalho))
	for i, c := range cabecalho {
		cab[c] = i
	}

	for {
		linha, err := r.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		fn(cab, linha)
	}
}

func Campo(cab map[string]int, linha []string, nome string) string {
	if i, ok := cab[nome]; ok && i < len(linha) {
		return strings.TrimSpace(linha[i])
	}
	return ""
}
