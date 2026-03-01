-- ============================================================
-- 004_match_address.sql
-- Adiciona endereço opcional para exibição no Google Maps
-- ============================================================

ALTER TABLE matches
  ADD COLUMN IF NOT EXISTS address VARCHAR(300) NULL;
