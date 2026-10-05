package planilha

import (
	"io"
	"unicode/utf8"
)

// LeitorLatin1 converte um fluxo ISO-8859-1 em UTF-8 (cada byte é o próprio code point).
type LeitorLatin1 struct {
	Origem   io.Reader `json:"-"`
	pendente []byte
}

func (l *LeitorLatin1) Read(p []byte) (int, error) {
	if len(l.pendente) == 0 {
		buf := make([]byte, len(p)/2+1)
		n, err := l.Origem.Read(buf)
		for _, b := range buf[:n] {
			l.pendente = utf8.AppendRune(l.pendente, rune(b))
		}
		if n == 0 {
			return 0, err
		}
	}
	n := copy(p, l.pendente)
	l.pendente = l.pendente[n:]
	return n, nil
}
