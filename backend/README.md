# Quem eu voto: backend

API em Go (só biblioteca padrão, mais `ledongthuc/pdf` para ler PDFs) que junta os dados abertos
do TSE, da Câmara, do Senado, da CGU e do GDELT e os entrega prontos para o frontend.

## Como rodar

```bash
go run ./cmd/api        # http://localhost:8080
go test ./...
go vet ./...
```

Na primeira execução, o servidor baixa vários arquivos oficiais (alguns com GB, lidos por partes).
Os índices ficam em memória (cerca de 800 MB de RAM) e o que é caro de obter fica em cache no disco.

| Variável        | Padrão                         | Para quê                                        |
| --------------- | ------------------------------ | ----------------------------------------------- |
| `PORT`          | `8080`                         | porta HTTP                                      |
| `CORS_ORIGINS`  | vazio (qualquer origem)        | origens do frontend, separadas por vírgula      |
| `CACHE_DIR`     | pasta de cache do usuário      | onde gravar a cache em disco                    |

OCR (opcional): com `pdftoppm` (Poppler) e `tesseract` no PATH, as propostas digitalizadas também
são resumidas. No Windows: `winget install oschwartz10612.Poppler UB-Mannheim.TesseractOCR`.

## Estrutura

```
cmd/api/                    main: configuração, ligação das dependências, encerramento gracioso
internal/
├── api/                    HTTP: rotas, middlewares (CORS, recover, log), parâmetros → serviço → JSON
├── transparencia/          regras da aplicação: listas, perfis, partidos, custos, índice de atuação
├── propostas/              propostas de governo (PDF): listagem e resumo por tema, sem IA
├── fontes/                 um pacote por fonte oficial
│   ├── tse/                candidaturas, bens, contas, votação, vagas, fotos, apuração ao vivo
│   ├── camara/             deputados, presença, votações, cota, gabinetes
│   ├── senado/             senadores, votações, cota (CEAPS)
│   ├── sancoes/            CGU (CEIS, CNEP, CEAF) e cassações do TSE
│   └── noticias/           GDELT
├── dominio/                conceitos partilhados: calendário eleitoral, cargos, UFs, tipos simples
├── config/                 variáveis de ambiente
└── plataforma/             infraestrutura sem regra de negócio
    ├── rede/               HTTP com novas tentativas
    ├── carga/              índices recarregados em segundo plano e cache por ID
    ├── cache/              cache em disco (JSON)
    ├── planilha/           CSV (Latin-1, BOM, dentro de zip)
    ├── zipremoto/          leitura de zips remotos por HTTP Range
    ├── ocr/                Poppler + Tesseract
    └── texto/              normalização de nomes, datas e números
```

As camadas e as regras de dependência estão em [docs/arquitetura.md](../docs/arquitetura.md).

## Endpoints

Todos respondem JSON (exceto fotos e PDFs) e aceitam só `GET`.

| Rota                                          | Descrição                                                    |
| --------------------------------------------- | ------------------------------------------------------------ |
| `/healthz`                                    | verificação de saúde (`ok`)                                  |
| `/api/candidatos?cargo=&uf=&…`                | candidaturas de 2026, com filtros, ordenação e página        |
| `/api/eleitos?cargo=&uf=&municipio=&…`        | quem exerce o cargo hoje, com estatísticas                   |
| `/api/deputados?uf=`                          | deputados federais em exercício                              |
| `/api/perfil?id=`                             | perfil completo (SQ do TSE, `camara-<id>` ou `senado-<id>`)  |
| `/api/partidos`                               | resumo por partido                                           |
| `/api/municipios?uf=`                         | municípios da UF                                             |
| `/api/vagas?uf=&municipio=`                   | quantos representantes o local elege                         |
| `/api/apuracao?cargo=&uf=`                    | apuração ao vivo do TSE                                      |
| `/api/propostas?ano=&uf=&sq=`                 | propostas de governo e resumo por tema                       |
| `/api/propostas/arquivo/{ano}/{uf}/{nome}`    | PDF da proposta                                              |
| `/api/noticias?nome=`                         | notícias recentes (GDELT)                                    |
| `/api/fotos/{ano}/{uf}/{sq}`                  | foto oficial da candidatura                                  |

Erros: `400` (parâmetro inválido), `404` (não encontrado), `502` (fonte oficial fora do ar),
com a mensagem em texto no corpo.
