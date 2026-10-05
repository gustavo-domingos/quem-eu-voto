import { useEffect, useState } from 'react';
import {
  Accordion, AccordionDetails, AccordionGroup, AccordionSummary, Alert, Box, Button, Chip, CircularProgress, Divider, Link, List, ListItem, Modal, ModalClose, ModalDialog, Sheet, Stack, Tab, TabList, TabPanel, Table, Tabs, Typography,
} from '@mui/joy';
import { API_URL, buscarAPI, urlFoto } from '../../services/api';
import { formatarData } from '../../utils/formatacao';
import { Foto } from '../../components/Foto';
import { Metricas } from '../../components/Metricas';
import { ResultadoNaApuracao } from '../apuracao/Apuracao';

const numero = (n) => n.toLocaleString('pt-BR');
const pct = (n) => `${n.toLocaleString('pt-BR', { minimumFractionDigits: 1, maximumFractionDigits: 1 })}%`;
const reais = (n) => n.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL', maximumFractionDigits: 0 });

// ---------- Peças ----------

function Secao({ titulo, children, extra }) {
  return (
    <Box sx={{ mb: 3 }}>
      <Stack direction="row" justifyContent="space-between" alignItems="baseline" sx={{ mb: 1 }}>
        <Typography level="title-md">{titulo}</Typography>
        {extra}
      </Stack>
      {children}
    </Box>
  );
}

function Campo({ rotulo, children }) {
  if (!children) return null;
  return (
    <Box>
      <Typography level="body-xs" textTransform="uppercase" letterSpacing="0.05em">{rotulo}</Typography>
      <Typography level="body-md">{children}</Typography>
    </Box>
  );
}

function Indicador({ rotulo, valor, detalhe }) {
  return (
    <Sheet variant="soft" sx={{ borderRadius: 'md', p: 1.5 }}>
      <Typography level="body-xs" fontWeight="lg">{rotulo}</Typography>
      <Typography level="h4">{valor}</Typography>
      {detalhe && <Typography level="body-xs">{detalhe}</Typography>}
    </Sheet>
  );
}

// Barras horizontais de uma série; valores em cor de texto, na ponta da barra.
function Barras({ itens, formatar = numero }) {
  if (!itens?.length) return null;
  const maior = Math.max(...itens.map((i) => i.valor), 1);
  return (
    <Stack spacing={0.75}>
      {itens.map((i) => (
        <Box key={i.rotulo} title={`${i.rotulo}: ${formatar(i.valor)}`}
          sx={{ display: 'grid', gridTemplateColumns: 'minmax(110px, 40%) 1fr', gap: 1, alignItems: 'center' }}>
          <Typography level="body-xs" noWrap>{i.rotulo}</Typography>
          <Stack direction="row" spacing={0.75} alignItems="center" sx={{ minWidth: 0 }}>
            <Box sx={{ height: 10, width: `${(i.valor / maior) * 100}%`, minWidth: i.valor > 0 ? 2 : 0, bgcolor: 'primary.500', borderRadius: '0 4px 4px 0' }} />
            <Typography level="body-xs" fontWeight="lg" sx={{ flexShrink: 0 }}>{formatar(i.valor)}</Typography>
          </Stack>
        </Box>
      ))}
    </Stack>
  );
}

const ESTILO_INDICADOR = {
  atencao: { cor: 'warning', icone: '⚠️', titulo: 'Pontos de atenção' },
  destaque: { cor: 'success', icone: '✅', titulo: 'Destaques' },
  info: { cor: 'neutral', icone: 'ℹ️', titulo: 'Para saber' },
};

function Indicadores({ indicadores }) {
  if (!indicadores.length) return null;
  return (
    <Stack spacing={1.5} sx={{ mb: 2 }}>
      {['atencao', 'destaque', 'info'].map((tipo) => {
        const itens = indicadores.filter((i) => i.tipo === tipo);
        if (!itens.length) return null;
        const e = ESTILO_INDICADOR[tipo];
        return (
          <Alert key={tipo} variant="soft" color={e.cor} sx={{ alignItems: 'flex-start' }} startDecorator={e.icone}>
            <Box>
              <Typography level="title-sm" textColor="inherit">{e.titulo}</Typography>
              <List size="sm" sx={{ '--ListItem-paddingX': 0, '--ListItem-minHeight': 0 }}>
                {itens.map((i) => (
                  <ListItem key={i.texto} sx={{ display: 'block' }}>
                    <Typography level="body-sm" textColor="inherit">{i.texto}</Typography>
                    <Typography level="body-xs" textColor="inherit" sx={{ opacity: 0.75 }}>Fonte: {i.fonte}</Typography>
                  </ListItem>
                ))}
              </List>
            </Box>
          </Alert>
        );
      })}
    </Stack>
  );
}

// ---------- Justiça e órgãos de controle ----------

