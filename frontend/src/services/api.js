export const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export const MENSAGEM_ERRO =
  'Não foi possível carregar os dados. Verifique se o backend está em execução e tente novamente.';

// Faz GET à API e devolve o JSON; lança erro se o servidor não responder 200.
export async function buscarAPI(caminho, signal) {
  const res = await fetch(`${API_URL}${caminho}`, { signal });
  if (!res.ok) throw new Error(`O servidor respondeu ${res.status}`);
  return res.json();
}

// Fotos do TSE vêm como caminho do nosso backend; as da Câmara e do Senado já são URLs completos.
export function urlFoto(foto) {
  if (!foto) return null;
  return foto.startsWith('http') ? foto : `${API_URL}${foto}`;
}
