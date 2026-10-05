import { useEffect, useState } from 'react';
import {
  Alert, Box, Card, Chip, Divider, FormControl, FormLabel, Input, Option, Select, Sheet, Stack, Typography,
} from '@mui/joy';
import { buscarAPI, mensagemDeErro, urlFoto } from '../../services/api';
import { ESTADOS } from '../../utils/constantes';
import { FiltrosPessoa } from '../../components/FiltrosPessoa';
import { Foto } from '../../components/Foto';
import { Grade, GradeCarregando } from '../../components/Grade';
import { FaixaJustica, FiltroJustica } from '../../components/Justica';
import { ListaVices } from '../../components/ListaVices';
import { Avisos, ErroCarregamento, Mensagem } from '../../components/Mensagens';
import { Metricas } from '../../components/Metricas';
import { Paginacao } from '../../components/Paginacao';
import { SeletorMunicipio } from '../../components/SeletorMunicipio';
import { estiloClicavel, propsClicavel } from '../../components/clicavel';
import { QuantoElege } from '../vagas/QuantoElege';

const MUNICIPAIS = ['prefeito', 'vereador'];
// Cargos com dados de atuação no mandato (Câmara e Senado) e o nome do indicador em cada um.
const COM_DESEMPENHO = { 'deputado-federal': 'assiduidade', senador: 'participação em votações' };

const reaisCompacto = (n) => n.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL', notation: 'compact', maximumFractionDigits: 1 });

// Linha de números extra sob as métricas: índice de atuação, votações, alinhamento e gastos.
function ExtrasDesempenho({ d, custo }) {
  const itens = [];
  if (d.indiceAtuacao != null) itens.push(['Índice de atuação', `${d.indiceAtuacao}/100`, 'Média dos percentis de participação, projetos e votações entre colegas do mesmo cargo']);
  if (d.votacoes != null) itens.push(['Votações', numero(d.votacoes), 'Votações nominais em que registou voto']);
  if (d.alinhamentoGoverno != null) itens.push(['Com o Governo', pct(d.alinhamentoGoverno, 0), 'Votou como o Governo orientou (votações com orientação Sim/Não)']);
  if (d.gastos != null) itens.push(['Cota gasta', reaisCompacto(d.gastos), 'Cota parlamentar desde 2023']);
  if (custo != null) itens.push(['Custo do mandato', reaisCompacto(custo), 'Subsídio estimado + cota parlamentar, desde 2023 (ver o perfil)']);
  if (!itens.length) return null;
  return (
    <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(90px, 1fr))', gap: 1 }}>
      {itens.map(([rotulo, valor, dica]) => (
        <Box key={rotulo} title={dica}>
          <Typography level="body-xs">{rotulo}</Typography>
          <Typography level="title-sm">{valor}</Typography>
        </Box>
      ))}
    </Box>
  );
}

const numero = (n) => n.toLocaleString('pt-BR');
const pct = (n, casas = 1) => `${n.toLocaleString('pt-BR', { minimumFractionDigits: casas, maximumFractionDigits: casas })}%`;

// ---------- Resumo (estatísticas do conjunto filtrado) ----------

function Indicador({ rotulo, valor, detalhe, cor }) {
  return (
    <Sheet variant="soft" color={cor} sx={{ borderRadius: 'md', p: 1.5, minWidth: 0 }}>
      <Typography level="body-xs" fontWeight="lg">{rotulo}</Typography>
      <Typography level="h3" sx={{ mt: 0.25 }}>{valor}</Typography>
      {detalhe && <Typography level="body-xs">{detalhe}</Typography>}
    </Sheet>
  );
}

