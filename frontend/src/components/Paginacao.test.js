import { fireEvent, render, screen } from '@testing-library/react';
import { Paginacao, paginasVisiveis } from './Paginacao';

test('mostra a primeira, a última e as vizinhas da página atual', () => {
  expect(paginasVisiveis(7, 232)).toEqual([1, '…', 5, 6, 7, 8, 9, '…', 232]);
  expect(paginasVisiveis(1, 232)).toEqual([1, 2, 3, '…', 232]);
  expect(paginasVisiveis(232, 232)).toEqual([1, '…', 230, 231, 232]);
});

test('um salto de uma só página mostra o número em vez de reticências', () => {
  expect(paginasVisiveis(4, 10)).toEqual([1, 2, 3, 4, 5, 6, '…', 10]);
  expect(paginasVisiveis(1, 5)).toEqual([1, 2, 3, 4, 5]);
});

test('clicar num número e usar "Ir para" mudam a página', () => {
  const onMudar = jest.fn();
  render(<Paginacao pagina={7} totalPaginas={232} onMudar={onMudar} />);

  fireEvent.click(screen.getByRole('button', { name: '9' }));
  expect(onMudar).toHaveBeenLastCalledWith(9);

  fireEvent.click(screen.getByRole('button', { name: 'Última página' }));
  expect(onMudar).toHaveBeenLastCalledWith(232);

  fireEvent.change(screen.getByLabelText('Número da página'), { target: { value: '150' } });
  fireEvent.click(screen.getByRole('button', { name: 'Ir' }));
  expect(onMudar).toHaveBeenLastCalledWith(150);
});

test('não deixa ir para uma página que não existe', () => {
  const onMudar = jest.fn();
  render(<Paginacao pagina={1} totalPaginas={20} onMudar={onMudar} />);
  fireEvent.change(screen.getByLabelText('Número da página'), { target: { value: '99' } });
  expect(screen.getByRole('button', { name: 'Ir' })).toBeDisabled();
});
