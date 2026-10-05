import { FormControl, FormLabel, Option, Select } from '@mui/joy';
import { FAIXAS_ETARIAS } from '../utils/constantes';

// Filtros de partido, gênero e faixa etária (comuns a eleitos e candidatos).
export function FiltrosPessoa({ partidos, partido, genero, faixa, onPartido, onGenero, onFaixa }) {
  return (
    <>
      <FormControl sx={{ flex: 1, minWidth: 160 }}>
        <FormLabel>Partido</FormLabel>
        <Select value={partido} onChange={(_, v) => onPartido(v)}>
          <Option value="">Todos os partidos</Option>
          {partido && !partidos?.includes(partido) && <Option value={partido}>{partido}</Option>}
          {partidos?.map((p) => <Option key={p} value={p}>{p}</Option>)}
        </Select>
      </FormControl>
      <FormControl sx={{ flex: 1, minWidth: 140 }}>
        <FormLabel>Gênero</FormLabel>
        <Select value={genero} onChange={(_, v) => onGenero(v)}>
          <Option value="">Todos</Option>
          <Option value="F">Mulheres</Option>
          <Option value="M">Homens</Option>
        </Select>
      </FormControl>
      <FormControl sx={{ flex: 1, minWidth: 140 }}>
        <FormLabel>Faixa etária</FormLabel>
        <Select value={faixa} onChange={(_, v) => onFaixa(v)}>
          <Option value="">Todas</Option>
          {FAIXAS_ETARIAS.map((f) => <Option key={f} value={f}>{f}</Option>)}
        </Select>
      </FormControl>
    </>
  );
}
