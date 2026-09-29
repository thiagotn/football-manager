-- Migration: 054_match_teams_draw_strategy.sql
-- Persiste a estratégia usada no sorteio de times.
--
-- Até aqui a estratégia ('balanced' | 'simple') era só um parâmetro do
-- POST /matches/{id}/teams e era descartada no fim do request (decisão
-- registrada no PRD 044 §17). Isso passou a ser um problema: no sorteio
-- simplificado as posições de linha são ignoradas de propósito, então a tela
-- de times não deve exibi-las — e para saber disso ao renderizar um sorteio
-- já existente é preciso ter a estratégia gravada.
--
-- Fica em match_teams (e não em matches) porque o sorteio já apaga e recria
-- todas as linhas de match_teams a cada execução: o valor é reescrito junto
-- com os times e nunca fica obsoleto.
--
-- O DEFAULT 'balanced' faz o backfill dos sorteios já existentes, que sempre
-- exibiram as posições e continuam exibindo.

ALTER TABLE match_teams
    ADD COLUMN IF NOT EXISTS draw_strategy VARCHAR(16) NOT NULL DEFAULT 'balanced';

ALTER TABLE match_teams
    DROP CONSTRAINT IF EXISTS match_teams_draw_strategy_check;

ALTER TABLE match_teams
    ADD CONSTRAINT match_teams_draw_strategy_check
    CHECK (draw_strategy IN ('balanced', 'simple'));