function Justica({ p }) {
  const regs = p.justica || [];
  const nota = (
    <Typography level="body-xs" sx={{ mt: 1 }}>
      Consultamos o CEIS, o CNEP e o CEAF da Controladoria-Geral da União (inclui as condenações por improbidade
      comunicadas pelo CNJ){p.justicaDataCGU && `, atualizados em ${formatarData(p.justicaDataCGU)}`}, e as decisões da
      Justiça Eleitoral sobre candidaturas de 2022, 2024 e 2026. Improbidade administrativa é uma condenação cível, não
      criminal. Não incluímos inquéritos nem processos sem decisão. Antecedentes criminais não estão disponíveis em dados abertos.
    </Typography>
  );
  if (regs.length === 0) {
    return (
      <Alert variant="soft" color={p.justicaVerificada ? 'success' : 'neutral'} sx={{ mb: 2, display: 'block' }}>
        <Typography level="title-sm" textColor="inherit">
          ⚖️ {p.justicaVerificada ? 'Nenhum registro encontrado nos cadastros oficiais consultados' : 'Verificação parcial'}
        </Typography>
        {!p.justicaVerificada && (
          <Typography level="body-sm" textColor="inherit">
            Não foi possível cruzar com os cadastros da CGU porque o TSE não divulga o CPF desta pessoa.
            Só as decisões da Justiça Eleitoral foram consultadas.
          </Typography>
        )}
        {nota}
      </Alert>
    );
  }
  return (
    <Sheet variant="outlined" color="danger" sx={{ borderRadius: 'lg', mb: 2, overflow: 'hidden', borderWidth: 2 }}>
      <Box sx={{ bgcolor: 'danger.solidBg', color: '#fff', px: 2, py: 1.25 }}>
        <Typography level="title-lg" textColor="inherit">⚖️ Registros na Justiça e em órgãos de controle ({regs.length})</Typography>
      </Box>
      <Stack divider={<Divider />} sx={{ px: 2 }}>
        {regs.map((r, i) => (
          <Box key={i} sx={{ py: 1.5 }}>
            <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" alignItems="center">
              <Typography level="title-md" color="danger">{r.titulo}</Typography>
              {r.corrupcao && <Chip size="sm" color="danger" variant="solid">envolve improbidade ou corrupção</Chip>}
              {r.vigente === true && <Chip size="sm" color="danger" variant="soft">sanção em vigor</Chip>}
              {r.vigente === false && <Chip size="sm" variant="soft">sanção já encerrada</Chip>}
            </Stack>
            {r.detalhe && <Typography level="body-sm" sx={{ mt: 0.5 }}>{r.detalhe}</Typography>}
            <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: 1, mt: 1 }}>
              <Campo rotulo="Órgão">{r.orgao}</Campo>
              <Campo rotulo="Processo">{r.processo}</Campo>
              <Campo rotulo="Início da sanção">{r.inicio && formatarData(r.inicio)}</Campo>
              <Campo rotulo="Fim da sanção">{r.inicio ? (r.fim ? formatarData(r.fim) : 'sem prazo') : null}</Campo>
              <Campo rotulo="Trânsito em julgado">{r.transito && formatarData(r.transito)}</Campo>
            </Box>
            <Typography level="body-xs" sx={{ mt: 0.5 }}>
              Fonte: {r.fonte} · <Link href={r.link} target="_blank" rel="noopener noreferrer">consultar na fonte ↗</Link>
            </Typography>
          </Box>
        ))}
      </Stack>
      <Box sx={{ px: 2, pb: 1.5 }}>{nota}</Box>
    </Sheet>
  );
}

// ---------- Abas ----------

function Curriculo({ p }) {
  const ps = p.pessoal;
  return (
    <>
      <Secao titulo="Dados pessoais">
        <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: 2 }}>
          <Campo rotulo="Nome completo">{ps.nome}</Campo>
          <Campo rotulo="Idade">{ps.idade != null && `${ps.idade} anos${ps.nascimento ? ` (${formatarData(ps.nascimento)})` : ''}`}</Campo>
          <Campo rotulo="Naturalidade">{ps.naturalidade}</Campo>
          <Campo rotulo="Escolaridade">{ps.escolaridade}</Campo>
          <Campo rotulo="Ocupação declarada">{ps.ocupacao}</Campo>
          <Campo rotulo="Gênero">{ps.genero}</Campo>
          <Campo rotulo="Estado civil">{ps.estadoCivil}</Campo>
          <Campo rotulo="Cor/raça (autodeclarada)">{ps.corRaca}</Campo>
          <Campo rotulo="E-mail">{ps.email}</Campo>
        </Box>
      </Secao>
      {ps.redes.length > 0 && (
        <Secao titulo="Redes e site">
          <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
            {ps.redes.map((r) => (
              <Button key={r.url} component="a" href={r.url} target="_blank" rel="noopener noreferrer" size="sm" variant="outlined" color="neutral">
                {r.rotulo} ↗
              </Button>
            ))}
          </Stack>
        </Secao>
      )}
      <Secao titulo={`Candidatura em ${p.anoEleicao2026}`}>
        {p.candidaturas2026.length === 0 ? (
          <Typography level="body-sm">Não é candidato(a) em {p.anoEleicao2026}.</Typography>
        ) : p.candidaturas2026.map((c) => (
          <Sheet key={c.id} variant="soft" color={c.apto ? 'success' : 'danger'} sx={{ borderRadius: 'md', p: 1.5, mb: 1, display: 'flex', gap: 2, alignItems: 'center' }}>
            <Typography level="h3">{c.numero}</Typography>
            <Box>
              <Typography level="body-md"><b>{c.cargo}</b> ({c.uf === 'BR' ? 'Brasil' : c.uf}) · {c.partido}</Typography>
              <Typography level="body-xs">Na urna: {c.nomeUrna} · Situação: {c.situacao || '—'}{c.coligacao ? ` · ${c.coligacao}` : ''}</Typography>
              {p.apuracao2026?.[c.id] && <Box sx={{ mt: 0.5 }}><ResultadoNaApuracao res={p.apuracao2026[c.id]} /></Box>}
            </Box>
          </Sheet>
        ))}
      </Secao>
    </>
  );
}

