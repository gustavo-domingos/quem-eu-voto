import { useState } from 'react';
import { AspectRatio, Typography } from '@mui/joy';

// Foto com fundo neutro e iniciais quando a imagem não existe.
export function Foto({ src, nome, largura = 84 }) {
  const [falhou, setFalhou] = useState(false);
  const iniciais = nome.split(/\s+/).filter(Boolean).slice(0, 2).map((p) => p[0]).join('');
  return (
    <AspectRatio ratio="3/4" variant="soft" sx={{ width: largura, flexShrink: 0, borderRadius: 'md' }}>
      {falhou || !src ? (
        <Typography level="title-lg" color="neutral">{iniciais}</Typography>
      ) : (
        <img src={src} alt={nome} loading="lazy" onError={() => setFalhou(true)} />
      )}
    </AspectRatio>
  );
}
