# PRD 038 — Mapa de Locais de Futebol em /discover

**Número**: 038  
**Versão**: 1.1  
**Status**: ❌ Cancelado  
**Data**: Abril de 2026

---

## Problema

A tela `/discover` exibe partidas abertas em formato de lista. O usuário não tem percepção espacial de onde essas partidas acontecem, nem consegue descobrir novos locais de futebol próximos a ele. Um mapa tornaria a descoberta mais intuitiva e revelaria o potencial de crescimento da plataforma ao mostrar quadras e campos que ainda não usam o rachao.app.

---

## Solução

Adicionar um mapa interativo na tela `/discover`, exibindo dois tipos de marcadores:

- **Pin padrão** (neutro): quadras e campos de futebol genéricos obtidos via API pública de mapas — locais que existem na realidade mas não têm partidas no rachao.app
- **Pin rachao.app** (destaque): locais onde há partidas abertas cadastradas no app — marcador verde com ícone de bola de futebol, popup com detalhes da partida e botão de ação

---

## Comparativo: Google Maps vs Alternativas

### Google Maps / Google Places API

**Limitações:**

| Item | Detalhe |
|------|---------|
| **Custo** | Places Nearby Search: ~R$ 0,18/req (USD 0,032). Maps JS: USD 7/1.000 loads após 28k gratuitos/mês. Inviável em produção sem billing robusto. |
| **Termos de uso** | Proíbe cachear resultados do Places API. Proíbe exibir dados do Google em mapas não-Google. Dados não podem ser exportados ou usados fora do contexto do mapa. |
| **Privacidade** | Envia localização do usuário para servidores do Google. |
| **Vendor lock-in** | Migração posterior é cara e trabalhosa. |
| **Bundle size** | SDK proprietário pesado (~500 KB). |

### Alternativa adotada: Leaflet + OpenStreetMap + Overpass API

| Componente | Papel | Custo |
|-----------|-------|-------|
| **Leaflet** | Biblioteca de mapas — leve (~40 KB gzip), open source, MIT. Já era dependência do projeto. | Gratuito |
| **OpenStreetMap tiles** | Tiles do mapa base via `tile.openstreetmap.org`. | Gratuito (uso moderado) |
| **Overpass API** | API pública para consultar dados OSM. Busca campos/quadras de futebol por raio geográfico. Tags: `sport=soccer`, `leisure=pitch`. Sem chave de API. | Gratuito |
| **Nominatim** | Geocodificação de endereço → lat/lng (OSM). Gratuito para uso moderado (1 req/s). | Gratuito |

---

## Pinos no mapa

