import { Divider, Stack, Typography } from '@mui/joy';

// Lista de vices ou suplentes, comum aos cartões de candidatos e de eleitos.
export function ListaVices({ vices }) {
  if (!vices?.length) return null;
  return (
    <>
      <Divider />
      <Stack spacing={0.25}>
        {vices.map((v) => (
          <Typography key={v.cargo + v.nomeUrna} level="body-sm">
            <Typography textColor="text.tertiary">{v.cargo}:</Typography>{' '}
            <Typography fontWeight="lg">{v.nomeUrna}</Typography>
            {v.partido && <Typography textColor="text.tertiary"> ({v.partido})</Typography>}
          </Typography>
        ))}
      </Stack>
    </>
  );
}