function Votacoes({ v }) {
  return (
    <Secao titulo={`Votações nominais (${v.casa})`}>
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 1.5, mb: 2 }}>
        <Indicador rotulo="Votações no mandato" valor={numero(v.total)} />
        {v.alinhamentoGoverno != null && (
          <Indicador rotulo="Votou como o Governo orientou" valor={pct(v.alinhamentoGoverno)} detalhe={`em ${numero(v.comparaveis)} votações com orientação Sim/Não`} />
        )}
      </Box>
      {v.contagem?.length > 0 && <Box sx={{ mb: 2 }}><Barras itens={v.contagem.map((c) => ({ rotulo: c.rotulo, valor: c.total }))} /></Box>}
      {v.recentes?.length > 0 && (
        <>
          <Typography level="title-sm" sx={{ mb: 1 }}>Votações mais recentes</Typography>
          <Sheet variant="outlined" sx={{ borderRadius: 'md', overflow: 'auto' }}>
            <Table size="sm" sx={{ minWidth: 560, '& td': { verticalAlign: 'top' } }}>
              <thead>
                <tr><th style={{ width: 90 }}>Data</th><th>Proposição</th><th style={{ width: 110 }}>Voto</th>{v.recentes.some((r) => r.orientacaoGoverno) && <th style={{ width: 110 }}>Governo</th>}</tr>
              </thead>
              <tbody>
                {v.recentes.map((r, i) => (
                  <tr key={i}>
                    <td>{formatarData(r.data)}</td>
                    <td>
                      <Typography level="body-sm" fontWeight="lg">{r.proposicao || '—'}</Typography>
                      <Typography level="body-xs">{r.ementa || r.descricao}</Typography>
                    </td>
                    <td><Chip size="sm" variant="soft">{r.voto || '—'}</Chip></td>
                    {v.recentes.some((x) => x.orientacaoGoverno) && <td><Typography level="body-xs">{r.orientacaoGoverno || '—'}</Typography></td>}
                  </tr>
                ))}
              </tbody>
            </Table>
          </Sheet>
        </>
      )}
    </Secao>
  );
}

function Gastos({ g }) {
  const dif = g.mediaDeputados ? ((g.total - g.mediaDeputados) / g.mediaDeputados) * 100 : null;
  return (
    <Secao titulo="Gastos da cota parlamentar (desde 2023)">
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 1.5, mb: 2 }}>
        <Indicador rotulo="Total gasto" valor={reais(g.total)} detalhe={dif != null ? `${dif >= 0 ? '+' : ''}${pct(dif)} vs. média` : null} />
        <Indicador rotulo="Média dos deputados" valor={reais(g.mediaDeputados)} />
        <Indicador rotulo="Posição" valor={`${g.posicao}º de ${g.de}`} detalhe="1º = quem mais gastou" />
      </Box>
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))', gap: 3 }}>
        <Box><Typography level="title-sm" sx={{ mb: 1 }}>Por ano</Typography><Barras itens={g.porAno.map((a) => ({ rotulo: a.rotulo, valor: a.valor }))} formatar={reais} /></Box>
        <Box><Typography level="title-sm" sx={{ mb: 1 }}>Maiores categorias</Typography><Barras itens={g.categorias.map((a) => ({ rotulo: a.rotulo, valor: a.valor }))} formatar={reais} /></Box>
        <Box><Typography level="title-sm" sx={{ mb: 1 }}>Maiores fornecedores</Typography><Barras itens={g.fornecedores.map((a) => ({ rotulo: a.rotulo, valor: a.valor }))} formatar={reais} /></Box>
      </Box>
    </Secao>
  );
}

