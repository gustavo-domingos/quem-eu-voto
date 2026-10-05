import { useEffect, useState } from 'react';
import { Alert, Box, Card, Chip, FormControl, FormLabel, Input, Sheet, Stack, Table, Typography } from '@mui/joy';
import { buscarAPI, mensagemDeErro } from '../../services/api';
import { Avisos, ErroCarregamento, Mensagem } from '../../components/Mensagens';

// [chave, rótulo, função que lê o valor] — colunas numéricas ordenáveis.
const COLUNAS_2026 = [
  ['governador', 'Gov.', (p) => p.candidatos2026.governador || 0],
  ['senador', 'Sen.', (p) => p.candidatos2026.senador || 0],
  ['deputado-federal', 'Dep. fed.', (p) => p.candidatos2026['deputado-federal'] || 0],
  ['deputado-estadual', 'Dep. est.', (p) => p.candidatos2026['deputado-estadual'] || 0],
];

const VALORES = {
  numero: (p) => Number(p.numero) || 999,
  sigla: (p) => p.sigla,
  total: (p) => p.totalCandidatos2026,
  ...Object.fromEntries(COLUNAS_2026.map(([chave, , ler]) => [chave, ler])),
  deputados: (p) => p.deputadosFederais,
  assiduidade: (p) => p.mediaAssiduidade ?? -1,
  projetos: (p) => p.totalProjetos,
  prefeitos: (p) => p.prefeitos2024,
  vereadores: (p) => p.vereadores2024,
};

function Cabecalho({ chave, children, ordem, onOrdenar, titulo }) {
  const ativa = ordem.chave === chave;
  return (
    <th title={titulo} style={{ textAlign: chave === 'sigla' ? 'left' : 'right', width: chave === 'sigla' ? 280 : undefined }}>
      <Box
        component="button"
        onClick={() => onOrdenar(chave)}
        sx={{
          all: 'unset', cursor: 'pointer', fontWeight: 'lg', whiteSpace: 'nowrap',
          color: ativa ? 'primary.plainColor' : 'inherit',
        }}
      >
        {children} {ativa ? (ordem.crescente ? '▲' : '▼') : ''}
      </Box>
    </th>
  );
}

