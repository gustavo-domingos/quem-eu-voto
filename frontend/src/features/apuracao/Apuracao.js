import { useEffect, useState } from 'react';
import { Alert, Box, Button, Card, Chip, LinearProgress, Stack, Typography } from '@mui/joy';
import { buscarAPI, mensagemDeErro } from '../../services/api';

const numero = (n) => n.toLocaleString('pt-BR');
const pct = (n, casas = 2) => `${n.toLocaleString('pt-BR', { minimumFractionDigits: casas, maximumFractionDigits: casas })}%`;

// Cargos em que só um (ou dois, no Senado) é eleito: mostramos todos os candidatos.
const MAJORITARIOS = ['presidente', 'governador', 'senador'];
const ATUALIZAR_A_CADA = 60_000;

function corSituacao(res) {
  if (res.eleito) return 'success';
  if (/2º turno/i.test(res.situacao || '')) return 'warning';
  return 'neutral';
}

// Linha curta com o resultado de um candidato (cartões e perfil).
export function ResultadoNaApuracao({ res }) {
  if (!res) return null;
  return (
    <Stack direction="row" spacing={1} alignItems="center" useFlexGap flexWrap="wrap">
      <Typography level="body-sm">
        📊 <b>{res.posicao ? `${res.posicao}º` : '—'}</b> · {numero(res.votos)} votos ({pct(res.percentual)} dos válidos)
      </Typography>
      {res.situacao && <Chip size="sm" variant="soft" color={corSituacao(res)}>{res.situacao}</Chip>}
      {res.destino && <Chip size="sm" variant="soft" color="warning" title="Votos de candidatura sub judice">{res.destino}</Chip>}
    </Stack>
  );
}

