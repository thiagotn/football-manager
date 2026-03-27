# PRD — Avatar de Jogador

**Status:** Rascunho
**Data:** 2026-03-26
**Domínio:** Perfil do jogador

---

## 1. Objetivo

Permitir que jogadores façam upload de uma foto de perfil (avatar) no rachao.app, tornando o perfil mais pessoal e aumentando o engajamento dentro dos grupos. A feature inclui uma imagem padrão gerada automaticamente e uma política de uso aceitável vinculada aos Termos de Uso.

---

## 2. Contexto

Atualmente todo jogador exibe apenas nome e nickname. Grupos com 20+ jogadores têm dificuldade de identificação visual. Um avatar reduz a fricção de reconhecimento, especialmente em listas de presença e sorteio de times.

---

## 3. Avatar padrão (fallback)

Quando o jogador não faz upload, o sistema deve gerar um avatar automaticamente. Existem duas abordagens consagradas:

### Opção A — Iniciais com cor determinística (recomendada)

Gerar um SVG/canvas no frontend com as iniciais do nome + cor derivada do hash do `player_id`.

```
"Zetti Futeboleiro" → "ZF" sobre fundo #3b82f6
```

**Vantagens:**
- Zero custo de storage (gerado em tempo real no cliente)
- Sempre único e consistente para o mesmo jogador
- Sem dependência de serviço externo

**Como implementar:** função `avatarColor(id: string)` que mapeia UUID → cor da paleta do projeto. GitHub, Notion, Linear usam exatamente esse padrão.

### Opção B — Avatares gerados por API (DiceBear)

Usar `https://api.dicebear.com/9.x/initials/svg?seed={player_id}` como `src` do `<img>`.

**Vantagens:** visual mais rico, estilos variados (avataaars, lorelei, pixel-art)
**Desvantagens:** dependência de serviço externo; adiciona latência; pode ser bloqueado por ad-blockers

### Decisão recomendada

**Opção A** para produção. Implementação 100% local, sem custo, sem dependência externa. DiceBear pode ser usado como fallback ou em fase de experimento.

---

## 4. Opções de armazenamento — tradeoffs

### 4.1 Base64 no banco de dados

Serializar a imagem como string Base64 e salvar em coluna `TEXT` ou `BYTEA` no PostgreSQL.

| Aspecto | Avaliação |
|---------|-----------|
| Complexidade de implementação | Muito baixa |
| Custo | Zero (usa banco existente) |
| Performance | **Ruim** — cada SELECT do jogador traz a imagem inteira; queries ficam lentas; backup do banco cresce muito |
| Escalabilidade | **Péssima** — inviável com >1.000 jogadores |
| CDN/cache | Impossível sem cache manual |

**Conclusão: não recomendado.** Antipadrão reconhecido — usar apenas para protótipos descartáveis.

---

### 4.2 Arquivo no VPS (filesystem local)

Salvar o arquivo em `/opt/football-manager/uploads/avatars/{player_id}.webp` e servir via Nginx/Traefik.

| Aspecto | Avaliação |
|---------|-----------|
| Complexidade de implementação | Baixa |
| Custo | Zero (usa VPS existente) |
| Performance | Boa para escala pequena |
| Escalabilidade | Limitada ao disco do VPS; perde arquivos em re-deploy sem volume persistente |
| CDN/cache | Possível com Cloudflare free na frente |
| Backup | Manual; risco de perda em falha de disco |
| Multi-instância | Não funciona com mais de 1 container sem volume compartilhado |

**Conclusão: viável a curto prazo**, mas cria dívida técnica. Requer volume Docker persistente e risco de perda em rebuild.

---

### 4.3 Supabase Storage (recomendado — alinhado à stack atual)

O projeto já usa Supabase como banco de dados em produção. O Supabase Storage é um serviço S3-compatible integrado à mesma conta.

| Aspecto | Avaliação |
|---------|-----------|
| Complexidade de implementação | Baixa — SDK Python (`supabase-py`) ou chamada HTTP direta |
| Custo | Plano free: 1 GB storage + 2 GB egress/mês. Plano Pro: $25/mês com 100 GB |
| Performance | Boa — CDN global da Supabase (Cloudflare por baixo) |
| Escalabilidade | Alta |
| Integração com RLS | Nativa — políticas de acesso por jogador via JWT |
| URL pública | `https://<project>.supabase.co/storage/v1/object/public/avatars/{player_id}.webp` |
| Backup | Gerenciado pela Supabase |

**Conclusão: melhor custo-benefício para o rachao.app.** Usa infraestrutura já existente, sem novo serviço para gerenciar.

---

### 4.4 Cloudflare R2

Object storage S3-compatible da Cloudflare. Egress gratuito (diferencial vs S3).

| Aspecto | Avaliação |
|---------|-----------|
| Custo | $0 egress; $0.015/GB/mês armazenamento; 10 GB free |
| Performance | Excelente — CDN Cloudflare global |
| Complexidade | Média — novo serviço, nova conta, Workers para servir |
| Integração | Via SDK S3-compatible (boto3 / aws-sdk) |

**Conclusão: excelente para escala**, mas adiciona complexidade operacional desnecessária para o estágio atual do produto. Reavalie quando egress do Supabase se tornar gargalo.

---

### 4.5 AWS S3 + CloudFront

