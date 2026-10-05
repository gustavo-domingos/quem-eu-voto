import { useEffect, useState } from 'react';
import { Box, Card, Sheet, Typography } from '@mui/joy';
import { buscarAPI } from '../../services/api';
import { ESTADOS } from '../../utils/constantes';

const nomeUF = (uf) => (ESTADOS.find(([s]) => s === uf) || [uf, uf])[1];

function Quadro({ rotulo, valor, detalhe, ativo }) {
  return (
    <Sheet
      variant={ativo ? 'solid' : 'soft'}
      color={ativo ? 'primary' : 'neutral'}
      sx={{ borderRadius: 'md', px: 1.5, py: 1, minWidth: 0 }}
    >
      <Typography level="h3" textColor="inherit">{valor}</Typography>
      <Typography level="body-sm" textColor="inherit" fontWeight="lg">{rotulo}</Typography>
      {detalhe && <Typography level="body-xs" textColor="inherit" sx={{ opacity: 0.85 }}>{detalhe}</Typography>}
    </Sheet>
  );
}

// Quantos representantes o local elege (modo "candidatos", vagas em disputa em 2026) ou
// quantas cadeiras tem (modo "eleitos"). O cargo da página aparece em destaque.
export function QuantoElege({ uf, municipio, cargo, modo }) {
  const [v, setV] = useState(null);

  useEffect(() => {
    const controller = new AbortController();
    const params = new URLSearchParams({ uf: uf || 'TODOS', municipio: municipio || 'TODOS' });
    buscarAPI(`/api/vagas?${params}`, controller.signal)
      .then(setV)
      .catch(() => setV(null));
    return () => controller.abort();
  }, [uf, municipio]);

  if (!v) return null;
  const nacional = v.local === 'BR';
  const lugar = nacional ? 'O Brasil' : nomeUF(v.local);
  const distrital = v.local === 'DF';
  const g = v.vagas2026;
  const eleitos = modo === 'eleitos';

  const quadros = [];
  if (eleitos) {
    if (nacional) quadros.push(['presidente', 'Presidente', 1]);
    quadros.push(['governador', nacional ? 'Governadores' : 'Governador', g.governador]);
    quadros.push(['senador', 'Senadores', v.cadeirasSenado, `${g.senador} vagas em disputa em 2026`]);
  } else {
    quadros.push(['presidente', 'Presidente', g.presidente, 'eleição nacional']);
    quadros.push(['governador', nacional ? 'Governadores' : 'Governador', g.governador]);
    quadros.push(['senador', 'Senadores', g.senador, `de ${v.cadeirasSenado} cadeiras; a outra ${nacional ? 'parte' : 'vaga'} é renovada em 2030`]);
  }
  quadros.push(['deputado-federal', 'Deputados federais', g['deputado-federal'], nacional ? 'na Câmara dos Deputados' : 'na Câmara dos Deputados (bancada do estado)']);
  quadros.push(['deputado-estadual', distrital ? 'Deputados distritais' : nacional ? 'Deputados estaduais e distritais' : 'Deputados estaduais',
    g['deputado-estadual'], distrital ? 'na Câmara Legislativa do DF' : 'nas Assembleias Legislativas']);
  if (v.municipais) {
    quadros.push(['prefeito', 'Prefeito', v.municipais.prefeito, 'eleito em 2024']);
    quadros.push(['vereador', 'Vereadores', v.municipais.vereador, 'eleitos em 2024']);
  }

  return (
    <Card variant="outlined" sx={{ gap: 1 }}>
      <Typography level="title-md">
        {eleitos
          ? `🏛️ Quantos representantes ${nacional ? 'o Brasil' : nomeUF(v.local)} tem`
          : `🗳️ Quantos ${lugar} elege em 2026`}
      </Typography>
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))', gap: 1 }}>
        {quadros.filter(([, , valor]) => valor != null).map(([chave, rotulo, valor, detalhe]) => (
          <Quadro key={chave} rotulo={rotulo} valor={valor.toLocaleString('pt-BR')} detalhe={detalhe} ativo={chave === cargo} />
        ))}
      </Box>
      <Typography level="body-xs">Fonte: TSE (vagas por cargo). Cada estado tem 3 senadores; em cada eleição renovam-se 1 ou 2.</Typography>
    </Card>
  );
}