function CustoMandato({ c }) {
  const s = c.subsidio;
  return (
    <Secao titulo="Quanto o mandato custou até agora">
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(170px, 1fr))', gap: 1.5, mb: 2 }}>
        <Indicador rotulo="Total identificado" valor={reais(c.total)} detalhe="subsídio estimado + cota parlamentar" />
        {s && (
          <Indicador
            rotulo="Subsídio (salário)"
            valor={reais(s.total)}
            detalhe={`${numero(s.dias)} dias em exercício · hoje ${reais(s.mensal)}/mês`}
          />
        )}
        {c.cota && <Indicador rotulo="Cota parlamentar" valor={reais(c.cota.total)} detalhe="escritório, viagens, divulgação…" />}
        {c.secretarios != null && (
          <Indicador rotulo="Funcionários do gabinete" valor={numero(c.secretarios)} detalhe="secretários parlamentares contratados hoje" />
        )}
      </Box>
      {c.cota && (
        <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(min(380px, 100%), 1fr))', gap: 3, mb: 2 }}>
          <Box>
            <Typography level="title-sm" sx={{ mb: 1 }}>Cota por tipo de gasto</Typography>
            <Barras itens={c.cota.porGrupo.map((v) => ({ rotulo: v.rotulo, valor: v.valor }))} formatar={reais} />
          </Box>
          <Box>
            <Typography level="title-sm" sx={{ mb: 1 }}>Cota por ano</Typography>
            <Barras itens={c.cota.porAno.map((v) => ({ rotulo: v.rotulo, valor: v.valor }))} formatar={reais} />
          </Box>
        </Box>
      )}
      <Alert size="sm" variant="soft" color="neutral" sx={{ display: 'block' }}>
        <Typography level="body-xs" textColor="inherit">
          <b>Como calculamos:</b> o subsídio é estimado pelo valor fixado em lei ({s?.fonte || 'Decreto Legislativo nº 172/2022'}),
          proporcional aos dias em exercício desde 01/02/2023; a cota vem dos dados abertos ({c.cota?.nome || 'Câmara e Senado'}).
        </Typography>
        <Typography level="body-xs" textColor="inherit" sx={{ mt: 0.5 }}><b>Não incluído (sem dados abertos):</b></Typography>
        <List size="sm" sx={{ '--ListItem-minHeight': 0, '--ListItem-paddingY': 0 }}>
          {c.naoIncluido.map((t) => <ListItem key={t}><Typography level="body-xs" textColor="inherit">• {t}</Typography></ListItem>)}
        </List>
      </Alert>
    </Secao>
  );
}

function Mandato({ m }) {
  const v = m.votacao;
  return (
    <>
      <Secao titulo="Mandato atual" extra={m.pagina && <Link href={m.pagina.url} target="_blank" rel="noopener noreferrer" level="body-sm">{m.pagina.rotulo} ↗</Link>}>
        <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: 2 }}>
          <Campo rotulo="Cargo">{m.cargo}</Campo>
          <Campo rotulo="Local">{m.local}</Campo>
          <Campo rotulo="Período">{m.periodo}</Campo>
          <Campo rotulo="Situação">{m.situacao}</Campo>
          <Campo rotulo={`Eleição de ${m.anoEleicao || '—'}`}>
            {v ? `${numero(v.v)} votos · ${pct(v.pc)} dos válidos · ${v.p}º de ${numero(v.d)}${v.t === 2 ? ' (2º turno)' : ''}` : m.anoEleicao ? 'Votação não disponível' : null}
          </Campo>
          {m.vices?.length > 0 && <Campo rotulo={m.vices[0].cargo.includes('Suplente') ? 'Suplentes' : 'Vice'}>{m.vices.map((x) => x.nomeUrna).join(', ')}</Campo>}
        </Box>
        {m.nota && <Alert size="sm" variant="soft" color="warning" sx={{ mt: 1.5 }}>🔁 {m.nota}</Alert>}
      </Secao>
      {m.desempenho && (
        <Secao titulo="Atuação">
          <Metricas rotulo={m.desempenho.rotulo} percentual={m.desempenho.percentual} detalhe={m.desempenho.detalhe} projetos={m.desempenho.projetos} />
        </Secao>
      )}
      {!m.desempenho && !m.votacoes && (
        <Alert variant="soft" size="sm" sx={{ mb: 3 }}>
          Para este cargo ainda não há uma fonte nacional de dados de atuação (presença, votações) em formato aberto.
        </Alert>
      )}
      {m.custo && <CustoMandato c={m.custo} />}
      {m.votacoes && <Votacoes v={m.votacoes} />}
      {m.gastos && <Gastos g={m.gastos} />}
      {m.projetos?.length > 0 && (
        <Secao titulo="Projetos mais recentes (PL, PLP, PEC)">
          <Stack spacing={1}>
            {m.projetos.map((pr) => (
              <Sheet key={pr.titulo} variant="outlined" sx={{ borderRadius: 'md', p: 1.5 }}>
                <Link href={pr.url} target="_blank" rel="noopener noreferrer" level="title-sm">{pr.titulo} ↗</Link>
                {pr.data && <Typography level="body-xs">{formatarData(pr.data)}</Typography>}
                <Typography level="body-sm">{pr.ementa}</Typography>
              </Sheet>
            ))}
          </Stack>
        </Secao>
      )}
    </>
  );
}

function Patrimonio({ pat, ano }) {
  return (
    <>
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(170px, 1fr))', gap: 1.5, mb: 2 }}>
        <Indicador rotulo={`Declarado em ${ano}`} valor={reais(pat.total2026)} detalhe={`${pat.bens.length} bem(ns)`} />
        {pat.total2022 != null && <Indicador rotulo="Declarado em 2022" valor={reais(pat.total2022)} />}
        {pat.variacao != null && <Indicador rotulo="Variação" valor={`${pat.variacao >= 0 ? '+' : ''}${pct(pat.variacao)}`} detalhe="valores nominais, sem correção pela inflação" />}
      </Box>
      {pat.bens.length > 0 && (
        <Sheet variant="outlined" sx={{ borderRadius: 'md', overflow: 'auto' }}>
          <Table size="sm" sx={{ minWidth: 480 }}>
            <thead><tr><th style={{ width: '30%' }}>Tipo</th><th>Descrição</th><th style={{ width: 130, textAlign: 'right' }}>Valor</th></tr></thead>
            <tbody>
              {pat.bens.map((b, i) => (
                <tr key={i}>
                  <td>{b.tipo}</td>
                  <td><Typography level="body-xs">{b.descricao}</Typography></td>
                  <td style={{ textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}>{reais(b.valor)}</td>
                </tr>
              ))}
            </tbody>
          </Table>
        </Sheet>
      )}
      <Typography level="body-xs" sx={{ mt: 1 }}>Fonte: declaração de bens entregue ao TSE no registro da candidatura.</Typography>
    </>
  );
}

