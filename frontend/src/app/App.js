import { useState } from 'react';
import { CssVarsProvider } from '@mui/joy/styles';
import CssBaseline from '@mui/joy/CssBaseline';
import { Box, Drawer, IconButton, Sheet, Stack, Typography } from '@mui/joy';
import Candidatos from '../features/candidatos/Candidatos';
import Eleitos from '../features/eleitos/Eleitos';
import Partidos from '../features/partidos/Partidos';
import Perfil from '../features/perfil/Perfil';
import { BotaoTema, EscolhaVista, Logo, Menu } from './Navegacao';
import { CARGOS, DESCRICAO_ELEITOS, PARTIDOS, descricaoCandidatos, useRota } from './rotas';
import { tema } from './tema';

function App() {
  const { rota, navegar: irPara, mudarPerfil } = useRota();
  const [menuAberto, setMenuAberto] = useState(false);
  const cargo = CARGOS.find((c) => c.cargo === rota.id);
  const item = cargo ? { ...cargo, id: cargo.cargo } : PARTIDOS;

  const navegar = (id, vista) => {
    irPara(id, vista);
    setMenuAberto(false);
  };

  let descricao = 'Resumo de cada partido: candidaturas em 2026, bancada na Câmara e eleitos em 2024.';
  let conteudo = <Partidos />;
  if (cargo && rota.vista === 'candidatos') {
    descricao = descricaoCandidatos(cargo.titulo);
    // key: cada cargo/vista começa com os filtros limpos.
    conteudo = <Candidatos key={`c-${cargo.cargo}`} cargo={cargo.cargo} onAbrirPerfil={mudarPerfil} />;
  } else if (cargo) {
    descricao = DESCRICAO_ELEITOS[cargo.cargo];
    conteudo = <Eleitos key={`e-${cargo.cargo}`} cargo={cargo.cargo} onAbrirPerfil={mudarPerfil} />;
  }

  return (
    <CssVarsProvider theme={tema} defaultMode="light">
      <CssBaseline />
      <Box sx={{ display: 'flex', minHeight: '100vh' }}>
        {/* Menu lateral (ecrãs largos) */}
        <Sheet
          component="nav"
          sx={{
            display: { xs: 'none', md: 'flex' },
            flexDirection: 'column',
            gap: 2,
            width: 248,
            flexShrink: 0,
            position: 'sticky',
            top: 0,
            height: '100vh',
            overflowY: 'auto',
            p: 2,
            borderRight: '1px solid',
            borderColor: 'divider',
          }}
        >
          <Stack direction="row" justifyContent="space-between" alignItems="center">
            <Logo />
            <BotaoTema />
          </Stack>
          <Menu ativo={item.id} onEscolher={(id) => navegar(id, rota.vista)} />
          <Typography level="body-xs" sx={{ mt: 'auto' }}>
            Dados oficiais do TSE, da Câmara dos Deputados e do Senado Federal.
          </Typography>
        </Sheet>

        {/* Menu em gaveta (telemóvel) */}
        <Drawer open={menuAberto} onClose={() => setMenuAberto(false)} size="sm">
          <Stack spacing={2} sx={{ p: 2 }}>
            <Logo />
            <Menu ativo={item.id} onEscolher={(id) => navegar(id, rota.vista)} />
          </Stack>
        </Drawer>

        <Box component="main" sx={{ flex: 1, minWidth: 0 }}>
          <Sheet
            sx={{
              display: { xs: 'flex', md: 'none' },
              alignItems: 'center',
              gap: 1,
              px: 1.5,
              py: 1,
              position: 'sticky',
              top: 0,
              zIndex: 10,
              borderBottom: '1px solid',
              borderColor: 'divider',
            }}
          >
            <IconButton variant="outlined" color="neutral" onClick={() => setMenuAberto(true)} aria-label="Abrir menu">
              ☰
            </IconButton>
            <Typography level="title-md" sx={{ flex: 1 }} noWrap>{item.icone} {item.titulo}</Typography>
            <BotaoTema />
          </Sheet>

          <Box
            component="header"
            sx={{
              background: 'linear-gradient(135deg, #0B4A8B 0%, #0B6BCB 50%, #1A7F37 100%)',
              color: '#fff',
              px: { xs: 2, md: 4 },
              pt: { xs: 3, md: 5 },
              pb: { xs: 7, md: 8 },
            }}
          >
            <Typography level="h1" sx={{ color: '#fff', fontSize: { xs: '1.7rem', md: '2.3rem' } }}>
              {item.icone} {item.titulo}
            </Typography>
            <Typography level="body-md" sx={{ color: 'rgba(255,255,255,0.85)', mt: 0.5, maxWidth: 760 }}>
              {descricao}
            </Typography>
            {cargo && <EscolhaVista cargo={cargo} vista={rota.vista} onMudar={(v) => navegar(cargo.cargo, v)} />}
          </Box>

          <Box sx={{ px: { xs: 2, md: 4 }, mt: { xs: -4, md: -5 }, pb: 6, maxWidth: 1400 }}>
            {conteudo}
          </Box>
          <Perfil id={rota.perfil} onFechar={() => mudarPerfil(null)} />
        </Box>
      </Box>
    </CssVarsProvider>
  );
}

export default App;