// Barras horizontais de uma só série: a cor identifica a série; rótulos e valores usam cores de texto.
function Barras({ titulo, itens, total }) {
  if (!itens?.length) return null;
  const maior = Math.max(...itens.map((i) => i.total), 1);
  return (
    <Box sx={{ minWidth: 0 }}>
      <Typography level="title-sm" sx={{ mb: 1 }}>{titulo}</Typography>
      <Stack spacing={0.75}>
        {itens.map((i) => (
          <Box
            key={i.rotulo}
            title={`${i.rotulo}: ${numero(i.total)} (${pct(total ? (i.total / total) * 100 : 0)})`}
            sx={{ display: 'grid', gridTemplateColumns: 'minmax(90px, 38%) 1fr', alignItems: 'center', gap: 1 }}
          >
            <Typography level="body-xs" noWrap>{i.rotulo}</Typography>
            <Stack direction="row" alignItems="center" spacing={0.75} sx={{ minWidth: 0 }}>
              <Box
                sx={{
                  height: 10,
                  width: `${(i.total / maior) * 100}%`,
                  minWidth: i.total > 0 ? 2 : 0,
                  bgcolor: 'primary.500',
                  borderRadius: '0 4px 4px 0',
                }}
              />
              <Typography level="body-xs" fontWeight="lg" sx={{ flexShrink: 0 }}>{numero(i.total)}</Typography>
            </Stack>
          </Box>
        ))}
      </Stack>
    </Box>
  );
}

function Resumo({ resumo, cargo, ano2026 }) {
  const r = resumo;
  if (!r || r.total === 0) return null;
  const municipal = MUNICIPAIS.includes(cargo);
  return (
    <Card variant="outlined" sx={{ gap: 2 }}>
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: 1.5 }}>
        <Indicador rotulo="Eleitos" valor={numero(r.total)} />
        {r.comGenero > 0 && (
          <Indicador rotulo="Mulheres" valor={pct((r.mulheres / r.comGenero) * 100, 0)} detalhe={`${numero(r.mulheres)} de ${numero(r.comGenero)}`} />
        )}
        {r.idadeMedia != null && <Indicador rotulo="Idade média" valor={`${Math.round(r.idadeMedia)} anos`} />}
        <Indicador
          rotulo={`Concorrem em ${ano2026}`}
          valor={pct((r.candidatos2026 / r.total) * 100, 0)}
          detalhe={municipal
            ? `${numero(r.candidatos2026)} a outros cargos`
            : `${numero(r.reeleicao)} à reeleição · ${numero(r.outroCargo)} a outro cargo`}
        />
        <Indicador
          rotulo="Com registros na Justiça"
          valor={numero(r.comRegistros)}
          detalhe={r.comRegistros > 0 ? 'improbidade, sanções da CGU, Ficha Limpa ou cassação' : 'nenhum encontrado nos cadastros oficiais'}
          cor={r.comRegistros > 0 ? 'danger' : undefined}
        />
        {r.mediaDesempenho != null && (
          <Indicador rotulo={`${r.rotuloDesempenho} média`} valor={pct(r.mediaDesempenho)} />
        )}
        {r.totalProjetos != null && (
          <Indicador rotulo="Projetos apresentados" valor={numero(r.totalProjetos)} detalhe="PL, PLP e PEC desde 2023" />
        )}
      </Box>
      {r.total > 1 && (
        <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: 3 }}>
          <Barras titulo="Por partido" itens={r.porPartido} total={r.total} />
          <Barras titulo="Faixa etária" itens={r.faixaEtaria} total={r.total} />
          <Barras titulo="Escolaridade" itens={r.escolaridade?.slice(0, 6)} total={r.total} />
        </Box>
      )}
    </Card>
  );
}

// ---------- Cartão de cada eleito ----------

function Votacao({ e }) {
  const v = e.votacao;
  if (v) {
    return (
      <Typography level="body-sm">
        🗳️ <b>{numero(v.v)}</b> votos em {e.anoEleicao}{v.t === 2 ? ' (2º turno)' : ''}
        {' · '}{pct(v.pc)} dos válidos · {v.p}º de {numero(v.d)}
      </Typography>
    );
  }
  if (e.situacao === 'Suplente em exercício') {
    return <Typography level="body-sm">🔁 Assumiu a vaga como suplente</Typography>;
  }
  if (e.anoEleicao) {
    return <Typography level="body-sm">🗳️ Eleito(a) em {e.anoEleicao}</Typography>;
  }
  return null;
}

