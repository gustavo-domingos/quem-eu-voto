# syntax=docker/dockerfile:1

# ---------- build ----------
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/quem-eu-voto ./cmd/api

# ---------- execução ----------
FROM alpine:3.22
# Poppler + Tesseract (português) fazem o OCR das propostas digitalizadas. São opcionais:
# sem eles a API funciona, só não lê PDFs que são imagens.
RUN apk add --no-cache ca-certificates tzdata poppler-utils tesseract-ocr tesseract-ocr-data-por \
    && adduser -D -u 10001 app \
    && mkdir /dados && chown app /dados

ENV TZ=America/Sao_Paulo \
    PORT=8080 \
    CACHE_DIR=/dados

COPY --from=build /out/quem-eu-voto /usr/local/bin/quem-eu-voto

USER app
# A cache em disco (listas da Câmara, contas de campanha, análises de propostas) fica num volume
# para o servidor voltar rápido depois de um reinício.
VOLUME /dados
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s CMD wget -qO- http://localhost:8080/healthz || exit 1
ENTRYPOINT ["/usr/local/bin/quem-eu-voto"]