function Historico({ h }) {
  if (!h.length) return <Typography level="body-sm">Sem candidaturas encontradas nas eleições gerais de 2018 e 2022 nem entre os eleitos de 2024.</Typography>;
  return (
    <>
      <Sheet variant="outlined" sx={{ borderRadius: 'md', overflow: 'auto' }}>
        <Table size="sm" sx={{ minWidth: 560 }}>
          <thead><tr><th style={{ width: 60 }}>Ano</th><th>Cargo</th><th>Local</th><th>Partido</th><th>Resultado</th><th style={{ textAlign: 'right' }}>Votos</th></tr></thead>
          <tbody>
            {h.map((c, i) => (
              <tr key={i}>
                <td>{c.ano}</td>
                <td>{c.cargo} <Typography level="body-xs">nº {c.numero}</Typography></td>
                <td>{c.local}</td>
                <td>{c.partido}</td>
                <td><Chip size="sm" variant="soft" color={c.eleito ? 'success' : 'neutral'}>{c.resultado}</Chip></td>
                <td style={{ textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}>{c.votos != null ? `${numero(c.votos)} (${pct(c.percentual)})` : '—'}</td>
              </tr>
            ))}
          </tbody>
        </Table>
      </Sheet>
      <Typography level="body-xs" sx={{ mt: 1 }}>Fonte: TSE. Inclui as eleições gerais de 2018 e 2022 e, para quem foi eleito em 2024, o mandato municipal. Votos disponíveis a partir de 2022.</Typography>
    </>
  );
}

function Noticias({ nome }) {
  const [dados, setDados] = useState(null);
  useEffect(() => {
    const controller = new AbortController();
    buscarAPI(`/api/noticias?nome=${encodeURIComponent(nome)}`, controller.signal)
      .then(setDados)
      .catch((err) => { if (err.name !== 'AbortError') setDados({ noticias: [], pesquisa: [], aviso: 'Não foi possível carregar as notícias.' }); });
    return () => controller.abort();
  }, [nome]);

  if (!dados) {
    return <Stack direction="row" spacing={1} alignItems="center"><CircularProgress size="sm" /><Typography level="body-sm">Buscando notícias sobre “{nome}”…</Typography></Stack>;
  }
  return (
    <>
      {dados.aviso && <Alert size="sm" variant="soft" color="warning" sx={{ mb: 2 }}>{dados.aviso}</Alert>}
      {dados.noticias.length > 0 ? (
        <Stack spacing={1} sx={{ mb: 2 }}>
          {dados.noticias.map((n) => (
            <Sheet key={n.url} variant="outlined" sx={{ borderRadius: 'md', p: 1.5 }}>
              <Link href={n.url} target="_blank" rel="noopener noreferrer" level="title-sm">{n.titulo}</Link>
              <Typography level="body-xs">{n.fonte}{n.data && ` · ${formatarData(n.data)}`}</Typography>
            </Sheet>
          ))}
        </Stack>
      ) : !dados.aviso && <Typography level="body-sm" sx={{ mb: 2 }}>Nenhuma notícia encontrada nos últimos 12 meses.</Typography>}
      <Typography level="title-sm" sx={{ mb: 1 }}>Pesquisar por conta própria</Typography>
      <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
        {dados.pesquisa.map((l) => (
          <Button key={l.url} component="a" href={l.url} target="_blank" rel="noopener noreferrer" size="sm" variant="outlined" color="neutral">{l.rotulo} ↗</Button>
        ))}
      </Stack>
      <Typography level="body-xs" sx={{ mt: 2 }}>
        Notícias de veículos de terceiros, encontradas pela base aberta GDELT. Não verificamos o conteúdo: confira sempre a fonte.
      </Typography>
    </>
  );
}

// ---------- Contas de campanha ----------

const reaisCentavos = (n) => n.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL', minimumFractionDigits: 2, maximumFractionDigits: 2 });

function ContasCampanha({ c, ultima }) {
  const publicoPct = c.receitas > 0 ? (c.dinheiroPublico / c.receitas) * 100 : null;
  const limitePct = c.limite ? (c.despesas / c.limite) * 100 : null;
  const barras = (lista) => lista.map((v) => ({ rotulo: v.rotulo, valor: v.valor }));
  return (
    <Secao titulo={`Campanha a ${c.cargo} (${ultima})`}>
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 1.5, mb: 2 }}>
        <Indicador rotulo="Gastou (despesas contratadas)" valor={reais(c.despesas)} />
        <Indicador rotulo="Arrecadou" valor={reais(c.receitas)} />
        <Indicador
          rotulo="Dinheiro público"
          valor={reais(c.dinheiroPublico)}
          detalhe={publicoPct != null ? `${pct(publicoPct)} do que arrecadou (Fundo Eleitoral e Partidário)` : 'Fundo Eleitoral e Partidário'}
        />
        {limitePct != null && <Indicador rotulo="Do limite legal de gastos" valor={pct(limitePct)} detalhe={`limite: ${reais(c.limite)}`} />}
        {c.custoPorVoto != null && <Indicador rotulo="Custo por voto" valor={reaisCentavos(c.custoPorVoto)} detalhe="gasto ÷ votos na apuração" />}
      </Box>
      {c.despesas === 0 && c.receitas === 0 ? (
        <Typography level="body-sm">Esta candidatura ainda não declarou receitas nem despesas ao TSE.</Typography>
      ) : (
        <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(min(380px, 100%), 1fr))', gap: 3 }}>
          <Box><Typography level="title-sm" sx={{ mb: 1 }}>Em que gastou</Typography><Barras itens={barras(c.porTipoDespesa)} formatar={reais} /></Box>
          <Box><Typography level="title-sm" sx={{ mb: 1 }}>De onde veio o dinheiro</Typography><Barras itens={barras(c.porOrigemReceita)} formatar={reais} /></Box>
          <Box><Typography level="title-sm" sx={{ mb: 1 }}>Maiores fornecedores</Typography><Barras itens={barras(c.fornecedores)} formatar={reais} /></Box>
          <Box><Typography level="title-sm" sx={{ mb: 1 }}>Maiores doadores</Typography><Barras itens={barras(c.doadores)} formatar={reais} /></Box>
        </Box>
      )}
      <Typography level="body-xs" sx={{ mt: 1.5 }}>
        {c.ultimaPrestacao && `Valores declarados ao TSE até ${formatarData(c.ultimaPrestacao)}. `}
        Durante a campanha as contas são parciais: a prestação final é entregue até 30 dias depois da eleição.
        Doações do partido costumam ser repasses do Fundo Eleitoral.
      </Typography>
    </Secao>
  );
}

