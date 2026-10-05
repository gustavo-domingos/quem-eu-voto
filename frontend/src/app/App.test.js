import { fireEvent, render, screen } from '@testing-library/react';
import App from './App';

beforeEach(() => {
  // O jsdom não implementa matchMedia, usado pelo Joy UI para o tema claro/escuro.
  window.matchMedia = window.matchMedia || (() => ({
    matches: false, addListener() {}, removeListener() {}, addEventListener() {}, removeEventListener() {},
  }));
  // O jsdom também não implementa scrollTo (usado ao trocar de página/secção).
  window.scrollTo = jest.fn();
  // Pedidos à API ficam pendentes: aqui só testamos a estrutura da página.
  global.fetch = jest.fn(() => new Promise(() => {}));
  window.location.hash = '';
});

test('mostra os cargos no menu lateral e os botões Eleitos / Candidatos', () => {
  render(<App />);
  for (const nome of ['Governador', 'Senador', 'Deputado Federal', 'Deputado Estadual', 'Prefeito', 'Vereador', 'Partidos políticos']) {
    expect(screen.getAllByText(nome).length).toBeGreaterThan(0);
  }
  expect(screen.getByRole('button', { name: /Eleitos/ })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: /Candidatos 2026/ })).toBeInTheDocument();
});

test('abre nos eleitos para Presidente', () => {
  render(<App />);
  expect(global.fetch).toHaveBeenCalledWith(
    expect.stringContaining('/api/eleitos?cargo=presidente'),
    expect.anything(),
  );
});

test('o botão Candidatos 2026 mostra os candidatos do cargo', () => {
  window.location.hash = '#senador';
  render(<App />);
  fireEvent.click(screen.getByRole('button', { name: /Candidatos 2026/ }));
  expect(global.fetch).toHaveBeenCalledWith(
    expect.stringContaining('/api/candidatos?cargo=senador'),
    expect.anything(),
  );
});

test('abre o perfil indicado no endereço', () => {
  window.location.hash = '#deputado-federal?p=camara-220556';
  render(<App />);
  expect(global.fetch).toHaveBeenCalledWith(expect.stringContaining('/api/perfil?id=camara-220556'), expect.anything());
});

test('em Vereador não há candidatos em 2026', () => {
  window.location.hash = '#vereador/candidatos';
  render(<App />);
  expect(screen.getByRole('button', { name: /Candidatos 2026/ })).toBeDisabled();
  expect(global.fetch).toHaveBeenCalledWith(expect.stringContaining('/api/eleitos?cargo=vereador'), expect.anything());
});
