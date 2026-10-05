import { Alert, Button, Stack, Typography } from '@mui/joy';

export function Avisos({ avisos }) {
  return avisos?.map((aviso) => (
    <Alert key={aviso} color="warning" variant="soft" size="sm" startDecorator="⚠️">{aviso}</Alert>
  ));
}

export function Mensagem({ children, cor = 'neutral' }) {
  return (
    <Alert color={cor} variant="soft" sx={{ justifyContent: 'center', py: 3 }}>
      {children}
    </Alert>
  );
}

// ErroCarregamento mostra o motivo da falha (vindo da API) e, se houver onTentarDeNovo, um botão para repetir.
export function ErroCarregamento({ mensagem, onTentarDeNovo }) {
  return (
    <Alert color="danger" variant="soft" sx={{ justifyContent: 'center', py: 3 }}>
      <Stack spacing={1.5} alignItems="center" sx={{ textAlign: 'center' }}>
        <Typography textColor="inherit" fontWeight="md">⚠️ {mensagem}</Typography>
        {onTentarDeNovo && (
          <Button size="sm" color="danger" variant="solid" onClick={onTentarDeNovo}>Tentar de novo</Button>
        )}
      </Stack>
    </Alert>
  );
}
