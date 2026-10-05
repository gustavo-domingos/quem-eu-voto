import { Alert } from '@mui/joy';

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