function Campanha({ p }) {
  const entradas = Object.entries(p.campanha2026 || {});
  if (!entradas.length) {
    return <Alert variant="soft" color="neutral">Não há contas de campanha de {p.anoEleicao2026} declaradas para esta pessoa.</Alert>;
  }
  return (
    <>
      {entradas.map(([id, c]) => <ContasCampanha key={id} c={c} ultima={p.anoEleicao2026} />)}
      <Typography level="body-xs">Fonte: prestação de contas eleitorais dos candidatos (TSE, dados abertos).</Typography>
    </>
  );
}

// ---------- Propostas de governo ----------

const tamanhoMB = (bytes) => `${(bytes / (1024 * 1024)).toLocaleString('pt-BR', { maximumFractionDigits: 1 })} MB`;

function ResumoPorTema({ temas }) {
  if (!temas.length) {
    return <Typography level="body-sm">Não encontramos frases de compromisso no formato “Criar…”, “Ampliar…”, “Implantar…”.</Typography>;
  }
  return (
    <AccordionGroup variant="outlined" sx={{ borderRadius: 'md', overflow: 'hidden' }}>
      {temas.map((t, i) => (
        <Accordion key={t.tema} defaultExpanded={i < 2}>
          <AccordionSummary>
            <Typography level="title-sm" sx={{ flex: 1 }}>{t.tema}</Typography>
            <Chip size="sm" variant="soft">{t.total} {t.total === 1 ? 'compromisso' : 'compromissos'}</Chip>
          </AccordionSummary>
          <AccordionDetails>
            <Stack spacing={1} sx={{ pt: 0.5 }}>
              {t.compromissos.map((c) => (
                <Sheet key={c} variant="soft" sx={{ borderRadius: 'sm', px: 1.5, py: 1, borderLeft: '3px solid', borderLeftColor: 'primary.500' }}>
                  <Typography level="body-sm">“{c}”</Typography>
                </Sheet>
              ))}
              {t.total > t.compromissos.length && (
                <Typography level="body-xs">
                  Mostrando os {t.compromissos.length} mais concretos de {t.total}. Os restantes estão no documento.
                </Typography>
              )}
            </Stack>
          </AccordionDetails>
        </Accordion>
      ))}
    </AccordionGroup>
  );
}

