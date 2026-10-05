export const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

const MENSAGEM_PADRAO = 'Não foi possível carregar os dados. Tente novamente em instantes.';

// ErroAPI traz a mensagem para mostrar a quem usa o site e o código HTTP (0 = sem conexão com a API).
export class ErroAPI extends Error {
  constructor(mensagem, status) {
    super(mensagem);
    this.name = 'ErroAPI';
    this.status = status;
  }
}

// Lê a mensagem do corpo de erro da API ({"erro": "...", "status": 502}); aceita texto simples.
async function mensagemDaResposta(res) {
  const texto = (await res.text().catch(() => '')).trim();
  try {
    const corpo = JSON.parse(texto);
    if (corpo && typeof corpo.erro === 'string' && corpo.erro) return corpo.erro;
  } catch {
    // não era JSON
  }
  if (texto && texto.length <= 300 && !texto.startsWith('<')) return texto;
  return `O servidor respondeu com o código ${res.status}.`;
}

// Faz GET à API e devolve o JSON. Em caso de falha lança ErroAPI com uma mensagem legível;
// cancelamentos (AbortError) passam sem alteração.
export async function buscarAPI(caminho, signal) {
  let res;
  try {
    res = await fetch(`${API_URL}${caminho}`, { signal });
  } catch (err) {
    if (err.name === 'AbortError') throw err;
    throw new ErroAPI(
      `Não foi possível conectar à API (${API_URL}). Verifique se o backend está rodando e sua conexão com a internet.`,
      0,
    );
  }
  if (!res.ok) throw new ErroAPI(await mensagemDaResposta(res), res.status);
  return res.json();
}

// mensagemDeErro devolve o texto a mostrar para um erro de buscarAPI.
export function mensagemDeErro(err, padrao = MENSAGEM_PADRAO) {
  return err instanceof ErroAPI ? err.message : padrao;
}

// Fotos do TSE vêm como caminho do nosso backend; as da Câmara e do Senado já são URLs completos.
export function urlFoto(foto) {
  if (!foto) return null;
  return foto.startsWith('http') ? foto : `${API_URL}${foto}`;
}
