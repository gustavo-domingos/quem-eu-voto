import { useEffect, useState } from 'react';
import {
  Alert, Box, Card, Checkbox, Chip, Divider, FormControl, FormLabel, Input, Option, Select, Stack, Typography,
} from '@mui/joy';
import { buscarAPI, mensagemDeErro, urlFoto } from '../../services/api';
import { ESTADOS } from '../../utils/constantes';
import { formatarData } from '../../utils/formatacao';
import { FiltrosPessoa } from '../../components/FiltrosPessoa';
import { Foto } from '../../components/Foto';
import { Grade, GradeCarregando } from '../../components/Grade';
import { FaixaJustica, FiltroJustica } from '../../components/Justica';
import { ListaVices } from '../../components/ListaVices';
import { Avisos, ErroCarregamento, Mensagem } from '../../components/Mensagens';
import { Metricas } from '../../components/Metricas';
import { Paginacao } from '../../components/Paginacao';
import { estiloClicavel, propsClicavel } from '../../components/clicavel';
import { PainelApuracao, ResultadoNaApuracao } from '../apuracao/Apuracao';
import { QuantoElege } from '../vagas/QuantoElege';

// Só mostramos etiqueta de situação quando não é um simples "Deferido".
function corSituacao(c) {
  if (!c.apto) return 'danger';
  if (c.situacao !== 'Deferido') return 'warning';
  return null;
}

const reais = (n) => n.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL', maximumFractionDigits: 0 });
const reaisCentavos = (n) => n.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL', minimumFractionDigits: 2, maximumFractionDigits: 2 });

// Linha com o gasto declarado ao TSE, a parte paga com dinheiro público e o custo por voto.
function GastoNaCampanha({ campanha, votos }) {
  if (!campanha?.despesas) return null;
  const publico = campanha.dinheiroPublico > 0 ? Math.min(100, (campanha.dinheiroPublico / campanha.despesas) * 100) : 0;
  return (
    <Typography level="body-sm" title="Despesas contratadas declaradas ao TSE (prestação parcial)">
      💸 Gastou <b>{reais(campanha.despesas)}</b> na campanha
      {campanha.dinheiroPublico > 0 && ` · ${reais(campanha.dinheiroPublico)} de dinheiro público`}
      {publico >= 99.5 && ' (equivale a todo o gasto)'}
      {votos > 0 && <> · <b>{reaisCentavos(campanha.despesas / votos)}</b> por voto</>}
    </Typography>
  );
}

