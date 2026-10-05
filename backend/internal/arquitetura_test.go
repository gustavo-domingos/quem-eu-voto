package internal_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// camadas diz, para cada pacote (ou prefixo terminado em "/"), que pacotes internos ele pode
// importar. São as regras de docs/arquitetura.md: as dependências só descem.
var camadas = map[string][]string{
	"config":        {},
	"plataforma/":   {"plataforma/"},
	"dominio":       {"plataforma/texto"},
	"fontes/":       {"dominio", "plataforma/"},
	"propostas":     {"dominio", "plataforma/", "fontes/tse"},
	"transparencia": {"dominio", "plataforma/", "fontes/", "propostas"},
	"api":           {"transparencia", "propostas", "fontes/", "dominio"},
}

func casa(pacote, padrao string) bool {
	if strings.HasSuffix(padrao, "/") {
		return strings.HasPrefix(pacote, padrao)
	}
	return pacote == padrao
}

func TestDependenciasEntreCamadas(t *testing.T) {
	const prefixo = "quemeuvoto/internal/"
	importacoes := map[string]map[string]bool{} // pacote -> pacotes internos importados
	err := filepath.WalkDir(".", func(caminho string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(caminho, ".go") || strings.HasSuffix(caminho, "_test.go") {
			return err
		}
		f, err := parser.ParseFile(token.NewFileSet(), caminho, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		pacote := filepath.ToSlash(filepath.Dir(caminho))
		for _, im := range f.Imports {
			p, _ := strconv.Unquote(im.Path.Value)
			if dep, ok := strings.CutPrefix(p, prefixo); ok && dep != pacote {
				if importacoes[pacote] == nil {
					importacoes[pacote] = map[string]bool{}
				}
				importacoes[pacote][dep] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for pacote, deps := range importacoes {
		var regra []string
		achou := false
		for padrao, permitidos := range camadas {
			if casa(pacote, padrao) {
				regra, achou = permitidos, true
			}
		}
		if !achou {
			t.Errorf("pacote %s não tem camada definida em arquitetura_test.go", pacote)
			continue
		}
		for dep := range deps {
			ok := false
			for _, p := range regra {
				ok = ok || casa(dep, p)
			}
			if !ok {
				t.Errorf("%s não pode importar %s (veja docs/arquitetura.md)", pacote, dep)
			}
		}
	}
}
