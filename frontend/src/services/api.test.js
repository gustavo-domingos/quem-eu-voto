import { buscarAPI, ErroAPI, mensagemDeErro } from './api';

function resposta(status, corpo) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    text: () => Promise.resolve(corpo),
    json: () => Promise.resolve(JSON.parse(corpo)),
  });
}

afterEach(() => {
  delete global.fetch;
});

test('devolve o JSON quando a API responde 200', async () => {
  global.fetch = jest.fn(() => resposta(200, '{"total": 3}'));
  await expect(buscarAPI('/api/x')).resolves.toEqual({ total: 3 });
});

test('usa a mensagem do corpo de erro da API', async () => {
  global.fetch = jest.fn(() => resposta(502, '{"erro":"Dados de candidaturas do TSE indisponíveis no momento","status":502}'));
  const err = await buscarAPI('/api/candidatos').catch((e) => e);
  expect(err).toBeInstanceOf(ErroAPI);
  expect(err.status).toBe(502);
  expect(mensagemDeErro(err)).toBe('Dados de candidaturas do TSE indisponíveis no momento');
});

test('aceita erro em texto simples', async () => {
  global.fetch = jest.fn(() => resposta(405, 'Method Not Allowed\n'));
  const err = await buscarAPI('/api/x').catch((e) => e);
  expect(err.message).toBe('Method Not Allowed');
});

test('explica quando não consegue conectar à API', async () => {
  global.fetch = jest.fn(() => Promise.reject(new TypeError('Failed to fetch')));
  const err = await buscarAPI('/api/x').catch((e) => e);
  expect(err.status).toBe(0);
  expect(err.message).toMatch(/Não foi possível conectar à API/);
});

test('cancelamentos não viram mensagem de erro', async () => {
  const abort = Object.assign(new Error('aborted'), { name: 'AbortError' });
  global.fetch = jest.fn(() => Promise.reject(abort));
  await expect(buscarAPI('/api/x')).rejects.toBe(abort);
});

test('mensagemDeErro usa o texto padrão para erros que não vêm da API', () => {
  expect(mensagemDeErro(new Error('x'), 'padrão')).toBe('padrão');
});
