# Quem eu voto: frontend

Interface web em React 19 + [Joy UI](https://mui.com/joy-ui/getting-started/). Consome a API do
[backend](../backend) e não guarda dados próprios.

## Como rodar

```bash
npm install
npm start          # http://localhost:3000 (espera a API em http://localhost:8080)
npm test           # testes (Jest + Testing Library)
npm run build      # versão de produção em build/
```

A URL da API vem de `REACT_APP_API_URL` (veja [.env.example](.env.example)). Ela é lida **no
build**: ao mudar o endereço, é preciso gerar o build de novo.

## Estrutura

```
src/
├── index.js              ponto de entrada
├── app/                  casca da aplicação
│   ├── App.js            layout (menu lateral, cabeçalho, conteúdo, perfil)
│   ├── Navegacao.js      menu, logo, botão de tema, escolha Eleitos/Candidatos
│   ├── rotas.js          cargos, textos e roteamento por hash (useRota)
│   └── tema.js           tema do Joy UI
├── features/             uma pasta por funcionalidade
│   ├── candidatos/       candidaturas de 2026
│   ├── eleitos/          quem exerce o cargo hoje, com estatísticas
│   ├── perfil/           perfil completo (currículo, justiça, mandato, propostas…)
│   ├── apuracao/         apuração ao vivo do TSE
│   ├── partidos/         resumo por partido
│   └── vagas/            quantos representantes cada estado elege
├── components/           componentes reutilizados por várias features
├── services/api.js       cliente HTTP da API (buscarAPI, urlFoto)
└── utils/                constantes (UFs, faixas etárias) e formatação
```

Regras de dependência:

- `features/*` podem usar `components/`, `services/` e `utils/`. Uma feature só usa outra pelo
  componente que ela exporta (ex.: `ResultadoNaApuracao`).
- `components/` não conhecem as features.
- Só `services/api.js` faz pedidos HTTP.

## Rotas

A navegação usa o hash da URL, por isso funciona em qualquer hospedagem estática, sem configuração:

| Endereço                              | Página                                    |
| ------------------------------------- | ----------------------------------------- |
| `#senador`                            | senadores em exercício                    |
| `#senador/candidatos`                 | candidaturas ao Senado em 2026            |
| `#partidos`                           | partidos                                  |
| `#deputado-federal?p=camara-220556`   | página com o perfil de uma pessoa aberto  |
