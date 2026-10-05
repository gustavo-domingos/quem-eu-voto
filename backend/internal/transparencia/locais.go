package transparencia

import (
	"context"
	"strings"

	"quemeuvoto/internal/dominio"
	"quemeuvoto/internal/fontes/tse"
)

// Municipios lista os municípios de uma UF com prefeito e vereadores eleitos em 2024.
func (d *Servico) Municipios(ctx context.Context, uf string) ([]tse.Municipio, error) {
	uf = strings.ToUpper(uf)
	if !dominio.UFsValidas[uf] {
		return nil, entradaInvalida("UF inválida")
	}
	indice := d.municipal.Obter(ctx)
	if indice == nil {
		return nil, indisponivel("Dados dos municípios indisponíveis no momento")
	}
	return indice.Municipios[uf], nil
}

// Vagas diz quantos representantes a UF (ou o país, com "TODOS") elege em cada cargo e,
// se for indicado um município, quantos prefeitos e vereadores ele elegeu em 2024.
func (d *Servico) Vagas(ctx context.Context, uf, municipio string) (*tse.VagasDTO, error) {
	uf = strings.ToUpper(uf)
	if uf != "" && uf != "TODOS" && !dominio.UFsValidas[uf] {
		return nil, entradaInvalida("UF inválida")
	}
	idx := d.vagas.Obter(ctx)
	if idx == nil {
		return nil, indisponivel("Vagas indisponíveis no momento")
	}
	v := idx.De(uf, municipio)
	return &v, nil
}
