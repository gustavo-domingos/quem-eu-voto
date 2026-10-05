// Package transparencia é o núcleo da aplicação: junta as fontes oficiais e monta o que o site
// mostra (listas de candidatos e eleitos, perfis, partidos, custos do mandato, índice de
// atuação). Não sabe nada de HTTP; os erros saem como *Erro, com o tipo e uma mensagem
// pronta para o utilizador.
package transparencia