function CartaoCandidato({ c, onAbrir }) {
  const cor = corSituacao(c);
  return (
    <Card
      variant="outlined"
      {...propsClicavel(() => onAbrir(c.id))}
      sx={{
        ...estiloClicavel,
        gap: 1.5,
        transition: 'box-shadow 0.2s, border-color 0.2s',
        '&:hover': { boxShadow: 'md', borderColor: 'neutral.outlinedHoverBorder' },
        ...(cor === 'danger' && { opacity: 0.75 }),
        ...(c.justica?.total > 0 && { borderColor: 'danger.400' }),
      }}
    >
      <FaixaJustica justica={c.justica} />
      <Stack direction="row" spacing={2} alignItems="center">
        <Foto src={urlFoto(c.foto)} nome={c.nomeUrna} />
        <Box sx={{ minWidth: 0 }}>
          <Typography level="h2" color="success" sx={{ letterSpacing: '0.04em', lineHeight: 1 }}>
            {c.numero}
          </Typography>
          <Typography level="title-lg" sx={{ mt: 0.5 }}>{c.nomeUrna}</Typography>
          <Typography level="body-xs" title={c.nome}>{c.nome}</Typography>
        </Box>
      </Stack>

      <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
        <Chip variant="solid" color="primary" size="sm">{c.partido}</Chip>
        <Chip variant="soft" size="sm">{c.uf === 'BR' ? 'Brasil' : c.uf}</Chip>
        {cor && <Chip variant="soft" color={cor} size="sm">{c.situacao}</Chip>}
      </Stack>

      <ResultadoNaApuracao res={c.apuracao} />
      <GastoNaCampanha campanha={c.campanha} votos={c.apuracao?.votos} />
      {c.patrimonio != null && (
        <Typography level="body-sm" title="Soma dos bens declarados ao TSE no registro da candidatura">
          💰 Bens declarados: <b>{c.patrimonio.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL', maximumFractionDigits: 0 })}</b>
        </Typography>
      )}
      {(c.coligacao || c.ocupacao) && (
        <Box>
          {c.coligacao && <Typography level="body-sm" title="Coligação ou federação">🤝 {c.coligacao}</Typography>}
          {c.ocupacao && <Typography level="body-sm">💼 {c.ocupacao}</Typography>}
        </Box>
      )}

      <ListaVices vices={c.vices} />

      {c.deputadoFederal && (
        <>
          <Divider />
          <Chip variant="soft" color="primary" size="sm" startDecorator="🏛️" sx={{ alignSelf: 'flex-start' }}>
            Deputado(a) federal em exercício
          </Chip>
          <Metricas
            percentual={c.deputadoFederal.assiduidade}
            detalhe={c.deputadoFederal.assiduidade != null
              ? `${c.deputadoFederal.sessoesPresentes}/${c.deputadoFederal.sessoesTotal} sessões` : null}
            projetos={c.deputadoFederal.totalProjetos}
          />
        </>
      )}
      <Typography level="body-xs" color="primary" fontWeight="lg" sx={{ mt: 'auto', pt: 0.5 }}>Ver perfil completo →</Typography>
    </Card>
  );
}

// Lista paginada dos candidatos de 2026 a um cargo.
function Candidatos({ cargo, onAbrirPerfil }) {
  const [uf, setUf] = useState('SC');
  const [busca, setBusca] = useState('');
  const [buscaAplicada, setBuscaAplicada] = useState('');
  // No dia da eleição, a lista começa ordenada pela apuração.
  const [ordem, setOrdem] = useState('apuracao');
  const [inaptos, setInaptos] = useState(false);
  const [soJustica, setSoJustica] = useState(false);
  const [partido, setPartido] = useState('');
  const [genero, setGenero] = useState('');
  const [faixa, setFaixa] = useState('');
  const [pagina, setPagina] = useState(1);

  const [resultado, setResultado] = useState(null);
  const [loading, setLoading] = useState(true);
  const [erro, setErro] = useState(null);
  const [tentativa, setTentativa] = useState(0);

  // Qualquer mudança de filtro volta à primeira página.
  const filtro = (setter) => (valor) => {
    setter(valor);
    setPagina(1);
  };

  // Espera o utilizador parar de escrever antes de pesquisar no servidor.
  useEffect(() => {
    const t = setTimeout(() => {
      if (busca.trim() !== buscaAplicada) {
        setBuscaAplicada(busca.trim());
        setPagina(1);
      }
    }, 300);
    return () => clearTimeout(t);
  }, [busca, buscaAplicada]);

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setErro(null);

    const params = new URLSearchParams({
      cargo, uf, q: buscaAplicada, ordem, pagina, inaptos: inaptos ? '1' : '0', justica: soJustica ? '1' : '0', partido, genero, faixa,
    });
    buscarAPI(`/api/candidatos?${params}`, controller.signal)
      .then((data) => {
        setResultado(data);
        setLoading(false);
      })
      .catch((err) => {
        if (err.name === 'AbortError') return;
        console.error('Erro ao procurar candidatos:', err);
        setErro(mensagemDeErro(err));
        setResultado(null);
        setLoading(false);
      });

    return () => controller.abort();
  }, [cargo, uf, buscaAplicada, ordem, inaptos, soJustica, partido, genero, faixa, pagina, tentativa]);

  const mudarPagina = (p) => {
    setPagina(p);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  return (
    <Stack spacing={2.5}>
      <Card variant="outlined">
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} useFlexGap flexWrap="wrap">
          {cargo !== 'presidente' && (
            <FormControl sx={{ flex: 1, minWidth: 200 }}>
              <FormLabel>Estado</FormLabel>
              <Select value={uf} onChange={(_, v) => filtro(setUf)(v)}>
                <Option value="TODOS">🌎 Todos os Estados</Option>
                {ESTADOS.map(([sigla, nome]) => <Option key={sigla} value={sigla}>{nome} ({sigla})</Option>)}
              </Select>
            </FormControl>
          )}

          <FormControl sx={{ flex: 1.5, minWidth: 220 }}>
            <FormLabel>Pesquisar</FormLabel>
            <Input
              placeholder="Nome, partido ou número…"
              startDecorator="🔍"
              value={busca}
              onChange={(e) => setBusca(e.target.value)}
            />
          </FormControl>

          <FormControl sx={{ flex: 1, minWidth: 180 }}>
            <FormLabel>Ordenar por</FormLabel>
            <Select value={ordem} onChange={(_, v) => filtro(setOrdem)(v)}>
              <Option value="apuracao">📊 Mais votados na apuração</Option>
              <Option value="campanha">💸 Maior gasto de campanha</Option>
              <Option value="campanha-asc">Menor gasto de campanha</Option>
              <Option value="custo-voto-asc">Menor custo por voto</Option>
              <Option value="custo-voto">Maior custo por voto</Option>
              <Option value="nome">Nome na urna (A-Z)</Option>
              <Option value="numero">Número</Option>
              <Option value="partido">Partido</Option>
              <Option value="patrimonio">Maior patrimônio declarado</Option>
              <Option value="patrimonio-asc">Menor patrimônio declarado</Option>
              <Option value="idade-asc">Mais novos</Option>
              <Option value="idade">Mais velhos</Option>
            </Select>
          </FormControl>
        </Stack>
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} useFlexGap flexWrap="wrap" sx={{ mt: 1 }}>
          <FiltrosPessoa
            partidos={resultado?.partidos}
            partido={partido} genero={genero} faixa={faixa}
            onPartido={filtro(setPartido)} onGenero={filtro(setGenero)} onFaixa={filtro(setFaixa)}
          />
        </Stack>
        <Checkbox
          size="sm"
          label="Mostrar também candidaturas inaptas (indeferidas, renúncias, fora da urna)"
          checked={inaptos}
          onChange={(e) => filtro(setInaptos)(e.target.checked)}
        />
        <FiltroJustica marcado={soJustica} onMudar={filtro(setSoJustica)} />
      </Card>

      <QuantoElege uf={cargo === 'presidente' ? 'TODOS' : uf} cargo={cargo} modo="candidatos" />
      <PainelApuracao cargo={cargo} uf={uf} />

      {resultado && !erro && (
        <Stack spacing={1}>
          <Alert variant="soft" color="primary" size="sm" startDecorator="ℹ️">
            <span>
              <b>{resultado.total}</b> candidatura(s) · Dados Abertos do TSE, eleições {resultado.anoEleicao}
              {resultado.candidatosAtualizadosEm && ` (atualizado em ${formatarData(resultado.candidatosAtualizadosEm)})`}.
              Para deputados federais em exercício, mostramos também a assiduidade e os projetos desde{' '}
              {formatarData(resultado.inicioPeriodo)}.
            </span>
          </Alert>
          <Avisos avisos={resultado.avisos} />
        </Stack>
      )}

      {loading && !resultado ? (
        <GradeCarregando />
      ) : erro ? (
        <ErroCarregamento mensagem={erro} onTentarDeNovo={() => setTentativa((t) => t + 1)} />
      ) : resultado.candidatos.length === 0 ? (
        <Mensagem>Nenhum candidato encontrado.</Mensagem>
      ) : (
        <>
          <Grade atualizando={loading}>
            {resultado.candidatos.map((c) => <CartaoCandidato key={c.id} c={c} onAbrir={onAbrirPerfil} />)}
          </Grade>
          <Paginacao pagina={resultado.pagina} totalPaginas={resultado.totalPaginas} desativado={loading} onMudar={mudarPagina} />
        </>
      )}
    </Stack>
  );
}

export default Candidatos;
