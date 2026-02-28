-- ============================================================
-- 002_seed_dev.sql
-- Carga inicial de desenvolvimento: grupo Futebol GQC + jogadores
-- Senha padrão de todos os jogadores: joga123
-- ============================================================

-- ── Grupo ─────────────────────────────────────────────────────
INSERT INTO groups (name, description, slug) VALUES
  ('Futebol GQC', 'Grupo de futebol GQC', 'futebol-gqc')
ON CONFLICT (slug) DO NOTHING;

-- ── Jogadores ─────────────────────────────────────────────────
-- Senha padrão: joga123 (gerada via pgcrypto Blowfish, compatível com bcrypt)
INSERT INTO players (name, nickname, whatsapp, password_hash, role) VALUES
  ('Thiago Nogueira',         'Thiagol',            '11991915070', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Claudio Reboucas',        'Claudio',             '11973836269', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Danilo',                  'Danilo',              '11953563694', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Eduardo Franklin',        'Dudu',                '11966137434', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Leonardo Ambrosio',       'Le',                  '11981977351', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Adriano Coimbra',         'Adriano Coimbra',     '11983840508', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Augusto',                 'Augusto',             '11994999192', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Cleber Nunes',            'Cleber',              '11991778799', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Daniel Leme',             'Daniel Leme',         '11964287513', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Fe Barrasoccer',          'Fe Barrasoccer',      '11947121698', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Rodrigo Rafaine',         'Guigo',               '11994688332', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Guilherme Trindade',      'Guilherme Trindade',  '11981067557', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Gustavo',                 'Gustavo',             '11994170413', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Jonathas Moreira Lemos',  'Jo',                  '11943662109', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Carlos Carvalho',         'Kafa',                '11982728050', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Lucas',                   'Lucas',               '11977140150', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Pedro',                   'Pedro Violeiro',      '11977305373', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Rafael Rocha',            'Skate',               '11964060250', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Regis Orioni',            'Regis',               '11996288336', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Renan',                   'Renan',               '11991640484', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Rian',                    'Rian',                '11970353456', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Vinicius',                'Vini',                '11963955894', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Francisco Dantas',        'Francisco Dantas',    '11976272873', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Guaracy',                 'Guaracy',             '8192425775',  crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('PP',                      'PP',                  '13997584227', crypt('joga123', gen_salt('bf', 12)), 'player'),
  ('Yago',                    'Yago',                '11961983670', crypt('joga123', gen_salt('bf', 12)), 'player')
ON CONFLICT (whatsapp) DO NOTHING;

-- ── Membros do grupo (todos os jogadores acima) ────────────────
INSERT INTO group_members (group_id, player_id, role)
SELECT g.id, p.id, 'member'
FROM groups g
CROSS JOIN players p
WHERE g.slug = 'futebol-gqc'
  AND p.whatsapp IN (
    '11991915070', '11973836269', '11953563694', '11966137434',
    '11981977351', '11983840508', '11994999192', '11991778799',
    '11964287513', '11947121698', '11994688332', '11981067557',
    '11994170413', '11943662109', '11982728050', '11977140150',
    '11977305373', '11964060250', '11996288336', '11991640484',
    '11970353456', '11963955894', '11976272873', '8192425775',
    '13997584227', '11961983670'
  )
ON CONFLICT (group_id, player_id) DO NOTHING;

-- ── Admin também entra no grupo como admin ─────────────────────
INSERT INTO group_members (group_id, player_id, role)
SELECT g.id, p.id, 'admin'
FROM groups g
CROSS JOIN players p
WHERE g.slug = 'futebol-gqc'
  AND p.whatsapp = '11999990000'
ON CONFLICT (group_id, player_id) DO NOTHING;
