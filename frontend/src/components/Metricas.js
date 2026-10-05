import { Box, Divider, LinearProgress, Stack, Typography } from '@mui/joy';

function corAssiduidade(valor) {
  if (valor >= 90) return 'success';
  if (valor >= 75) return 'warning';
  return 'danger';
}

// Indicador de atuação no mandato (assiduidade na Câmara, participação em votações no
// Senado) e projetos apresentados. A cor fica na barra; o número usa a cor de texto.
export function Metricas({ rotulo = 'Assiduidade', percentual, detalhe, projetos, dica }) {
  const tem = percentual != null;
  return (
    <Stack direction="row" spacing={2} sx={{ mt: 'auto', pt: 1.5 }}>
      <Box sx={{ flex: 1, minWidth: 0 }} title={dica || detalhe || 'Sem dados no período'}>
        <Stack direction="row" justifyContent="space-between" alignItems="baseline" spacing={1}>
          <Typography level="body-xs" textTransform="uppercase" fontWeight="lg" letterSpacing="0.05em" noWrap>
            {rotulo}
          </Typography>
          <Typography level="title-md">{tem ? `${percentual.toFixed(1)}%` : '—'}</Typography>
        </Stack>
        <LinearProgress
          determinate
          variant="soft"
          value={tem ? percentual : 0}
          color={tem ? corAssiduidade(percentual) : 'neutral'}
          thickness={6}
          sx={{ my: 0.5 }}
        />
        {detalhe && <Typography level="body-xs">{detalhe}</Typography>}
      </Box>
      <Divider orientation="vertical" />
      <Box sx={{ textAlign: 'center', minWidth: 64 }} title="PL, PLP e PEC de autoria ou coautoria na legislatura atual">
        <Typography level="body-xs" textTransform="uppercase" fontWeight="lg" letterSpacing="0.05em">
          Projetos
        </Typography>
        <Typography level="h3">{projetos ?? '—'}</Typography>
      </Box>
    </Stack>
  );
}
