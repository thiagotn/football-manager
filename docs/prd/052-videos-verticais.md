# PRD 052 — Vídeos Curtos Verticais por Rachão (experimental)

- **Status:** 🚧 Em implementação (2026-08-20)
- **Feature flag:** `players.videos_enabled` — habilitada pelo super admin em `/admin/videos`, por conta. Vale para **todos os grupos que o dono da conta administra**.
- **ADR de infra:** homelab `docs/adr/0007-videos-curtos-transcode-worker.md`

## Problema / Objetivo

Para cada rachão realizado, membros querem registrar e rever gols e lances (pré, durante ou pós).
Objetivo: upload de vídeos **verticais de até 1 minuto** por partida, assistíveis na própria página
pública do rachão (como times e resultados) — vitrine do rachão e do app.

## Decisões de produto

| Decisão | Valor |
|---|---|
| Flag | Por conta (`videos_enabled`), semântica "dono do grupo": habilita nos grupos onde a conta é admin. Grupo com 2+ admins habilita se **qualquer** um tiver a flag. Flag off = kill switch (esconde vídeos já publicados). |
| Quem sobe | Membros com presença **confirmada** na partida + admins do grupo + super admin |
| Quem exclui | Autor (o próprio vídeo), admin do grupo e super admin (qualquer vídeo) |
| Visibilidade | Pública em `/match/[hash]/feed` (redirect de `/videos`) — feed fullscreen estilo TikTok (snap vertical, autoplay mudo, swipe); deep link `?item=<id>`; compartilhar via Web Share/WhatsApp |
| Curtidas | Qualquer usuário logado curte (idempotente); lista de quem curtiu é pública (migration 051, `match_video_likes`) |
| Fotos | Mesmo feed aceita imagens (JPG/PNG/WebP, ≤25MB; migration 052 `media_type`); worker corrige EXIF + redimensiona ≤1080×1920 + JPEG; URL final em `poster_url` (`video_url` nulo) |
| Views | Contador por item (migration 053 `view_count`; `POST /videos/{id}/view` público, dedupe por visita no front); exibido no rail do feed |
| Progresso | Linha estilo TikTok no rodapé do slide: vídeo acompanha a reprodução, foto tem timer de 6s; ao completar, dica animada de swipe se houver próximo item |
| Limites | 10 vídeos/partida (fixo, `VIDEO_LIMIT_REACHED`), 150MB original, ≤ 65s (ffprobe é a autoridade; client valida advisory) |

## Arquitetura

```
Browser ─(1) POST /api/v2/matches/{id}/videos ───► API v2 (valida flag/membership/limite; row pending; presigned PUT)
Browser ─(2) PUT presigned ──────────────────────► R2 videos/original/{match_id}/{video_id}.mp4
Browser ─(3) POST .../videos/{vid}/confirm ──────► API v2 (StatObject; status→uploaded)
Worker  ─(4) poll 5s FOR UPDATE SKIP LOCKED ─────► claim (uploaded→processing; reclaim >15min)
Worker  ─(5) ffprobe → x264 720×1280 → poster ───► R2 videos/{match_id}/{vid}.{mp4,webp}; ready; apaga original
Browser ─(6) GET /api/v2/matches/public/{hash}/videos ► feed público via cdn.rachao.app
```

- Upload **não passa pela API** (ReadTimeout 15s) nem pelo túnel CF (cap 100MB): presigned PUT
  direto no endpoint S3 do R2 (`minio-go PresignedPutObject`, expira em 15min).
- Transcode obrigatório (iPhone grava HEVC → não toca em Chrome/Android): worker
  `football-api-go/cmd/worker` em imagem própria alpine+ffmpeg
  (`ghcr.io/thiagotn/football-manager-media-worker`), Deployment `rachao-worker` no k3s
  (replicas 1, Recreate, limits 1500m/512Mi). Estados: `pending → uploaded → processing →
  ready | failed` (retry ≤3; cleanup de pending >24h).

## Superfícies

| Área | Item |
|---|---|
| Migration | `050_match_videos.sql` — tabela `match_videos` + `players.videos_enabled` |
| API v2 | `POST /matches/{id}/videos` · `POST /matches/{id}/videos/{vid}/confirm` · `GET /matches/public/{hash}/videos` (OptionalAuth; `like_count`/`liked_by_me`) · `DELETE /videos/{vid}` · `POST/DELETE /videos/{vid}/like` · `GET /videos/{vid}/likes` (público) · `GET/PATCH /admin/video-users` · `group_videos_enabled` no payload do match |
| Worker | `cmd/worker` + `internal/worker` (+ `Dockerfile.worker`) |
| Frontend | `/match/[hash]/videos` (feed vertical, upload com progresso XHR, polling de status) · card 🎬 na página da partida · `/admin/videos` (toggle por conta) · namespace `matchVideos` |
| Homelab | `helm/apps/rachao/deployment-worker.yml` + imagem no kustomization (ADR 0007) |

## Pendências manuais (rollout)

1. Migration 050 via psql no `postgres-0` (ns `postgres`) + `INSERT INTO schema_migrations (version) VALUES ('050_match_videos.sql');`
2. **CORS no bucket R2 `rachao-media`** (sem isso todo upload falha no preflight):
   `AllowedOrigins: https://rachao.app, https://www.rachao.app, http://localhost:5173` · `AllowedMethods: PUT` · `AllowedHeaders: content-type`
3. Package GHCR `football-manager-media-worker` público após o primeiro build.
4. Bump das tags no `kustomization.yaml` do homelab (api-go, frontend, media-worker).
5. Smoke: habilitar a flag numa conta de teste, subir vídeo de ~20s (mp4 e .mov/HEVC), conferir `ready` + playback.

## Rollback

- Flag off em `/admin/videos` → feature invisível + upload bloqueado server-side.
- `kubectl -n rachao scale deploy/rachao-worker --replicas=0` → pausa a fila sem afetar a API.

## Evolução futura

- Limite de vídeos por plano (tocar `internal/db/groups.go` + `handlers/subscriptions.go`).
- VAAPI (iGPU) ou Cloudflare Stream/HLS se o volume crescer (escada na ADR 0007).
- Fotos de rachão no mesmo padrão (`matches/{match_id}/`, já previsto na ADR 0006).