function Partidos() {
  const [dados, setDados] = useState(null);
  const [erro, setErro] = useState(null);
  const [tentativa, setTentativa] = useState(0);
  const [busca, setBusca] = useState('');
  const [ordem, setOrdem] = useState({ chave: 'total', crescente: false });

  useEffect(() => {
    const controller = new AbortController();
    setErro(null);
    buscarAPI('/api/partidos', controller.signal)
      .then(setDados)
      .catch((err) => { if (err.name !== 'AbortError') setErro(mensagemDeErro(err)); });
    return () => controller.abort();
  }, [tentativa]);

  const ordenar = (chave) => setOrdem((o) => ({
    chave,
    // Texto começa em A-Z; números começam do maior para o menor.
    crescente: o.chave === chave ? !o.crescente : chave === 'sigla' || chave === 'numero',
  }));

  if (erro) return <ErroCarregamento mensagem={erro} onTentarDeNovo={() => setTentativa((t) => t + 1)} />;
  if (!dados) return <Mensagem>A carregar partidos…</Mensagem>;

  const termo = busca.trim().toLowerCase();
  const ler = VALORES[ordem.chave];
  const partidos = dados.partidos
    .filter((p) => !termo || `${p.sigla} ${p.nome} ${p.numero} ${p.federacao || ''}`.toLowerCase().includes(termo))
    .sort((a, b) => {
      const va = ler(a);
      const vb = ler(b);
      const cmp = typeof va === 'string' ? va.localeCompare(vb) : va - vb;
      return ordem.crescente ? cmp : -cmp;
    });

  const props = { ordem, onOrdenar: ordenar };
  return (
    <Stack spacing={2.5}>
      <Card variant="outlined">
        <FormControl sx={{ maxWidth: 420 }}>
          <FormLabel>Pesquisar partido</FormLabel>
          <Input
            placeholder="Sigla, nome, número ou federação…"
            startDecorator="🔍"
            value={busca}
            onChange={(e) => setBusca(e.target.value)}
          />
        </FormControl>
      </Card>

      <Stack spacing={1}>
        <Alert variant="soft" color="primary" size="sm" startDecorator="ℹ️">
          <span>
            <b>{partidos.length}</b> partido(s). Candidaturas aptas em {dados.anoEleicao} (TSE), deputados federais em
            exercício com a média de assiduidade e o total de projetos (Câmara), e prefeitos e vereadores eleitos em{' '}
            {dados.anoMunicipal} (TSE). Clique no cabeçalho de uma coluna para ordenar.
          </span>
        </Alert>
        <Avisos avisos={dados.avisos} />
      </Stack>

      <Sheet variant="outlined" sx={{ borderRadius: 'md', overflow: 'auto' }}>
        <Table
          stickyHeader
          hoverRow
          size="sm"
          sx={{
            minWidth: 1100,
            tableLayout: 'auto',
            '& td, & th': { whiteSpace: 'nowrap' },
            '& td:nth-of-type(2), & th:nth-of-type(2)': { whiteSpace: 'normal', minWidth: 240 },
            '& td': { textAlign: 'right', fontVariantNumeric: 'tabular-nums' },
            '& td:nth-of-type(2)': { textAlign: 'left' },
            '& thead tr:first-of-type th': { textAlign: 'center', bgcolor: 'background.level1' },
          }}
        >
          <thead>
            <tr>
              <th colSpan={2} />
              <th colSpan={5}>Candidatos {dados.anoEleicao}</th>
              <th colSpan={3}>Câmara dos Deputados</th>
              <th colSpan={2}>Eleitos {dados.anoMunicipal}</th>
            </tr>
            <tr>
              <Cabecalho chave="numero" {...props}>Nº</Cabecalho>
              <Cabecalho chave="sigla" {...props}>Partido</Cabecalho>
              <Cabecalho chave="total" {...props}>Total</Cabecalho>
              {COLUNAS_2026.map(([chave, rotulo]) => <Cabecalho key={chave} chave={chave} {...props}>{rotulo}</Cabecalho>)}
              <Cabecalho chave="deputados" {...props} titulo="Deputados federais em exercício">Deputados</Cabecalho>
              <Cabecalho chave="assiduidade" {...props} titulo="Média de assiduidade dos deputados do partido">Assid. média</Cabecalho>
              <Cabecalho chave="projetos" {...props} titulo="Soma dos PL, PLP e PEC dos deputados atuais na legislatura">Projetos</Cabecalho>
              <Cabecalho chave="prefeitos" {...props}>Prefeitos</Cabecalho>
              <Cabecalho chave="vereadores" {...props}>Vereadores</Cabecalho>
            </tr>
          </thead>
          <tbody>
            {partidos.map((p) => (
              <tr key={p.numero || p.sigla}>
                <td><Typography fontWeight="lg" color="success">{p.numero || '—'}</Typography></td>
                <td>
                  <Stack direction="row" spacing={1} alignItems="center" useFlexGap flexWrap="wrap">
                    <Typography fontWeight="lg">{p.sigla}</Typography>
                    {p.candidatos2026.presidente > 0 && (
                      <Chip size="sm" variant="soft" color="success" title="Tem candidato(a) à Presidência">⭐ Presidência</Chip>
                    )}
                  </Stack>
                  <Typography level="body-xs">{p.nome}</Typography>
                  {p.federacao && <Typography level="body-xs" color="primary">🤝 {p.federacao}</Typography>}
                </td>
                <td><b>{p.totalCandidatos2026}</b></td>
                {COLUNAS_2026.map(([chave, , lerValor]) => <td key={chave}>{lerValor(p) || '—'}</td>)}
                <td>{p.deputadosFederais || '—'}</td>
                <td>{p.mediaAssiduidade != null ? `${p.mediaAssiduidade.toFixed(1)}%` : '—'}</td>
                <td>{p.totalProjetos || '—'}</td>
                <td>{p.prefeitos2024 || '—'}</td>
                <td>{p.vereadores2024 ? p.vereadores2024.toLocaleString('pt-BR') : '—'}</td>
              </tr>
            ))}
          </tbody>
        </Table>
      </Sheet>
    </Stack>
  );
}

export default Partidos;
