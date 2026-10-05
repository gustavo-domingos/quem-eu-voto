package transparencia

import (
	"context"

	"quemeuvoto/internal/fontes/sancoes"
)

// cpfDe encontra o CPF de quem só conhecemos por nome + nascimento (o TSE esconde o CPF
// em 2024; a API do Senado não o publica), procurando nos registos de 2026 e 2022.
func (d *Servico) cpfDe(ctx context.Context, cpf, nomeNasc string) string {
	if len(cpf) == 11 {
		return cpf
	}
	if c := d.candidatos.Obter(ctx); c != nil {
		for _, x := range c.PorNomeNasc[nomeNasc] {
			if len(x.Cpf) == 11 {
				return x.Cpf
			}
		}
	}
	if g := d.geral2022.Obter(ctx); g != nil {
		if cpf, ok := g.CpfPorNomeNasc[nomeNasc]; ok {
			return cpf
		}
	}
	return ""
}

// ResumoJusticaDTO vai em cada cartão: quantos registos oficiais há e se algum envolve corrupção.
type ResumoJusticaDTO struct {
	Total      int  `json:"total"`
	Corrupcao  bool `json:"corrupcao"`
	Verificado bool `json:"verificado"` // false quando não há CPF para cruzar com a CGU
}

func resumoJustica(regs []sancoes.RegistroJustica, verificado bool) ResumoJusticaDTO {
	r := ResumoJusticaDTO{Total: len(regs), Verificado: verificado}
	for _, x := range regs {
		r.Corrupcao = r.Corrupcao || x.Corrupcao
	}
	return r
}
