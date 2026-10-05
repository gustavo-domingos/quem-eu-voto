package camara

import (
	"context"
	"log"
	"regexp"

	"quemeuvoto/internal/plataforma/planilha"
)

// ---------- Funcionários de gabinete (Câmara) ----------

const urlFuncionariosCamara = "funcionarios/csv/funcionarios.csv"

var padraoGabinete = regexp.MustCompile(`^GAB\.\s*(\d+)/(\d+)`)

// IndiceGabinetes conta os secretários parlamentares por gabinete ("4/206" = prédio 4, sala 206).
type IndiceGabinetes struct {
	PorGabinete map[string]int
}

func CarregarGabinetes(ctx context.Context) (*IndiceGabinetes, error) {
	idx := &IndiceGabinetes{PorGabinete: map[string]int{}}
	err := abrirCSV(ctx, urlFuncionariosCamara, func(cab map[string]int, l []string) {
		if planilha.Campo(cab, l, "codGrupo") != "6" { // 6 = Secretário Parlamentar
			return
		}
		if m := padraoGabinete.FindStringSubmatch(planilha.Campo(cab, l, "lotacao")); m != nil {
			idx.PorGabinete[m[1]+"/"+m[2]]++
		}
	})
	if err != nil {
		return nil, err
	}
	log.Printf("gabinetes: %d gabinetes com secretários parlamentares", len(idx.PorGabinete))
	return idx, nil
}
