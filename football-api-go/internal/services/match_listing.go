// Package services — match listing classifier (mirror of
// football-api/app/services/match_listing.py).
//
// Decide se cada partida pertence ao card "Atuais" do frontend, usando 3 sinais:
//  1. status `open` ou `in_progress` (jogo por vir / acontecendo).
//  2. janela de votação ainda não fechou.
//  3. é a partida mais recente do grupo E não há partida futura criada ainda
//     (evita que a aba "Atuais" fique vazia entre rachões).
//
// PRD 044 §17 — item de paridade entregue.
package services

import (
	"github.com/google/uuid"
)

// ListingMatch é o subset mínimo de campos necessários pra classificar.
// Tanto db.Match quanto db.PlayerMatch podem fornecer esses dados.
type ListingMatch struct {
	ID                   uuid.UUID
	Status               string // "open" | "in_progress" | "closed"
	MatchDate            string // "YYYY-MM-DD"
	StartTime            string // "HH:MM:SS"
	EndTime              *string
	VoteOpenDelayMinutes int
	VoteDurationHours    int
}

// ClassificationResult armazena os dois campos derivados por partida.
type ClassificationResult struct {
	IsCurrent     bool
	VotingStatus  VotingStatus
}

// ClassifyMatches recebe a lista completa de partidas DE UM MESMO GRUPO e
// devolve um mapa MatchID → (IsCurrent, VotingStatus). Se a lista cruzar
// múltiplos grupos, agrupe antes de chamar (ver handlers/players.go).
func ClassifyMatches(matches []ListingMatch) map[uuid.UUID]ClassificationResult {
	out := make(map[uuid.UUID]ClassificationResult, len(matches))
	if len(matches) == 0 {
		return out
	}

	hasFuture := false
	for _, m := range matches {
		if m.Status == "open" || m.Status == "in_progress" {
			hasFuture = true
			break
		}
	}

	// Determinar a partida fechada mais recente (por match_date, start_time desc).
	var mostRecentClosedID uuid.UUID
	var mostRecentClosedKey string
	for _, m := range matches {
		if m.Status != "closed" {
			continue
		}
		key := m.MatchDate + "T" + m.StartTime
		if key > mostRecentClosedKey {
			mostRecentClosedKey = key
			mostRecentClosedID = m.ID
		}
	}

	for _, m := range matches {
		vstatus := ComputeVotingStatus(VotingInput{
			MatchDate:            m.MatchDate,
			StartTime:            m.StartTime,
			EndTime:              m.EndTime,
			VoteOpenDelayMinutes: m.VoteOpenDelayMinutes,
			VoteDurationHours:    m.VoteDurationHours,
		})

		isCurrent := false
		switch {
		case m.Status == "open" || m.Status == "in_progress":
			isCurrent = true
		case vstatus == VotingStatusNotOpen || vstatus == VotingStatusOpen:
			isCurrent = true
		case m.ID == mostRecentClosedID && !hasFuture:
			isCurrent = true
		}

		out[m.ID] = ClassificationResult{
			IsCurrent:    isCurrent,
			VotingStatus: vstatus,
		}
	}
	return out
}
