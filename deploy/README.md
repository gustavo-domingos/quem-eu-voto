# Publicar o backend

O backend usa cerca de 800 MB de RAM depois de carregar os dados, por isso os planos grátis de
512 MB (Render, Koyeb) não servem. A opção gratuita recomendada é uma VM **Oracle Cloud Always
Free** (Ampere A1, Ubuntu), que tem memória de sobra.

## Passos

1. Crie a VM e libere as portas **80** e **443**, tanto na *Security List* da Oracle quanto no
   firewall do Ubuntu.
2. Aponte um domínio para o IP da VM. Um subdomínio grátis do [DuckDNS](https://www.duckdns.org)
   serve.
3. Na VM, instale o Docker e copie o repositório:

   ```bash
   curl -fsSL https://get.docker.com | sh
   git clone <seu-repositório> quem-eu-voto && cd quem-eu-voto/deploy
   cp .env.example .env    # ajuste DOMINIO e CORS_ORIGINS
   sudo docker compose up -d --build
   ```

4. Confira: `curl https://SEU-DOMINIO/healthz` deve responder `ok`. O Caddy obtém o certificado
   HTTPS sozinho na primeira requisição.

Para atualizar: `git pull && sudo docker compose up -d --build`.

## Frontend

Publique a pasta `frontend` numa hospedagem estática (Cloudflare Pages, Vercel ou Netlify):

| Configuração        | Valor                          |
| ------------------- | ------------------------------ |
| Pasta raiz          | `frontend`                     |
| Comando de build    | `npm run build`                |
| Pasta de saída      | `build`                        |
| Variável            | `REACT_APP_API_URL=https://SEU-DOMINIO` |

Depois, coloque o endereço do site em `CORS_ORIGINS` no `.env` do servidor e reinicie com
`sudo docker compose up -d`.
