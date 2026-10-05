package propostas

import (
	"reflect"
	"testing"
)

func TestExtrairCompromissos(t *testing.T) {
	texto := `PLANO DE GOVERNO
EDUCAÇÃO
• Implementar a educação em tempo integral em todas as escolas
estaduais até 2030;
• Fortalecer e ampliar o programa de PPPs e concessões do estado de
Santa Catarina
• Criar a Diretoria de Inteligência de Combate à Corrupção Pública
GARANTIR SAÚDE E EDUCAÇÃO PÚBLICAS DE QUALIDADE PARA TODOS OS CATARINENSES
Acreditamos num estado eficiente e moderno para todos os cidadãos.`

	got := extrairCompromissos(texto, 10)
	want := []string{
		// a linha seguinte começa em minúscula: é a mesma frase; e as que têm números vêm primeiro
		"Implementar a educação em tempo integral em todas as escolas estaduais até 2030",
		"Criar a Diretoria de Inteligência de Combate à Corrupção Pública",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("compromissos:\n got %q\nwant %q", got, want)
	}
}

func TestResumoPorTema(t *testing.T) {
	texto := `3. SEGURANÇA PÚBLICA
• Contratar 2.000 novos agentes para a Polícia Militar até 2028
• Implantar câmeras corporais em todas as viaturas do estado
4. SAÚDE
• Construir 5 hospitais regionais no interior do estado
• Ampliar o atendimento noturno nas unidades de pronto atendimento
• Reduzir as filas de cirurgias eletivas com mutirões mensais`

	got := resumirPorTema(extrairFrases(texto), 5)
	if len(got) != 2 || got[0].Tema != "Saúde" || got[0].Total != 3 || got[1].Tema != "Segurança pública" || got[1].Total != 2 {
		t.Fatalf("temas inesperados: %+v", got)
	}
	// Dentro do tema, a frase com números vem primeiro; "Ampliar…" não tem palavra-chave de saúde,
	// mas fica em Saúde por estar debaixo do título "4. SAÚDE".
	if got[0].Compromissos[0] != "Construir 5 hospitais regionais no interior do estado" {
		t.Errorf("primeiro compromisso de Saúde: %q", got[0].Compromissos[0])
	}
}

func TestCompromissosNominaisENoFuturo(t *testing.T) {
	texto := `Criação de 20 novas escolas técnicas no interior do estado.
O Estado investirá R$ 2 bilhões na duplicação de rodovias estaduais.
A administração pública pautar-se-á pela busca constante de eficiência e eficácia.
Será um governo de todos e para todos os catarinenses sem exceção.`
	got := extrairCompromissos(texto, 10)
	want := []string{
		"Criação de 20 novas escolas técnicas no interior do estado",
		"O Estado investirá R$ 2 bilhões na duplicação de rodovias estaduais",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("compromissos:\n got %q\nwant %q", got, want)
	}
}

func TestContarTemasPalavraInteira(t *testing.T) {
	// "SUS" não pode contar dentro de "sustentável".
	temas := contarTemas("Desenvolvimento sustentável e sustentabilidade ambiental")
	for _, c := range temas {
		if c.Rotulo == "Saúde" {
			t.Fatalf("'sustentável' contou como Saúde: %+v", temas)
		}
	}
}

func TestTerminaCortada(t *testing.T) {
	if !terminaCortada("Fortalecer o programa de concessões do estado de") {
		t.Error("frase que acaba em 'de' devia ser considerada cortada")
	}
	if terminaCortada("Universalizar o saneamento básico até 2033") {
		t.Error("frase completa não devia ser considerada cortada")
	}
}
