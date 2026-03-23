-- Backfill: insere presença 'pending' para membros de grupo que não têm
-- registro de presença em partidas abertas ou em andamento.
--
-- Cenário corrigido: jogadores aceitos via lista de espera eram adicionados
-- apenas à partida específica da candidatura, ficando ausentes das demais
-- partidas ativas do grupo.

INSERT INTO attendances (id, match_id, player_id, status, created_at, updated_at)
SELECT
    gen_random_uuid(),
    m.id,
    gm.player_id,
    'pending',
    now(),
    now()
FROM matches m
JOIN group_members gm ON gm.group_id = m.group_id
JOIN players p ON p.id = gm.player_id
LEFT JOIN attendances a ON a.match_id = m.id AND a.player_id = gm.player_id
WHERE m.status IN ('open', 'in_progress')
  AND p.role != 'admin'
  AND a.id IS NULL;