// Painel com a apuração ao vivo de um cargo numa UF (ou no Brasil, para Presidente).
export function PainelApuracao({ cargo, uf }) {
  const [dados, setDados] = useState(null);
  const [erro, setErro] = useState(null);
  const [todos, setTodos] = useState(false);
  const [tick, setTick] = useState(0);
  const nacional = cargo === 'presidente';
  const semUF = !nacional && uf === 'TODOS';

  useEffect(() => {
    setDados(null);
    setTodos(false);
  }, [cargo, uf]);

  useEffect(() => {
    if (semUF) return undefined;
    const controller = new AbortController();
    buscarAPI(`/api/apuracao?cargo=${cargo}&uf=${nacional ? 'BR' : uf}`, controller.signal)
      .then((d) => { setDados(d); setErro(null); })
      .catch((err) => {
        if (err.name === 'AbortError') return;
        // Sem conexão com a API, a página já mostra o erro principal; o painel fica em silêncio.
        if (err.status === 0) { setErro(null); return; }
        setErro(`${mensagemDeErro(err, 'A apuração do TSE não respondeu agora').replace(/\.$/, '')}. Tentamos de novo em 1 minuto.`);
      });
    return () => controller.abort();
  }, [cargo, uf, nacional, semUF, tick]);

  // Atualiza sozinho a cada minuto enquanto a contagem não terminar.
  useEffect(() => {
    if (semUF || dados?.totalizada) return undefined;
    const t = setInterval(() => setTick((n) => n + 1), ATUALIZAR_A_CADA);
    return () => clearInterval(t);
  }, [semUF, dados?.totalizada]);

  if (semUF) {
    return (
      <Alert variant="soft" color="primary" startDecorator="📊">
        Escolha um estado para ver a apuração ao vivo deste cargo.
      </Alert>
    );
  }
  if (!dados) {
    return erro ? <Alert variant="soft" color="warning">{erro}</Alert> : null;
  }

  const majoritario = MAJORITARIOS.includes(cargo);
  const lista = dados.candidatos.filter((c) => c.votos > 0 || majoritario);
  const visiveis = majoritario || todos ? lista : lista.slice(0, 10);
  const maior = Math.max(...lista.map((c) => c.percentual), 1);

  return (
    <Card variant="outlined" sx={{ gap: 1.5 }}>
      <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" spacing={1}>
        <Box>
          <Typography level="title-lg">
            📊 Apuração {dados.totalizada ? 'encerrada' : 'ao vivo'} · {dados.turno}º turno
            {' '}<Typography level="body-sm" textColor="text.tertiary">({nacional ? 'Brasil' : uf})</Typography>
          </Typography>
          <Typography level="body-xs">
            Fonte: TSE (resultados.tse.jus.br) · atualizado em {dados.atualizadoEm}
            {!dados.totalizada && ' · esta página atualiza sozinha a cada minuto'}
          </Typography>
        </Box>
        {!dados.totalizada && <Chip color="danger" variant="soft" sx={{ alignSelf: 'flex-start' }}>● Ao vivo</Chip>}
      </Stack>

      <Box>
        <Stack direction="row" justifyContent="space-between">
          <Typography level="body-sm"><b>{pct(dados.secoesApuradas)}</b> das urnas apuradas</Typography>
          <Typography level="body-xs">
            Comparecimento {pct(dados.comparecimento)} · brancos {pct(dados.brancos)} · nulos {pct(dados.nulos)}
          </Typography>
        </Stack>
        <LinearProgress determinate variant="soft" value={dados.secoesApuradas} thickness={8} sx={{ mt: 0.5 }} />
      </Box>

      {erro && <Alert size="sm" variant="soft" color="warning">{erro}</Alert>}

      <Stack spacing={0.75}>
        {visiveis.map((c) => (
          <Box
            key={c.sq}
            title={`${c.nomeUrna} (${c.partido}): ${numero(c.votos)} votos`}
            sx={{ display: 'grid', gridTemplateColumns: { xs: '28px 1fr', sm: '28px minmax(180px, 32%) 1fr' }, gap: 1, alignItems: 'center' }}
          >
            <Typography level="body-sm" fontWeight="lg" textAlign="right">{c.posicao || '—'}º</Typography>
            <Box sx={{ minWidth: 0 }}>
              <Typography level="body-sm" fontWeight="lg" noWrap>
                {c.nomeUrna} <Typography textColor="text.tertiary" fontWeight="md">{c.numero} · {c.partido}</Typography>
              </Typography>
              {c.vices?.length > 0 && <Typography level="body-xs" noWrap>{c.vices.map((v) => `${v.cargo}: ${v.nomeUrna}`).join(' · ')}</Typography>}
            </Box>
            <Stack direction="row" spacing={1} alignItems="center" sx={{ gridColumn: { xs: '2', sm: 'auto' }, minWidth: 0 }}>
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Box sx={{ height: 12, width: `${(c.percentual / maior) * 100}%`, minWidth: c.votos > 0 ? 2 : 0, bgcolor: 'primary.500', borderRadius: '0 4px 4px 0' }} />
              </Box>
              <Typography level="body-sm" fontWeight="lg" sx={{ minWidth: 64, textAlign: 'right' }}>{pct(c.percentual)}</Typography>
              <Typography level="body-xs" sx={{ minWidth: 90, textAlign: 'right', display: { xs: 'none', md: 'block' } }}>{numero(c.votos)}</Typography>
              {c.situacao && <Chip size="sm" variant="soft" color={corSituacao(c)}>{c.situacao}</Chip>}
            </Stack>
          </Box>
        ))}
      </Stack>
      {!majoritario && lista.length > 10 && (
        <Button size="sm" variant="plain" onClick={() => setTodos(!todos)} sx={{ alignSelf: 'flex-start' }}>
          {todos ? 'Mostrar só os 10 mais votados' : `Ver os ${lista.length} candidatos com votos`}
        </Button>
      )}
      <Typography level="body-xs">
        Resultados parciais até o fim da totalização: as posições ainda podem mudar. Percentuais sobre os votos válidos.
      </Typography>
    </Card>
  );
}
