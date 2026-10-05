import { useState } from 'react';
import { Button, Input, Stack, Typography } from '@mui/joy';

// Números de página a mostrar: sempre a primeira e a última, e a atual com `vizinhas` de
// cada lado; os saltos viram "…". Ex.: 1 … 5 6 [7] 8 9 … 232.
export function paginasVisiveis(pagina, total, vizinhas = 2) {
  const paginas = [];
  for (let p = 1; p <= total; p++) {
    if (p === 1 || p === total || Math.abs(p - pagina) <= vizinhas) {
      // Um salto de uma só página mostra o número em vez de "…".
      if (paginas.length && p - paginas[paginas.length - 1] === 2) paginas.push(p - 1);
      else if (paginas.length && p - paginas[paginas.length - 1] > 2) paginas.push('…');
      paginas.push(p);
    }
  }
  return paginas;
}

function IrParaPagina({ totalPaginas, desativado, onMudar }) {
  const [valor, setValor] = useState('');
  const n = Number(valor);
  const valido = Number.isInteger(n) && n >= 1 && n <= totalPaginas;
  const ir = (e) => {
    e.preventDefault();
    if (!valido) return;
    onMudar(n);
    setValor('');
  };
  return (
    <Stack component="form" direction="row" spacing={1} alignItems="center" onSubmit={ir}>
      <Typography level="body-sm">Ir para a página</Typography>
      <Input
        size="sm"
        type="number"
        value={valor}
        onChange={(e) => setValor(e.target.value)}
        placeholder={`1–${totalPaginas}`}
        slotProps={{ input: { min: 1, max: totalPaginas, 'aria-label': 'Número da página' } }}
        error={valor !== '' && !valido}
        sx={{ width: 96 }}
      />
      <Button size="sm" type="submit" variant="soft" disabled={!valido || desativado}>Ir</Button>
    </Stack>
  );
}

export function Paginacao({ pagina, totalPaginas, desativado, onMudar }) {
  if (totalPaginas <= 1) return null;
  const botao = { size: 'sm', color: 'neutral', disabled: desativado, sx: { minWidth: 36, px: 1 } };
  return (
    <Stack spacing={1.5} alignItems="center" sx={{ my: 4 }}>
      <Stack component="nav" aria-label="Paginação" direction="row" spacing={0.5} useFlexGap flexWrap="wrap" justifyContent="center" alignItems="center">
        <Button {...botao} variant="outlined" disabled={pagina <= 1 || desativado} onClick={() => onMudar(1)} aria-label="Primeira página">«</Button>
        <Button {...botao} variant="outlined" disabled={pagina <= 1 || desativado} onClick={() => onMudar(pagina - 1)} aria-label="Página anterior">‹</Button>
        {paginasVisiveis(pagina, totalPaginas).map((p, i) => (p === '…' ? (
          <Typography key={`r${i}`} level="body-sm" sx={{ px: 0.5 }}>…</Typography>
        ) : (
          <Button
            key={p}
            {...botao}
            variant={p === pagina ? 'solid' : 'plain'}
            color={p === pagina ? 'primary' : 'neutral'}
            aria-current={p === pagina ? 'page' : undefined}
            onClick={() => p !== pagina && onMudar(p)}
          >
            {p}
          </Button>
        )))}
        <Button {...botao} variant="outlined" disabled={pagina >= totalPaginas || desativado} onClick={() => onMudar(pagina + 1)} aria-label="Página seguinte">›</Button>
        <Button {...botao} variant="outlined" disabled={pagina >= totalPaginas || desativado} onClick={() => onMudar(totalPaginas)} aria-label="Última página">»</Button>
      </Stack>
      {totalPaginas > 7 && <IrParaPagina totalPaginas={totalPaginas} desativado={desativado} onMudar={onMudar} />}
    </Stack>
  );
}
