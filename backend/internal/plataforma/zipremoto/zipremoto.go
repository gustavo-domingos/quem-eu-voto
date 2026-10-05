package zipremoto

import (
	"archive/zip"
	"compress/flate"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"quemeuvoto/internal/plataforma/rede"
)

const (
	blocoAbertura = 1 << 20  // blocos grandes para ler o diretório central de uma vez
	blocoArquivo  = 64 << 10 // uma foto (~10 KB) costuma caber num único bloco
	maxBlocos     = 128      // ~8 MB de cache por zip
	maxArquivo    = 5 << 20
)

// Zip lê ficheiros de um zip num servidor HTTP com pedidos Range, sem o
// descarregar inteiro. Os pacotes de fotos do TSE chegam a vários GB (SP 2024),
// mas cada foto só custa um pedido de 64 KB.
type Zip struct {
	url      string
	tamanho  int64
	Arquivos map[string]*zip.File `json:"-"`

	mu       sync.Mutex
	tamBloco int64
	blocos   map[int64][]byte
	ordem    []int64
}

func Abrir(ctx context.Context, url string) (*Zip, error) {
	resp, err := rede.Pedir(ctx, rede.ClienteArquivos, http.MethodHead, url, nil)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	if resp.ContentLength <= 0 {
		return nil, fmt.Errorf("tamanho desconhecido para %s", url)
	}

	z := &Zip{url: url, tamanho: resp.ContentLength, tamBloco: blocoAbertura, blocos: map[int64][]byte{}}
	zr, err := zip.NewReader(z, z.tamanho)
	if err != nil {
		return nil, err
	}

	z.mu.Lock()
	z.tamBloco, z.blocos, z.ordem = blocoArquivo, map[int64][]byte{}, nil
	z.mu.Unlock()

	// Indexado sem extensão: o TSE mistura ".jpg" e ".jpeg" no mesmo pacote.
	z.Arquivos = make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		z.Arquivos[strings.TrimSuffix(f.Name, path.Ext(f.Name))] = f
	}
	return z, nil
}

// Ler devolve o conteúdo e a extensão (".jpg", ".jpeg", ...) do ficheiro com este
// nome sem extensão; ok=false se não existir.
func (z *Zip) Ler(semExtensao string) (conteudo []byte, extensao string, ok bool, err error) {
	f, existe := z.Arquivos[semExtensao]
	if !existe {
		return nil, "", false, nil
	}
	extensao = strings.ToLower(path.Ext(f.Name))
	rc, err := f.Open()
	if err != nil {
		return nil, extensao, true, err
	}
	defer rc.Close()
	conteudo, err = io.ReadAll(io.LimitReader(rc, maxArquivo))
	return conteudo, extensao, true, err
}

// AbrirFluxo lê um ficheiro grande do zip do princípio ao fim com um único pedido
// Range, descompactando à medida que chega (sem guardar o zip em memória ou disco).
func (z *Zip) AbrirFluxo(ctx context.Context, nome string) (io.ReadCloser, error) {
	f, ok := z.Arquivos[strings.TrimSuffix(nome, path.Ext(nome))]
	if !ok {
		return nil, fmt.Errorf("%s não existe no zip", nome)
	}
	inicio, err := f.DataOffset()
	if err != nil {
		return nil, err
	}
	fim := inicio + int64(f.CompressedSize64) - 1
	resp, err := rede.Baixar(ctx, z.url, map[string]string{"Range": "bytes=" + strconv.FormatInt(inicio, 10) + "-" + strconv.FormatInt(fim, 10)})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusPartialContent {
		resp.Body.Close()
		return nil, fmt.Errorf("servidor não respeitou o pedido Range (%d)", resp.StatusCode)
	}
	switch f.Method {
	case zip.Store:
		return resp.Body, nil
	case zip.Deflate:
		return fluxoDescompactado{flate.NewReader(resp.Body), resp.Body}, nil
	default:
		resp.Body.Close()
		return nil, fmt.Errorf("método de compressão %d não suportado", f.Method)
	}
}

type fluxoDescompactado struct {
	io.ReadCloser
	corpo io.Closer
}

func (f fluxoDescompactado) Close() error {
	f.ReadCloser.Close()
	return f.corpo.Close()
}

func (z *Zip) ReadAt(p []byte, off int64) (int, error) {
	n := 0
	for n < len(p) {
		pos := off + int64(n)
		if pos >= z.tamanho {
			return n, io.EOF
		}
		inicio, bloco, err := z.bloco(pos)
		if err != nil {
			return n, err
		}
		n += copy(p[n:], bloco[pos-inicio:])
	}
	return n, nil
}

func (z *Zip) bloco(pos int64) (int64, []byte, error) {
	z.mu.Lock()
	tam := z.tamBloco
	inicio := pos / tam * tam
	if b, ok := z.blocos[inicio]; ok {
		z.mu.Unlock()
		return inicio, b, nil
	}
	z.mu.Unlock()

	// O pedido é feito fora do lock para que várias fotos possam ser lidas em paralelo.
	fim := min(inicio+tam, z.tamanho) - 1
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	resp, err := rede.Baixar(ctx, z.url, map[string]string{"Range": "bytes=" + strconv.FormatInt(inicio, 10) + "-" + strconv.FormatInt(fim, 10)})
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent || !strings.HasPrefix(resp.Header.Get("Content-Range"), "bytes "+strconv.FormatInt(inicio, 10)+"-") {
		return 0, nil, fmt.Errorf("servidor não respeitou o pedido Range (%d)", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	if int64(len(b)) != fim-inicio+1 {
		return 0, nil, io.ErrUnexpectedEOF
	}

	z.mu.Lock()
	if tam == z.tamBloco { // não guarda blocos da fase de abertura depois de ela terminar
		if _, ok := z.blocos[inicio]; !ok {
			z.blocos[inicio] = b
			z.ordem = append(z.ordem, inicio)
			if len(z.ordem) > maxBlocos {
				delete(z.blocos, z.ordem[0])
				z.ordem = z.ordem[1:]
			}
		}
	}
	z.mu.Unlock()
	return inicio, b, nil
}
