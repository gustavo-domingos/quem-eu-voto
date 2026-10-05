// Package cache guarda em disco, como JSON, dados caros de obter (listas da Câmara,
// contas de campanha, análises de propostas), para sobreviverem a reinícios do servidor.
package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// pasta onde a cache fica; vazia = pasta de cache do utilizador.
var pasta string

// DefinirPasta muda a pasta da cache em disco (ex.: um volume persistente no servidor).
func DefinirPasta(p string) { pasta = p }

// Caminho devolve um ficheiro na pasta da cache (por omissão, %LocalAppData%\quem-eu-voto
// no Windows e ~/.cache/quem-eu-voto no Linux).
func Caminho(nome string) string {
	if pasta != "" {
		return filepath.Join(pasta, nome)
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "quem-eu-voto", nome)
}

// Ler lê um JSON da cache em disco se tiver sido gravado há menos de `validade`.
func Ler[T any](nome string, validade time.Duration) (*T, bool) {
	caminho := Caminho(nome)
	info, err := os.Stat(caminho)
	if err != nil || time.Since(info.ModTime()) > validade {
		return nil, false
	}
	f, err := os.Open(caminho)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	var v T
	if err := json.NewDecoder(f).Decode(&v); err != nil {
		return nil, false
	}
	return &v, true
}

// Gravar grava de forma atómica (ficheiro temporário + rename).
func Gravar(nome string, v any) error {
	caminho := Caminho(nome)
	if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
		return err
	}
	tmp := caminho + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(f).Encode(v); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, caminho)
}
