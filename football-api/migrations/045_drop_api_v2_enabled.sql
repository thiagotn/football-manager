-- Remove a coluna api_v2_enabled da tabela players.
-- O gate de acesso por usuário à API v2 foi descontinuado: o ambiente de
-- homologação (api-go /api/v2) é isolado por subdomínio (beta.rachao.app) +
-- banco próprio, dispensando controle de acesso por flag.
--
-- ATENÇÃO (ordem de deploy): aplicar SOMENTE após o deploy do código que não
-- referencia mais esta coluna (Go api-go sem o campo ApiV2Enabled). A API Python
-- não usa esta coluna.
ALTER TABLE players
    DROP COLUMN IF EXISTS api_v2_enabled;
