import { extendTheme } from '@mui/joy/styles';

// Tema do Joy UI: fonte Inter e fundo levemente cinzento no modo claro.
export const tema = extendTheme({
  fontFamily: {
    display: "'Inter', var(--joy-fontFamily-fallback)",
    body: "'Inter', var(--joy-fontFamily-fallback)",
  },
  colorSchemes: {
    light: { palette: { background: { body: 'var(--joy-palette-neutral-50)' } } },
  },
});