function Em2026({ e, ano }) {
  const aptas = e.candidaturas2026.filter((c) => c.apto);
  const inaptas = e.candidaturas2026.filter((c) => !c.apto);
  if (aptas.length === 0 && inaptas.length === 0) {
    return (
      <Sheet variant="soft" sx={{ borderRadius: 'md', px: 1.5, py: 1, textAlign: 'center' }}>
        <Typography level="body-sm">Não é candidato(a) em {ano}</Typography>
      </Sheet>
    );
  }
  return [...aptas, ...inaptas].map((c) => {
    const reeleicao = c.apto && c.cargo === e.cargo;
    const cor = !c.apto ? 'danger' : reeleicao ? 'success' : 'primary';
    return (
      <Sheet key={c.id} variant="soft" color={cor} sx={{ borderRadius: 'md', px: 1.5, py: 1, display: 'flex', alignItems: 'center', gap: 1.5 }}>
        <Typography level="h4" sx={{ minWidth: 48 }}>{c.numero}</Typography>
        <Box sx={{ minWidth: 0 }}>
          <Typography level="body-sm">
            {reeleicao ? 'Busca a reeleição' : `Em ${ano}, concorre a`} {!reeleicao && <b>{c.cargo}</b>} ({c.uf === 'BR' ? 'Brasil' : c.uf}) · {c.partido}
          </Typography>
          <Typography level="body-xs">Na urna: {c.nomeUrna}{!c.apto && ` · ${c.situacao}`}</Typography>
        </Box>
      </Sheet>
    );
  });
}

function CartaoEleito({ e, ano, onAbrir }) {
  const local = e.municipio ? `${e.municipio} - ${e.uf}` : e.uf === 'BR' ? 'Brasil' : e.uf;
  return (
    <Card
      variant="outlined"
      {...propsClicavel(() => onAbrir(e.id))}
      sx={{ ...estiloClicavel, ...(e.justica?.total > 0 && { borderColor: 'danger.400' }), gap: 1.5, transition: 'box-shadow 0.2s, border-color 0.2s', '&:hover': { boxShadow: 'md', borderColor: 'neutral.outlinedHoverBorder' } }}
    >
      <FaixaJustica justica={e.justica} />
      <Stack direction="row" spacing={2} alignItems="center">
        <Foto src={urlFoto(e.foto)} nome={e.nomeUrna} largura={76} />
        <Box sx={{ minWidth: 0 }}>
          <Typography level="title-lg">{e.nomeUrna}</Typography>
          {e.nome && e.nome.toUpperCase() !== e.nomeUrna.toUpperCase() && (
            <Typography level="body-xs" title={e.nome}>{e.nome}</Typography>
          )}
          <Stack direction="row" spacing={0.75} useFlexGap flexWrap="wrap" sx={{ mt: 0.75 }}>
            <Chip variant="solid" color="primary" size="sm">{e.partido}</Chip>
            <Chip variant="soft" size="sm">{local}</Chip>
            {e.situacao && e.situacao !== 'Eleito' && <Chip variant="outlined" size="sm">{e.situacao}</Chip>}
          </Stack>
        </Box>
      </Stack>

      <Box>
        <Votacao e={e} />
        <Typography level="body-xs" sx={{ mt: 0.25 }}>Mandato {e.mandato}</Typography>
        {e.observacao && <Typography level="body-xs" color="warning">⚠️ {e.observacao}</Typography>}
      </Box>

      <ListaVices vices={e.vices} />

      {e.desempenho && (
        <Metricas
          rotulo={e.desempenho.rotulo}
          percentual={e.desempenho.percentual}
          detalhe={e.desempenho.detalhe}
          projetos={e.desempenho.projetos}
        />
      )}
      {e.desempenho && <ExtrasDesempenho d={e.desempenho} custo={e.custoMandato} />}

      <Divider />
      <Em2026 e={e} ano={ano} />
      {e.nota && <Alert size="sm" variant="soft" color="warning" startDecorator="🔁">{e.nota}</Alert>}
      <Typography level="body-xs" color="primary" fontWeight="lg" sx={{ mt: 'auto', pt: 0.5 }}>Ver perfil completo →</Typography>
    </Card>
  );
}

// ---------- Página ----------

// Cargos com poucos eleitos abrem com o país inteiro; os restantes, com um estado.
const POUCOS_ELEITOS = ['governador', 'senador'];

