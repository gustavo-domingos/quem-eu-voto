// Package carga mantém em memória os dados das fontes oficiais: Periodico recarrega um índice em
// segundo plano a intervalos fixos e PorID guarda valores por identificador (ex.: deputado),
// juntando pedidos simultâneos e limitando a concorrência.
package carga
