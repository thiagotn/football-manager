// Package services — voting window helpers (mirror of football-api/app/services/voting.py).
//
// Mantém paridade com a v1: janela abre `vote_open_delay_minutes` após o
// `end_time` (ou 23:59 se nulo) e dura `vote_duration_hours`. Todos os
// cálculos são em America/Sao_Paulo.
package services

import (
	"time"
)

// VotingStatus enumera os estados da janela de votação de uma partida.
type VotingStatus string

const (
	VotingStatusNotOpen VotingStatus = "not_open"
	VotingStatusOpen    VotingStatus = "open"
	VotingStatusClosed  VotingStatus = "closed"
)

// VotingInput carrega só o que o cálculo precisa, evitando acoplar a função a
// um struct específico (db.Match vs db.PlayerMatch ambos podem fornecer).
type VotingInput struct {
	MatchDate            string  // "YYYY-MM-DD"
	EndTime              *string // "HH:MM:SS" ou nil
	StartTime            string  // fallback se EndTime nil — alinha com v1 mas o padrão da v1 é 23:59; aqui guardamos para fallback futuro
	VoteOpenDelayMinutes int
	VoteDurationHours    int
}

// VotingWindow devolve (opensAt, closesAt) em UTC para um match.
// Retorna ok=false se os campos não puderem ser parseados.
func VotingWindow(in VotingInput) (opensAt, closesAt time.Time, ok bool) {
	matchDate, err := time.Parse("2006-01-02", in.MatchDate)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}

	// Default end_time = 23:59 (paridade v1 voting.py).
	endTimeStr := "23:59:00"
	if in.EndTime != nil && *in.EndTime != "" {
		endTimeStr = *in.EndTime
	}
	endTime, err := time.Parse("15:04:05", endTimeStr)
	if err != nil {
		// tenta "HH:MM"
		endTime, err = time.Parse("15:04", endTimeStr)
		if err != nil {
			return time.Time{}, time.Time{}, false
		}
	}

	delay := time.Duration(in.VoteOpenDelayMinutes) * time.Minute
	if in.VoteOpenDelayMinutes == 0 {
		delay = 20 * time.Minute
	}
	duration := time.Duration(in.VoteDurationHours) * time.Hour
	if in.VoteDurationHours == 0 {
		duration = 24 * time.Hour
	}

	// Constrói o instante BRT e converte pra UTC somando 3h (BRT = UTC-3).
	endBRT := time.Date(
		matchDate.Year(), matchDate.Month(), matchDate.Day(),
		endTime.Hour(), endTime.Minute(), endTime.Second(), 0,
		time.UTC,
	).Add(3 * time.Hour)

	opensAt = endBRT.Add(delay)
	closesAt = opensAt.Add(duration)
	return opensAt, closesAt, true
}

// ComputeVotingStatus devolve "not_open" | "open" | "closed" para a partida.
// Se os timestamps não puderem ser parseados, devolve "closed" (mais seguro).
func ComputeVotingStatus(in VotingInput) VotingStatus {
	opensAt, closesAt, ok := VotingWindow(in)
	if !ok {
		return VotingStatusClosed
	}
	now := time.Now().UTC()
	if now.Before(opensAt) {
		return VotingStatusNotOpen
	}
	if !now.After(closesAt) {
		return VotingStatusOpen
	}
	return VotingStatusClosed
}
