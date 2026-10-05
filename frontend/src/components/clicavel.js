// Torna um cartão clicável com o rato e com o teclado (Enter/Espaço).
export function propsClicavel(abrir) {
  return {
    role: 'button',
    tabIndex: 0,
    onClick: abrir,
    onKeyDown: (e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        abrir();
      }
    },
  };
}

export const estiloClicavel = {
  cursor: 'pointer',
  '&:focus-visible': { outline: '2px solid', outlineColor: 'primary.500', outlineOffset: 2 },
};
