package transparencia

import (
	"math"
	"testing"
	"time"

	"quemeuvoto/internal/dominio"
)

func TestEstimarSubsidio(t *testing.T) {
	hoje := time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC)
	s := estimarSubsidio([]dominio.Periodo{{Inicio: "2023-02-01"}}, hoje)
	// fev e mar inteiros a 33.763,00; abr inteiro a 39.293,32; 15 de 31 dias de maio a 39.293,32
	esperado := math.Round((2*33763.00+39293.32+39293.32*15/31)*100) / 100
	if s.Dias != 28+31+30+15 || s.Total != esperado || s.Mensal != 39293.32 {
		t.Fatalf("subsídio inesperado: %+v (esperado total %v)", s, esperado)
	}
	// Afastamentos não contam: dois períodos de 10 dias em fevereiro (28 dias) = 20/28 do mês.
	s2 := estimarSubsidio([]dominio.Periodo{{Inicio: "2023-02-01", Fim: "2023-02-10"}, {Inicio: "2023-02-19", Fim: "2023-02-28"}}, hoje)
	if s2.Dias != 20 || s2.Total != math.Round(33763.00*20/28*100)/100 {
		t.Fatalf("períodos com afastamento: %+v", s2)
	}
	// Antes da legislatura não conta.
	if s3 := estimarSubsidio([]dominio.Periodo{{Inicio: "2019-02-01"}}, hoje); s3.Desde != dominio.InicioLegislatura {
		t.Fatalf("início devia ser limitado à legislatura: %+v", s3)
	}
}
