package ocr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"quemeuvoto/internal/plataforma/cache"
)

// OCR opcional para propostas entregues como imagem (PDF digitalizado). Usa dois programas
// gratuitos, se estiverem instalados: pdftoppm (Poppler) converte as páginas em imagens e o
// Tesseract lê o texto. Os caminhos podem ser indicados nas variáveis PDFTOPPM e TESSERACT.

const maxPaginasOCR = 80

var ocrUmDeCadaVez sync.Mutex // o OCR ocupa o processador; não vale a pena correr vários

// programa procura um executável na variável de ambiente, no PATH e em pastas habituais no Windows.
func programa(variavel, nome string, pastas ...string) string {
	if p := os.Getenv(variavel); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath(nome); err == nil {
		return p
	}
	for _, padrao := range pastas {
		if achados, _ := filepath.Glob(filepath.Join(padrao, nome+".exe")); len(achados) > 0 {
			return achados[len(achados)-1]
		}
	}
	return ""
}

func caminhoPdftoppm() string {
	return programa("PDFTOPPM", "pdftoppm",
		`C:\Program Files\poppler*\Library\bin`, `C:\Program Files\poppler*\bin`,
		filepath.Join(os.Getenv("LOCALAPPDATA"), `Microsoft\WinGet\Packages\*Poppler*\poppler*\Library\bin`))
}

func caminhoTesseract() string {
	return programa("TESSERACT", "tesseract", `C:\Program Files\Tesseract-OCR`, `C:\Program Files (x86)\Tesseract-OCR`)
}

func Disponivel() bool {
	return caminhoPdftoppm() != "" && caminhoTesseract() != ""
}

// argumentosIdioma usa português: primeiro o pacote "por" na pasta de cache da aplicação
// (%LocalAppData%\quem-eu-voto\tessdata, não precisa de administrador), depois o do Tesseract;
// sem nenhum, fica o inglês (lê bem o texto, só erra mais nos acentos).
func argumentosIdioma(ctx context.Context, tesseract string) []string {
	dir := cache.Caminho("tessdata")
	if _, err := os.Stat(filepath.Join(dir, "por.traineddata")); err == nil {
		return []string{"--tessdata-dir", dir, "-l", "por"}
	}
	saida, err := exec.CommandContext(ctx, tesseract, "--list-langs").CombinedOutput()
	if err == nil && strings.Contains(string(saida), "\npor") {
		return []string{"-l", "por"}
	}
	return []string{"-l", "eng"}
}

// TextoDoPDF lê até maxPaginasOCR páginas de um PDF digitalizado.
func TextoDoPDF(ctx context.Context, conteudo []byte) (string, error) {
	pdftoppm, tesseract := caminhoPdftoppm(), caminhoTesseract()
	if pdftoppm == "" || tesseract == "" {
		return "", fmt.Errorf("OCR indisponível: instale o Poppler (pdftoppm) e o Tesseract")
	}
	ocrUmDeCadaVez.Lock()
	defer ocrUmDeCadaVez.Unlock()

	dir, err := os.MkdirTemp("", "quem-eu-voto-ocr-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	entrada := filepath.Join(dir, "proposta.pdf")
	if err := os.WriteFile(entrada, conteudo, 0o600); err != nil {
		return "", err
	}
	if saida, err := exec.CommandContext(ctx, pdftoppm, "-r", "200", "-gray", "-png", "-l", strconv.Itoa(maxPaginasOCR),
		entrada, filepath.Join(dir, "pagina")).CombinedOutput(); err != nil {
		return "", fmt.Errorf("pdftoppm: %v: %s", err, saida)
	}
	paginas, _ := filepath.Glob(filepath.Join(dir, "pagina*.png"))
	sort.Strings(paginas) // pdftoppm numera com zeros à esquerda: a ordem alfabética é a das páginas

	idioma := argumentosIdioma(ctx, tesseract)
	var texto strings.Builder
	for _, pg := range paginas {
		args := append([]string{pg, "stdout", "--psm", "3"}, idioma...)
		saida, err := exec.CommandContext(ctx, tesseract, args...).Output()
		if err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			continue // uma página ilegível não estraga o resto
		}
		texto.Write(saida)
		texto.WriteString("\n")
	}
	return texto.String(), nil
}
