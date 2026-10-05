import { Checkbox, Sheet, Typography } from '@mui/joy';

// Faixa vermelha no topo do cartão quando há registos oficiais (CGU, Justiça Eleitoral).
export function FaixaJustica({ justica }) {
  if (!justica?.total) return null;
  return (
    <Sheet
      variant="solid"
      color="danger"
      sx={{ mx: 'calc(-1 * var(--Card-padding))', mt: 'calc(-1 * var(--Card-padding))', mb: 0.5, px: 'var(--Card-padding)', py: 0.75,
        borderTopLeftRadius: 'var(--Card-radius)', borderTopRightRadius: 'var(--Card-radius)' }}
    >
      <Typography level="body-sm" fontWeight="lg" textColor="inherit">
        ⚖️ {justica.corrupcao ? 'Condenação ou sanção por improbidade/corrupção' : 'Registro na Justiça ou órgão de controle'}
        {justica.total > 1 ? ` (${justica.total} registros)` : ''}
      </Typography>
      <Typography level="body-xs" textColor="inherit" sx={{ opacity: 0.9 }}>Veja os detalhes e as fontes no perfil.</Typography>
    </Sheet>
  );
}

export function FiltroJustica({ marcado, onMudar }) {
  return (
    <Checkbox
      size="sm"
      color="danger"
      label="⚖️ Somente quem tem registros na Justiça ou em órgãos de controle (improbidade, sanções, Ficha Limpa, cassações)"
      checked={marcado}
      onChange={(e) => onMudar(e.target.checked)}
    />
  );
}
