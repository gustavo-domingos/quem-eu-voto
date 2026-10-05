export function formatarData(iso) {
  // "2023-02-01" seria interpretado como UTC e mostrado como 31/01 no Brasil.
  const data = iso.length === 10 ? new Date(`${iso}T12:00:00`) : new Date(iso);
  return data.toLocaleDateString('pt-BR');
}