function PropostaGoverno({ refProposta }) {
  const [a, setA] = useState(null);
  const [erro, setErro] = useState(null);
  const [verPDF, setVerPDF] = useState(false);
  const [tentativa, setTentativa] = useState(0);
  const { ano, uf, sq, rotulo } = refProposta;

  useEffect(() => {
    setA(null);
    setTentativa(0);
  }, [ano, uf, sq]);

  useEffect(() => {
    const controller = new AbortController();
    setErro(null);
    buscarAPI(`/api/propostas?ano=${ano}&uf=${uf}&sq=${sq}`, controller.signal)
      .then(setA)
      .catch((err) => { if (err.name !== 'AbortError') setErro('Não foi possível obter a proposta no TSE agora.'); });
    return () => controller.abort();
  }, [ano, uf, sq, tentativa]);

  // Enquanto o servidor lê um PDF digitalizado (OCR), pergunta de novo a cada 10 segundos.
  useEffect(() => {
    if (!a?.processando) return undefined;
    const t = setTimeout(() => setTentativa((n) => n + 1), 10000);
    return () => clearTimeout(t);
  }, [a]);

  return (
    <Secao titulo={rotulo}>
      {erro && <Alert size="sm" color="warning" variant="soft">{erro}</Alert>}
      {!a && !erro && (
        <Stack direction="row" spacing={1} alignItems="center">
          <CircularProgress size="sm" /><Typography level="body-sm">Lendo a proposta entregue ao TSE…</Typography>
        </Stack>
      )}
      {a && !a.processando && a.arquivos.length === 0 && (
        <Typography level="body-sm">O TSE ainda não publicou a proposta de governo desta candidatura.</Typography>
      )}
      {a && a.arquivos.length > 0 && (
        <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" sx={{ mb: 2 }}>
          {a.arquivos.map((f) => (
            <Button key={f.url} component="a" href={`${API_URL}${f.url}`} target="_blank" rel="noopener noreferrer" size="sm" startDecorator="📄">
              Abrir proposta{a.arquivos.length > 1 ? ` (${f.nome})` : ''} · PDF {tamanhoMB(f.tamanho)} ↗
            </Button>
          ))}
          <Button size="sm" variant="outlined" color="neutral" onClick={() => setVerPDF(!verPDF)}>
            {verPDF ? 'Esconder o documento' : 'Ver o documento aqui'}
          </Button>
        </Stack>
      )}
      {a && verPDF && a.arquivos.length > 0 && (
        <Box
          component="iframe"
          title={`Proposta de governo: ${rotulo}`}
          src={`${API_URL}${a.arquivos[0].url}`}
          sx={{ width: '100%', height: '70vh', border: '1px solid', borderColor: 'divider', borderRadius: 'md', mb: 2 }}
        />
      )}

      {a?.processando && (
        <Alert size="sm" variant="soft" color="primary" startDecorator={<CircularProgress size="sm" />}>
          Este PDF foi entregue como imagem. Estamos lendo o texto com OCR, o que pode levar alguns minutos;
          o resumo aparece aqui sozinho quando terminar.
        </Alert>
      )}

      {a && !a.processando && a.arquivos.length > 0 && !a.textoDisponivel && (
        <Alert size="sm" variant="soft" color="neutral">
          {a.ocrDisponivel
            ? 'O PDF foi entregue como imagem e nem o OCR conseguiu ler texto suficiente. Abra o documento para ler as propostas.'
            : 'O PDF foi entregue como imagem (digitalizado). Para resumi-lo, o servidor precisa do OCR gratuito (Tesseract e Poppler) instalado. Abra o documento para ler as propostas.'}
        </Alert>
      )}

      {a && !a.processando && a.textoDisponivel && (
        <>
          {a.origem === 'ocr' && (
            <Alert size="sm" variant="soft" color="neutral" sx={{ mb: 2 }}>
              Texto lido da imagem do PDF com OCR: pode haver pequenos erros de leitura nas frases abaixo.
            </Alert>
          )}
          <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1.5fr 1fr' }, gap: 3 }}>
            <Box>
              <Typography level="title-sm">Resumo por tema</Typography>
              <Typography level="body-xs" sx={{ mb: 1 }}>
                Compromissos encontrados no documento, agrupados por área. As frases são transcritas do documento
                (as que têm números, mais concretas, aparecem primeiro).
              </Typography>
              <ResumoPorTema temas={a.resumoPorTema} />
            </Box>
            <Box>
              <Typography level="title-sm">Temas mais citados</Typography>
              <Typography level="body-xs" sx={{ mb: 1 }}>Quantas vezes cada tema aparece no texto ({a.palavras.toLocaleString('pt-BR')} palavras).</Typography>
              <Barras itens={a.temas.slice(0, 10).map((t) => ({ rotulo: t.rotulo, valor: t.total }))} />
            </Box>
          </Box>
        </>
      )}
    </Secao>
  );
}

function Propostas({ p }) {
  if (!p.propostas.length) {
    return (
      <Alert variant="soft" color="neutral">
        A lei só exige proposta de governo de candidatos a Presidente, Governador e Prefeito. Candidatos a Senador,
        Deputado e Vereador não entregam esse documento ao TSE.
      </Alert>
    );
  }
  return (
    <>
      {p.propostas.map((r) => <PropostaGoverno key={`${r.ano}-${r.sq}`} refProposta={r} />)}
      <Typography level="body-xs">
        Fonte: propostas de governo entregues ao TSE no registro da candidatura. O resumo é feito sem inteligência
        artificial: o programa só escolhe e arruma frases do próprio documento, por isso não substitui a sua leitura.
      </Typography>
    </>
  );
}

// ---------- Modal ----------

