package transparencia

// TipoErro classifica os erros do serviço; a camada HTTP traduz cada tipo num código de resposta.
type TipoErro int

const (
	EntradaInvalida TipoErro = iota + 1 // parâmetros do pedido inválidos
	NaoEncontrado                       // o recurso pedido não existe
	Indisponivel                        // uma fonte oficial (TSE, Câmara, Senado, CGU) não respondeu
)

// Erro é um erro com uma mensagem pronta para mostrar a quem usa o site.
type Erro struct {
	Tipo     TipoErro
	Mensagem string
}

func (e *Erro) Error() string { return e.Mensagem }

func entradaInvalida(msg string) error { return &Erro{Tipo: EntradaInvalida, Mensagem: msg} }
func naoEncontrado(msg string) error   { return &Erro{Tipo: NaoEncontrado, Mensagem: msg} }
func indisponivel(msg string) error    { return &Erro{Tipo: Indisponivel, Mensagem: msg} }