| Tipo | Origem dos dados | Visual |
|------|-----------------|--------|
| **Pin genérico** | Overpass API / OSM | Círculo cinza escuro (#475569), ícone ⚽, tamanho 22px |
| **Pin rachao.app** | `GET /matches/discover` | Círculo verde (#16a34a), ícone ⚽, tamanho 32px, popup rico |

O pin rachao.app sobrepõe visualmente o pin genérico quando existirem no mesmo local, indicando que aquele campo já usa a plataforma.

---

## Escopo técnico — v1

### Backend

**Migration `040_match_coordinates.sql`**
```sql
ALTER TABLE matches
  ADD COLUMN IF NOT EXISTS latitude  NUMERIC(10, 7) DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS longitude NUMERIC(10, 7) DEFAULT NULL;
```

**`app/models/match.py`**: campos `latitude` e `longitude` adicionados ao modelo `Match`

**`app/schemas/match.py`**: `latitude`/`longitude` em `MatchCreate`, `MatchUpdate`, `MatchResponse` e `DiscoverMatchResponse`

**`app/api/v1/routers/matches.py`**: `DiscoverMatchResponse` inclui `latitude` e `longitude` com conversão explícita de `Decimal` → `float`

### Frontend

**`src/lib/overpass.ts`** (novo):
- `fetchFootballVenues(lat, lng, radiusMeters): Promise<Venue[]>` — consulta Overpass API, filtra por `sport=soccer` e `leisure=pitch`, cacheia em memória por sessão
- `geocodeAddress(address): Promise<{lat, lng} | null>` — Nominatim geocoding

**`src/routes/discover/+page.svelte`**:
- Toggle "Lista / Mapa" no topo
- Mapa carregado lazy (importação dinâmica de Leaflet)
- Ao ativar: solicita geolocalização (fallback: centro do Brasil)
- Busca venues OSM via Overpass num raio de 5km
- Exibe pins rachao.app (verde) para cada `DiscoverMatch` com `latitude`/`longitude`
- Popup do pin rachao.app: grupo, data, vagas, botão "Quero jogar"
- Botão "📍" para re-centrar no usuário
- Atribuição OSM no rodapé

**`src/routes/groups/[id]/+page.svelte`**:
- Botão "Localizar no mapa" após campo de endereço (create e edit match)
- Dispara geocodificação via Nominatim → preenche `latitude`/`longitude` automaticamente
- Confirma com toast e exibe coordenadas inline
- Coordenadas enviadas ao criar/editar via `matchesApi.create`/`matchesApi.update`

### Nota sobre coordenadas

Na v1, as coordenadas são **opcionais**. O organizador pode informar ao criar/editar a partida usando o botão "Localizar no mapa". Se não preenchidas, a partida aparece na lista mas não no mapa.

---

## Arquivos criados/modificados

| Arquivo | Ação |
|---------|------|
| `football-api/migrations/040_match_coordinates.sql` | Criado |
| `football-api/app/models/match.py` | Modificado |
| `football-api/app/schemas/match.py` | Modificado |
| `football-api/app/api/v1/routers/matches.py` | Modificado |
| `football-api/CLAUDE.md` | Atualizado (próxima migration → 041) |
| `football-frontend/src/lib/overpass.ts` | Criado |
| `football-frontend/src/lib/api.ts` | Modificado (tipos Match e DiscoverMatch) |
| `football-frontend/src/routes/discover/+page.svelte` | Modificado (toggle + mapa) |
| `football-frontend/src/routes/groups/[id]/+page.svelte` | Modificado (geocodificação) |
| `football-frontend/messages/pt-BR.json` | Modificado |
| `football-frontend/messages/en.json` | Modificado |
| `football-frontend/messages/es.json` | Modificado |
| `docs/prd/038-mapa-discover.md` | Criado |
| `docs/prd/INDEX.md` | Atualizado |

---

## Fora de escopo (v1)

- Filtro por raio configurável pelo usuário
- Clustering de pins (muitos pins no mesmo lugar)
- Busca por cidade/bairro no mapa
- Tiles pagos (Mapbox, Google)
- Geocodificação automática em background de partidas antigas
- Grupos (sem partidas ativas) no mapa

---

## Verificação

1. Sem geolocalização → mapa centraliza no Brasil, não quebra
2. Com geolocalização → pins genéricos OSM carregam num raio de 5km
3. Partida com lat/lng → pin rachao.app verde aparece com destaque
4. Partida sem lat/lng → não aparece no mapa (só na lista)
5. Overpass API indisponível → sem crash; lista continua funcionando
6. Toggle Lista/Mapa → alterna view sem recarregar dados
7. Popup do pin rachao.app → botão "Quero jogar" funcional
8. Botão "Localizar no mapa" no form → coordenadas preenchidas e salvas

---

## Decisão — Abril 2026

**Funcionalidade cancelada.**

O frontend foi revertido ao estado anterior: `/discover` exibe somente lista, sem toggle Lista/Mapa; formulários de partida sem botão de geocodificação.

O backend também foi revertido integralmente: migration `040_match_coordinates.sql` removida, campos `latitude`/`longitude` não existem no model, schema ou router.

Se a funcionalidade for retomada no futuro, todos os itens da seção "Arquivos criados/modificados" precisarão ser implementados do zero.
