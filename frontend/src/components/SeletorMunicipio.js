import { useEffect, useState } from 'react';
import { Autocomplete, FormControl, FormLabel } from '@mui/joy';
import { buscarAPI } from '../services/api';

// Seletor de município (prefeito e vereador); carrega a lista da UF escolhida.
export function SeletorMunicipio({ uf, valor, onMudar }) {
  const [municipios, setMunicipios] = useState([]);

  useEffect(() => {
    if (uf === 'TODOS') {
      setMunicipios([]);
      return undefined;
    }
    const controller = new AbortController();
    buscarAPI(`/api/municipios?uf=${uf}`, controller.signal)
      .then((lista) => setMunicipios(lista || []))
      .catch((err) => { if (err.name !== 'AbortError') setMunicipios([]); });
    return () => controller.abort();
  }, [uf]);

  const selecionado = municipios.find((m) => m.codigo === valor) || null;
  return (
    <FormControl sx={{ flex: 1.5, minWidth: 220 }} disabled={uf === 'TODOS'}>
      <FormLabel>Município</FormLabel>
      <Autocomplete
        placeholder={uf === 'TODOS' ? 'Escolha um estado primeiro' : `Todos os ${municipios.length} municípios`}
        options={municipios}
        value={selecionado}
        getOptionLabel={(m) => m.nome}
        isOptionEqualToValue={(a, b) => a.codigo === b.codigo}
        onChange={(_, m) => onMudar(m ? m.codigo : 'TODOS')}
        noOptionsText="Nenhum município encontrado"
      />
    </FormControl>
  );
}
