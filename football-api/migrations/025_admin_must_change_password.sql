-- Força o admin padrão (whatsapp 11999990000) a trocar a senha no primeiro acesso.
UPDATE players
SET must_change_password = TRUE
WHERE whatsapp = '11999990000'
  AND must_change_password = FALSE;
