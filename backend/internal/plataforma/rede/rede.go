// Package rede faz os pedidos HTTP às fontes oficiais (Câmara, Senado, TSE, CGU), com novas
// tentativas em falhas temporárias.
package rede

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

var (
	// ClienteAPI é para APIs (respostas pequenas): desiste ao fim de 30 s.
	ClienteAPI = &http.Client{Timeout: 30 * time.Second}
	// ClienteArquivos é para downloads grandes: o prazo fica a cargo do ctx de quem chama.
	ClienteArquivos = &http.Client{}
)

// ErroStatus é uma resposta HTTP com um código diferente de 200/206.
type ErroStatus struct{ Status int }

func (e ErroStatus) Error() string { return fmt.Sprintf("o servidor respondeu HTTP %d", e.Status) }

// Get faz um GET a uma API com novas tentativas em erros de rede, 429 e 5xx.
// Quem chama é responsável por fechar o corpo da resposta.
func Get(ctx context.Context, url string, accept string) (*http.Response, error) {
	cab := map[string]string{}
	if accept != "" {
		cab["Accept"] = accept
	}
	return Pedir(ctx, ClienteAPI, http.MethodGet, url, cab)
}

// Baixar descarrega ficheiros grandes (dados em massa, zips do TSE), sem o limite de 30 s
// da API; o tempo máximo fica a cargo do ctx.
func Baixar(ctx context.Context, url string, cabecalhos map[string]string) (*http.Response, error) {
	return Pedir(ctx, ClienteArquivos, http.MethodGet, url, cabecalhos)
}

// Pedir faz o pedido com até 4 tentativas (espera crescente entre elas) em erros de rede,
// 429 e 5xx. Devolve a resposta só com 200 ou 206.
func Pedir(ctx context.Context, cli *http.Client, metodo, url string, cabecalhos map[string]string) (*http.Response, error) {
	var ultimoErro error
	for tentativa := 0; tentativa < 4; tentativa++ {
		if tentativa > 0 {
			espera := time.Duration(500<<tentativa) * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(espera):
			}
		}

		req, err := http.NewRequestWithContext(ctx, metodo, url, nil)
		if err != nil {
			return nil, err
		}
		for k, v := range cabecalhos {
			req.Header.Set(k, v)
		}

		resp, err := cli.Do(req)
		if err != nil {
			ultimoErro = err
			continue
		}
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent {
			return resp, nil
		}
		resp.Body.Close()
		ultimoErro = ErroStatus{Status: resp.StatusCode}
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			break
		}
	}
	return nil, ultimoErro
}