function Eleitos({ cargo, onAbrirPerfil }) {
  const municipal = MUNICIPAIS.includes(cargo);
  const [uf, setUf] = useState(POUCOS_ELEITOS.includes(cargo) ? 'TODOS' : 'SC');
  const [municipio, setMunicipio] = useState('TODOS');
  const [busca, setBusca] = useState('');
  const [buscaAplicada, setBuscaAplicada] = useState('');
  const [ordem, setOrdem] = useState(cargo === 'prefeito' ? 'municipio' : 'votos');
  const [em2026, setEm2026] = useState('todos');
  const [soJustica, setSoJustica] = useState(false);
  const [partido, setPartido] = useState('');
  const [genero, setGenero] = useState('');
  const [faixa, setFaixa] = useState('');
  const [nivel, setNivel] = useState('');
  const [pagina, setPagina] = useState(1);

  const [resultado, setResultado] = useState(null);
  const [loading, setLoading] = useState(true);
  const [erro, setErro] = useState(null);
  const [tentativa, setTentativa] = useState(0);

  const filtro = (setter) => (valor) => {
    setter(valor);
    setPagina(1);
  };

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
      cargo, uf, municipio, q: buscaAplicada, ordem, em2026, pagina, justica: soJustica ? '1' : '0', partido, genero, faixa, nivel,
    });
    buscarAPI(`/api/eleitos?${params}`, controller.signal)
      .then((data) => {
        setResultado(data);
        setLoading(false);
      })
      .catch((err) => {
        if (err.name === 'AbortError') return;
        console.error('Erro ao procurar eleitos:', err);
        setErro(mensagemDeErro(err));
        setResultado(null);
        setLoading(false);
      });
    return () => controller.abort();
  }, [cargo, uf, municipio, buscaAplicada, ordem, em2026, soJustica, partido, genero, faixa, nivel, pagina, tentativa]);

  const mudarUf = (valor) => {
    setUf(valor);
    setMunicipio('TODOS');
    setPagina(1);
  };

  const mudarPagina = (p) => {
    setPagina(p);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const ano2026 = resultado?.anoEleicao2026 ?? 2026;

  return (
    <Stack spacing={2.5}>
      <Card variant="outlined">
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} useFlexGap flexWrap="wrap">
          {cargo !== 'presidente' && (
            <FormControl sx={{ flex: 1, minWidth: 190 }}>
              <FormLabel>Estado</FormLabel>
              <Select value={uf} onChange={(_, v) => mudarUf(v)}>
                <Option value="TODOS">🌎 Todos os Estados</Option>
                {ESTADOS.map(([sigla, nome]) => <Option key={sigla} value={sigla}>{nome} ({sigla})</Option>)}
              </Select>
            </FormControl>
          )}

          {municipal && <SeletorMunicipio uf={uf} valor={municipio} onMudar={filtro(setMunicipio)} />}

          {cargo !== 'presidente' && (
            <FormControl sx={{ flex: 1.5, minWidth: 200 }}>
              <FormLabel>Pesquisar</FormLabel>
              <Input placeholder="Nome ou partido…" startDecorator="🔍" value={busca} onChange={(e) => setBusca(e.target.value)} />
            </FormControl>
          )}

          {cargo !== 'presidente' && (
            <FormControl sx={{ flex: 1.2, minWidth: 220 }}>
              <FormLabel>Ordenar por</FormLabel>
              <Select value={ordem} onChange={(_, v) => filtro(setOrdem)(v)}>
                {COM_DESEMPENHO[cargo] && [
                  <Option key="indice" value="indice">⭐ Melhor índice de atuação</Option>,
                  <Option key="indice-asc" value="indice-asc">Pior índice de atuação</Option>,
                  <Option key="desempenho" value="desempenho">Maior {COM_DESEMPENHO[cargo]}</Option>,
                  <Option key="desempenho-asc" value="desempenho-asc">Menor {COM_DESEMPENHO[cargo]}</Option>,
                  <Option key="projetos" value="projetos">Mais projetos apresentados</Option>,
                  <Option key="projetos-asc" value="projetos-asc">Menos projetos apresentados</Option>,
                  <Option key="votacoes" value="votacoes">Mais votações em que votou</Option>,
                ]}
                {COM_DESEMPENHO[cargo] && [
                  <Option key="custo" value="custo">💰 Maior custo do mandato</Option>,
                  <Option key="custo-asc" value="custo-asc">Menor custo do mandato</Option>,
                ]}
                {cargo === 'deputado-federal' && [
                  <Option key="gastos" value="gastos">Maior gasto de cota</Option>,
                  <Option key="gastos-asc" value="gastos-asc">Menor gasto de cota</Option>,
                  <Option key="governo" value="governo">Mais alinhado ao Governo</Option>,
                  <Option key="governo-asc" value="governo-asc">Menos alinhado ao Governo</Option>,
                ]}
                <Option value="votos">Mais votados na eleição</Option>
                <Option value="nome">Nome (A-Z)</Option>
                <Option value="partido">Partido</Option>
                {municipal && <Option value="municipio">Município</Option>}
                <Option value="idade-asc">Mais novos</Option>
                <Option value="idade">Mais velhos</Option>
              </Select>
            </FormControl>
          )}
        </Stack>

        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} useFlexGap flexWrap="wrap" sx={{ mt: 1 }}>
          {cargo !== 'presidente' && (
            <FiltrosPessoa
              partidos={resultado?.partidos}
              partido={partido} genero={genero} faixa={faixa}
              onPartido={filtro(setPartido)} onGenero={filtro(setGenero)} onFaixa={filtro(setFaixa)}
            />
          )}
          <FormControl sx={{ flex: 1, minWidth: 170 }}>
            <FormLabel>Em {ano2026}</FormLabel>
            <Select value={em2026} onChange={(_, v) => filtro(setEm2026)(v)}>
              <Option value="todos">Todos</Option>
              <Option value="candidato">Concorre a algum cargo</Option>
              {!municipal && <Option value="reeleicao">Busca a reeleição</Option>}
              {!municipal && <Option value="outro">Concorre a outro cargo</Option>}
              <Option value="nao">Não concorre</Option>
            </Select>
          </FormControl>
          {COM_DESEMPENHO[cargo] && (
            <FormControl sx={{ flex: 1, minWidth: 170 }}>
              <FormLabel>{COM_DESEMPENHO[cargo][0].toUpperCase() + COM_DESEMPENHO[cargo].slice(1)}</FormLabel>
              <Select value={nivel} onChange={(_, v) => filtro(setNivel)(v)}>
                <Option value="">Qualquer</Option>
                <Option value="95">95% ou mais</Option>
                <Option value="90">90% ou mais</Option>
                <Option value="abaixo80">Abaixo de 80%</Option>
              </Select>
            </FormControl>
          )}
        </Stack>
        <FiltroJustica marcado={soJustica} onMudar={filtro(setSoJustica)} />
      </Card>

      {resultado && !erro && (
        <>
          <QuantoElege uf={cargo === 'presidente' ? 'TODOS' : uf} municipio={municipal ? municipio : 'TODOS'} cargo={cargo} modo="eleitos" />
          <Resumo resumo={resultado.resumo} cargo={cargo} ano2026={ano2026} />
          {COM_DESEMPENHO[cargo] && (
            <Alert size="sm" variant="soft" color="neutral" startDecorator="ℹ️">
              <span>
                <b>Índice de atuação (0–100):</b> média dos percentis de {COM_DESEMPENHO[cargo]}, projetos apresentados e
                votações em que votou, comparando com os colegas do mesmo cargo (100 = o melhor em tudo, 50 = na média).
                Licenças (por exemplo, para ser ministro) não são descontadas e baixam estes números.
              </span>
            </Alert>
          )}
          <Avisos avisos={resultado.avisos} />
        </>
      )}

      {loading && !resultado ? (
        <GradeCarregando />
      ) : erro ? (
        <ErroCarregamento mensagem={erro} onTentarDeNovo={() => setTentativa((t) => t + 1)} />
      ) : resultado.eleitos.length === 0 ? (
        <Mensagem>Nenhum eleito encontrado com estes filtros.</Mensagem>
      ) : (
        <>
          <Grade atualizando={loading}>
            {resultado.eleitos.map((e) => <CartaoEleito key={e.id} e={e} ano={ano2026} onAbrir={onAbrirPerfil} />)}
          </Grade>
          <Paginacao pagina={resultado.pagina} totalPaginas={resultado.totalPaginas} desativado={loading} onMudar={mudarPagina} />
        </>
      )}
    </Stack>
  );
}

export default Eleitos;
