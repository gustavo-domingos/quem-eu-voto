# Quem eu voto

Transparência eleitoral para as eleições de 2026: quem são os candidatos, como os eleitos estão
trabalhando e quanto custam, com dados oficiais do TSE, da Câmara dos Deputados, do Senado
Federal e da CGU.

- **Eleitos:** presidente, governadores, senadores, deputados, prefeitos e vereadores, com
  presença, votações, projetos, gastos de gabinete e custo do mandato.
- **Candidatos 2026:** número na urna, patrimônio, contas de campanha, registros na Justiça,
  propostas de governo (com resumo por tema) e apuração ao vivo.
- **Perfil:** currículo, histórico eleitoral, processos e sanções, notícias e dados do mandato.

## Estrutura

```
quem-eu-voto/
├── backend/        API em Go (veja backend/README.md)
├── frontend/       site em React + Joy UI (veja frontend/README.md)
├── deploy/         docker compose + Caddy (HTTPS) para publicar o backend
├── docs/           arquitetura e decisões
└── .github/        integração contínua (testes e build a cada push)
```

## Rodar localmente

Requisitos: Go 1.27+ e Node.js 20+.

```bash
# terminal 1: API em http://localhost:8080
cd backend
go run ./cmd/api

# terminal 2: site em http://localhost:3000
cd frontend
npm install
npm start
```

## Testes

```bash
cd backend && go vet ./... && go test ./...
cd frontend && npm test -- --watchAll=false
```

## Publicar

- **Frontend:** qualquer hospedagem estática (Cloudflare Pages, Vercel, Netlify): pasta
  `frontend`, comando `npm run build`, saída `build`, variável `REACT_APP_API_URL` com o endereço
  da API.
- **Backend:** precisa de ~1 GB de RAM. Veja [deploy/README.md](deploy/README.md).

## Fontes

Todos os dados vêm de bases públicas: [TSE – dados abertos](https://dadosabertos.tse.jus.br),
[Câmara dos Deputados](https://dadosabertos.camara.leg.br),
[Senado Federal](https://www12.senado.leg.br/dados-abertos),
[Portal da Transparência (CGU)](https://portaldatransparencia.gov.br/download-de-dados) e
[GDELT](https://www.gdeltproject.org).
