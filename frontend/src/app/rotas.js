import { useEffect, useState } from 'react';

// Cada cargo tem duas vistas: quem está no cargo hoje e quem concorre em 2026 (nos municipais, só eleitos).
export const CARGOS = [
  { cargo: 'presidente', titulo: 'Presidente', icone: '⭐' },
  { cargo: 'governador', titulo: 'Governador', icone: '🏛️' },
  { cargo: 'senador', titulo: 'Senador', icone: '⚖️' },
  { cargo: 'deputado-federal', titulo: 'Deputado Federal', icone: '🏢' },
  { cargo: 'deputado-estadual', titulo: 'Deputado Estadual', icone: '🏫' },
  { cargo: 'prefeito', titulo: 'Prefeito', icone: '🏙️', municipal: true },
  { cargo: 'vereador', titulo: 'Vereador', icone: '🗳️', municipal: true },
];

export const DESCRICAO_ELEITOS = {
  presidente: 'Quem governa o país: votação em 2022 e se concorre em 2026.',
  governador: 'Governadores eleitos em 2022, votação e o que fazem em 2026. Quem concorre a outro cargo já deixou o mandato.',
  senador: 'Os 81 senadores em exercício, com a participação nas votações nominais e os projetos apresentados desde 2023.',
  'deputado-federal': 'Os 513 deputados em exercício, com assiduidade, projetos e votação em 2022.',
  'deputado-estadual': 'Deputados estaduais e distritais eleitos em 2022, com a votação e o que fazem em 2026.',
  prefeito: 'Prefeitos eleitos em 2024 (incluindo eleições suplementares), com o vice e a votação.',
  vereador: 'Vereadores eleitos em 2024, com a votação e quem concorre a outro cargo em 2026.',
};

export const PARTIDOS = { id: 'partidos', titulo: 'Partidos políticos', icone: '🎌' };

export const ITENS_MENU = [...CARGOS.map((c) => ({ ...c, id: c.cargo })), PARTIDOS];

// Endereço: "#governador" (eleitos), "#governador/candidatos" ou "#partidos";
// com um perfil aberto acrescenta "?p=<id>" (ex.: "#deputado-federal?p=camara-220556").
function rotaDoEndereco() {
  const [caminho, consulta = ''] = window.location.hash.replace('#', '').split('?');
  const perfil = new URLSearchParams(consulta).get('p');
  const [id, vista] = caminho.split('/');
  if (id === PARTIDOS.id) return { id, vista: null, perfil };
  const cargo = CARGOS.find((c) => c.cargo === id) || CARGOS[0];
  return { id: cargo.cargo, vista: vista === 'candidatos' && !cargo.municipal ? 'candidatos' : 'eleitos', perfil };
}

function hashDe({ id, vista, perfil }) {
  return (vista === 'candidatos' ? `${id}/candidatos` : id) + (perfil ? `?p=${encodeURIComponent(perfil)}` : '');
}

export const descricaoCandidatos = (titulo) => `Candidaturas a ${titulo} nas eleições de 2026, segundo o TSE.`;

// useRota guarda a página atual no endereço (#cargo[/candidatos][?p=perfil]), para sobreviver
// a um refresh e ao botão "voltar" do navegador.
export function useRota() {
  const [rota, setRota] = useState(rotaDoEndereco);

  useEffect(() => {
    const aoMudar = () => setRota(rotaDoEndereco());
    window.addEventListener('hashchange', aoMudar);
    return () => window.removeEventListener('hashchange', aoMudar);
  }, []);

  const ir = (nova) => {
    window.location.hash = hashDe(nova);
    setRota(nova);
  };

  // navegar muda de página (cargo/vista) e fecha o perfil aberto.
  const navegar = (id, vista) => {
    const alvo = CARGOS.find((c) => c.cargo === id);
    const v = !alvo ? null : vista === 'candidatos' && !alvo.municipal ? 'candidatos' : 'eleitos';
    ir({ id, vista: v, perfil: null });
    window.scrollTo({ top: 0 });
  };

  const mudarPerfil = (perfil) => ir({ ...rota, perfil });

  return { rota, navegar, mudarPerfil };
}
