import { useColorScheme } from '@mui/joy/styles';
import {
  Box, Button, IconButton, List, ListItem, ListItemButton, ListItemContent, ListItemDecorator, Stack, Tooltip, Typography,
} from '@mui/joy';
import { ITENS_MENU, PARTIDOS } from './rotas';

export function BotaoTema() {
  const { mode, setMode } = useColorScheme();
  const escuro = mode === 'dark';
  return (
    <IconButton
      variant="plain"
      color="neutral"
      onClick={() => setMode(escuro ? 'light' : 'dark')}
      title={escuro ? 'Mudar para tema claro' : 'Mudar para tema escuro'}
    >
      {escuro ? '☀️' : '🌙'}
    </IconButton>
  );
}

export function Logo() {
  return (
    <Stack direction="row" spacing={1.5} alignItems="center">
      <Box component="img" src={`${process.env.PUBLIC_URL}/favicon.svg`} alt="" sx={{ width: 40, height: 40, display: 'block' }} />
      <Box>
        <Typography level="title-lg" sx={{ lineHeight: 1.1 }}>Quem eu voto</Typography>
        <Typography level="body-xs">Transparência eleitoral</Typography>
      </Box>
    </Stack>
  );
}

export function Menu({ ativo, onEscolher }) {
  return (
    <List size="sm" sx={{ '--ListItem-radius': '8px', '--List-gap': '2px' }}>
      {ITENS_MENU.map((item) => (
        <ListItem key={item.id} sx={item.id === PARTIDOS.id ? { mt: 1.5 } : undefined}>
          <ListItemButton
            selected={ativo === item.id}
            color={ativo === item.id ? 'primary' : 'neutral'}
            variant={ativo === item.id ? 'soft' : 'plain'}
            onClick={() => onEscolher(item.id)}
            sx={{ py: 1, fontWeight: ativo === item.id ? 'lg' : 'md' }}
          >
            <ListItemDecorator sx={{ fontSize: '1.1rem' }}>{item.icone}</ListItemDecorator>
            <ListItemContent>{item.titulo}</ListItemContent>
          </ListItemButton>
        </ListItem>
      ))}
    </List>
  );
}

// Os dois botões no topo da página de cada cargo: quem já foi eleito ou quem concorre em 2026.
export function EscolhaVista({ cargo, vista, onMudar }) {
  const botao = (valor, rotulo, desativado) => {
    const ativo = vista === valor;
    const b = (
      <Button
        size="lg"
        variant={ativo ? 'solid' : 'plain'}
        disabled={desativado}
        onClick={() => onMudar(valor)}
        sx={{
          borderRadius: 'lg',
          flex: { xs: 1, sm: 'initial' },
          ...(ativo
            ? { bgcolor: '#fff', color: 'primary.700', '&:hover': { bgcolor: '#fff' }, boxShadow: 'sm' }
            : { color: '#fff', '&:hover': { bgcolor: 'rgba(255,255,255,0.15)' } }),
          '&.Mui-disabled': { color: 'rgba(255,255,255,0.45)' },
        }}
      >
        {rotulo}
      </Button>
    );
    return desativado ? (
      <Tooltip title="Não há eleição municipal em 2026" variant="soft">
        <Box component="span" sx={{ display: 'flex', flex: { xs: 1, sm: 'initial' } }}>{b}</Box>
      </Tooltip>
    ) : b;
  };
  return (
    <Stack
      direction="row"
      spacing={0.5}
      sx={{ mt: 2.5, p: 0.5, borderRadius: 'xl', bgcolor: 'rgba(0,0,0,0.2)', width: { xs: '100%', sm: 'fit-content' } }}
    >
      {botao('eleitos', '🏛️ Eleitos', false)}
      {botao('candidatos', '🗳️ Candidatos 2026', cargo.municipal)}
    </Stack>
  );
}