Padrão de mercado em grandes plataformas (Instagram, Twitter, LinkedIn).

| Aspecto | Avaliação |
|---------|-----------|
| Custo | ~$0.023/GB storage + egress cobrado ($0.085/GB) |
| Performance | Excelente com CloudFront |
| Complexidade | Alta — IAM, bucket policies, CloudFront distribution, signed URLs |

**Conclusão: over-engineering para o estágio atual.** Adequado somente se a plataforma escalar para centenas de milhares de usuários.

---

## 5. Recomendação de arquitetura

```
Upload flow:
  Frontend → POST /api/v1/players/me/avatar (multipart/form-data)
           → API valida (tamanho, tipo MIME, dimensões)
           → Redimensiona para 256×256 WebP (Pillow)
           → Upload para Supabase Storage (bucket: avatars, path: {player_id}.webp)
           → Salva URL pública em players.avatar_url (TEXT, nullable)
           → Retorna nova URL ao frontend

Serve flow:
  Frontend lê players.avatar_url
  Se null → renderiza avatar de iniciais (SVG gerado no cliente)
  Se preenchido → <img src={player.avatar_url} />
```

### Processamento obrigatório antes do upload

1. **Validar MIME type real** (não confiar no Content-Type do cliente) — usar `python-magic`
2. **Aceitar apenas:** `image/jpeg`, `image/png`, `image/webp`, `image/gif` (gif estático)
3. **Limite de tamanho:** máximo 5 MB no upload → convertido para WebP 256×256px (resultado ~15–30 KB)
4. **Converter sempre para WebP** com Pillow — reduz tamanho em ~70% vs JPEG, suportado por todos os browsers modernos
5. **Rejeitar imagens com metadados suspeitos** (EXIF com GPS, etc.) — strip todos os metadados no processamento

---

## 6. Política de conteúdo (Termos de Uso)

Adicionar seção ao `/terms` e ao modal de upload:

### Adição aos Termos de Uso

> **Imagens de perfil**
>
> Ao fazer upload de uma foto de perfil, você declara que:
>
> - A imagem não contém conteúdo ofensivo, pornográfico, discriminatório, violento ou que viole direitos de terceiros.
> - Você detém os direitos sobre a imagem ou tem permissão para utilizá-la.
> - A imagem representa uma pessoa real ou avatar neutro — não é permitido usar imagens de marcas, times ou celebridades sem autorização.
>
> **Consequências de violação:** o perfil poderá ter a imagem removida administrativamente, ser suspenso temporariamente ou encerrado permanentemente, dependendo da gravidade e reincidência, sem aviso prévio.
>
> O rachao.app reserva-se o direito de remover qualquer imagem que viole estas diretrizes a qualquer momento.

### Aviso no modal de upload (frontend)

```
Ao enviar uma foto, você concorda com nossa política de imagens.
Conteúdo ofensivo pode resultar na desativação do perfil.
```

---

## 7. Moderação

Para o estágio atual (base de usuários pequena), moderação **reativa** é suficiente:

- Botão "Denunciar jogador" (futura feature) → notifica admin por push/email
- Painel admin (`/admin/players`) exibe avatar e tem botão "Remover avatar" (chama `DELETE /api/v1/admin/players/{id}/avatar`)
- Logs de upload com `player_id` + IP para auditoria

**Moderação proativa com IA** (escala futura): Google Cloud Vision Safe Search API ou Amazon Rekognition detectam nudez/violência automaticamente na hora do upload. Custo: ~$1–2 por 1.000 imagens. Avaliar quando base > 10.000 jogadores ativos.

---

## 8. Planos e limites

| Plano | Upload de avatar |
|-------|-----------------|
| Free | Sim (1 avatar) |
| Pro | Sim (1 avatar) |

Avatar não é diferenciador de plano — é feature de retenção básica, deve estar disponível para todos.

---

## 9. Mudanças necessárias

### Backend
- `migrations/033_player_avatar_url.sql` — adiciona `avatar_url TEXT` em `players`
- `app/routers/players.py` — endpoint `PUT /players/me/avatar` (multipart) e `DELETE /players/me/avatar`
- `app/schemas/player.py` — inclui `avatar_url` no `PlayerOut`
- `app/services/storage.py` — wrapper Supabase Storage (upload, delete)
- `app/routers/admin.py` — endpoint `DELETE /admin/players/{id}/avatar`

### Frontend
- Componente `AvatarImage.svelte` — renderiza `<img>` se `avatar_url`, senão SVG de iniciais
- `/profile` — seção de upload com preview, crop opcional (16:9 → 1:1), botão remover
- Exibir avatar em: lista de jogadores do grupo, lista de presença, sorteio de times, ranking de votação

---

## 10. Decisões em aberto

| Questão | Opções | Recomendação |
|---------|--------|--------------|
| Crop no cliente? | Sim (react-easy-crop / vanilla canvas) / Não | Sim — melhora resultado sem custo de servidor |
| Avatar nos times sorteados? | Mostrar / Não mostrar | Fase 2 — avaliar impacto no design do card |
| URL com cache-busting? | `avatar_url?v={timestamp}` | Sim — evitar cache stale após troca de imagem |
| Bucket público ou privado? | Público com URL opaca / Privado com signed URLs | Público com nome de arquivo UUID — simplicidade sem exposição de dados |
