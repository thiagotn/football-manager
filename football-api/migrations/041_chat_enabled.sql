ALTER TABLE players
ADD COLUMN IF NOT EXISTS chat_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN players.chat_enabled IS
  'Controla se o usuário tem acesso ao assistente de IA em rachao.app/chat. Gerenciado pelo admin. Padrão: FALSE.';

ALTER TABLE players
ADD COLUMN IF NOT EXISTS chat_req_count INT NOT NULL DEFAULT 0;

ALTER TABLE players
ADD COLUMN IF NOT EXISTS chat_req_window TIMESTAMPTZ;
