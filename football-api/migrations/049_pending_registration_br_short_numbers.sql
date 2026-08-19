-- Backfill complementar da 048: números brasileiros (+55) fora do padrão
-- DDD (2 dígitos) + telefone (8 ou 9 dígitos). O regex genérico E.164 da 048
-- (7–15 dígitos) deixava passar placeholders curtos como +55117458, que nunca
-- receberão OTP. Contas assim ficam órfãs e devem ser marcadas como pendentes
-- para permitir o fluxo de claim.
-- Super admins ficam de fora: o claim-invite recusa esse alvo, então a flag
-- geraria um badge sem correção possível.
UPDATE players p
SET pending_registration = TRUE
WHERE p.pending_registration = FALSE
  AND p.role != 'admin'
  AND p.whatsapp ~ '^\+55'
  AND p.whatsapp !~ '^\+55[0-9]{10,11}$';
