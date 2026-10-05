import { Box, Card, Skeleton, Stack } from '@mui/joy';

export function Grade({ children, atualizando }) {
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(290px, 1fr))',
        gap: 2,
        opacity: atualizando ? 0.5 : 1,
        transition: 'opacity 0.2s',
      }}
    >
      {children}
    </Box>
  );
}

export function GradeCarregando() {
  return (
    <Grade>
      {Array.from({ length: 6 }, (_, i) => (
        <Card key={i} variant="outlined">
          <Stack direction="row" spacing={2}>
            <Skeleton variant="rectangular" width={84} height={112} sx={{ borderRadius: 'md' }} />
            <Box sx={{ flex: 1 }}>
              <Skeleton variant="text" level="h2" width="40%" />
              <Skeleton variant="text" level="title-lg" />
              <Skeleton variant="text" level="body-xs" width="70%" />
            </Box>
          </Stack>
          <Skeleton variant="text" level="body-sm" />
          <Skeleton variant="text" level="body-sm" width="60%" />
        </Card>
      ))}
    </Grade>
  );
}
