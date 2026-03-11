-- Migration: 019_match_voting_snapshot.sql
-- Snapshot das configs de votação do grupo no momento da criação da partida

ALTER TABLE matches
  ADD COLUMN vote_open_delay_minutes INT NOT NULL DEFAULT 20,
  ADD COLUMN vote_duration_hours     INT NOT NULL DEFAULT 24;