function Perfil({ id, onFechar }) {
  const [p, setP] = useState(null);
  const [erro, setErro] = useState(null);

  useEffect(() => {
    if (!id) return undefined;
    setP(null);
    setErro(null);
    const controller = new AbortController();
    buscarAPI(`/api/perfil?id=${encodeURIComponent(id)}`, controller.signal)
      .then(setP)
      .catch((err) => { if (err.name !== 'AbortError') setErro('Não foi possível carregar o perfil.'); });
    return () => controller.abort();
  }, [id]);

  const local = p && (p.municipio ? `${p.municipio} - ${p.uf}` : p.uf === 'BR' ? 'Brasil' : p.uf);

  return (
    <Modal open={!!id} onClose={onFechar}>
      <ModalDialog
        layout="center"
        sx={{ width: { xs: '100vw', md: 'min(980px, 94vw)' }, maxWidth: '100vw', height: { xs: '100dvh', md: '90vh' }, maxHeight: '100dvh', p: 0, overflow: 'hidden', borderRadius: { xs: 0, md: 'lg' } }}
      >
        <ModalClose sx={{ zIndex: 2 }} />
        {erro ? (
          <Box sx={{ p: 3 }}><Alert color="danger">{erro}</Alert></Box>
        ) : !p ? (
          <Stack alignItems="center" justifyContent="center" sx={{ height: '100%' }} spacing={1}>
            <CircularProgress />
            <Typography level="body-sm">Montando o perfil…</Typography>
          </Stack>
        ) : (
          <Box sx={{ overflowY: 'auto', height: '100%' }}>
            <Box sx={{ background: 'linear-gradient(135deg, #0B4A8B 0%, #0B6BCB 50%, #1A7F37 100%)', color: '#fff', p: { xs: 2, md: 3 } }}>
              <Stack direction="row" spacing={2.5} alignItems="center">
                <Box sx={{ bgcolor: '#fff', borderRadius: 'md', p: 0.5, flexShrink: 0 }}>
                  <Foto src={urlFoto(p.foto)} nome={p.pessoal.nomeUrna || p.pessoal.nome} largura={96} />
                </Box>
                <Box sx={{ minWidth: 0, pr: 4 }}>
                  <Typography level="h2" sx={{ color: '#fff' }}>{p.pessoal.nomeUrna || p.pessoal.nome}</Typography>
                  {p.pessoal.nome && <Typography level="body-sm" sx={{ color: 'rgba(255,255,255,0.85)' }}>{p.pessoal.nome}</Typography>}
                  <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" sx={{ mt: 1 }}>
                    <Chip variant="solid" sx={{ bgcolor: '#fff', color: 'primary.700' }}>{p.partido}</Chip>
                    {local && <Chip variant="soft" sx={{ bgcolor: 'rgba(255,255,255,0.2)', color: '#fff' }}>{local}</Chip>}
                    {p.pessoal.idade != null && <Chip variant="soft" sx={{ bgcolor: 'rgba(255,255,255,0.2)', color: '#fff' }}>{p.pessoal.idade} anos</Chip>}
                    {p.mandato && <Chip variant="soft" sx={{ bgcolor: 'rgba(255,255,255,0.2)', color: '#fff' }}>{p.mandato.cargo} · {p.mandato.periodo}</Chip>}
                  </Stack>
                </Box>
              </Stack>
            </Box>

            <Box sx={{ p: { xs: 2, md: 3 } }}>
              {p.avisos.map((a) => <Alert key={a} size="sm" variant="soft" color="warning" sx={{ mb: 1 }}>{a}</Alert>)}
              <Justica p={p} />
              <Indicadores indicadores={p.indicadores} />
              <Typography level="body-xs" sx={{ mb: 2 }}>
                Os pontos acima são calculados automaticamente a partir de dados públicos oficiais e não são juízos de valor.
              </Typography>

              <Tabs defaultValue={p.mandato ? 'mandato' : p.propostas.length ? 'propostas' : 'curriculo'} sx={{ bgcolor: 'transparent' }}>
                <TabList sx={{ overflowX: 'auto', flexWrap: 'nowrap', mb: 2 }}>
                  {p.mandato && <Tab value="mandato">Mandato</Tab>}
                  <Tab value="propostas">Propostas</Tab>
                  {Object.keys(p.campanha2026 || {}).length > 0 && <Tab value="campanha">Campanha</Tab>}
                  <Tab value="curriculo">Currículo</Tab>
                  {p.patrimonio && <Tab value="patrimonio">Patrimônio</Tab>}
                  <Tab value="historico">Histórico eleitoral</Tab>
                  <Tab value="noticias">Notícias</Tab>
                </TabList>
                {p.mandato && <TabPanel value="mandato" sx={{ p: 0 }}><Mandato m={p.mandato} /></TabPanel>}
                <TabPanel value="propostas" sx={{ p: 0 }}><Propostas p={p} /></TabPanel>
                <TabPanel value="campanha" sx={{ p: 0 }}><Campanha p={p} /></TabPanel>
                <TabPanel value="curriculo" sx={{ p: 0 }}><Curriculo p={p} /></TabPanel>
                {p.patrimonio && <TabPanel value="patrimonio" sx={{ p: 0 }}><Patrimonio pat={p.patrimonio} ano={p.anoEleicao2026} /></TabPanel>}
                <TabPanel value="historico" sx={{ p: 0 }}><Historico h={p.historico} /></TabPanel>
                <TabPanel value="noticias" sx={{ p: 0 }}><Noticias nome={p.nomeBusca} /></TabPanel>
              </Tabs>
              <Divider sx={{ my: 3 }} />
              <Typography level="body-xs">Fontes: TSE, Câmara dos Deputados e Senado Federal (dados abertos).</Typography>
            </Box>
          </Box>
        )}
      </ModalDialog>
    </Modal>
  );
}

export default Perfil;
