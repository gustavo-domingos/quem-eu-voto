// Package tse lê os dados abertos do Tribunal Superior Eleitoral: candidaturas, bens, contas de
// campanha, votação por candidato, vagas por cargo, fotos e a apuração ao vivo de 2026.
//
// Os ficheiros grandes (fotos, votação) são lidos por partes, com pedidos HTTP Range, sem
// descarregar o zip inteiro (ver internal/plataforma/zipremoto).
package tse
