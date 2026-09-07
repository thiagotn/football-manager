# Latência do rachao.app no Brasil — roteamento da Cloudflare

- **Status:** diagnóstico concluído, **nenhuma ação implementada** (documento de referência)
- **Data do diagnóstico:** 2026-08-21 (sessão "o rachao.app parece lento para carregar")
- **Revalidação:** 2026-09-07
- **Relacionado:** ADR 0001 do repo `homelab` (Cloudflare Tunnel), ADR 0006 (mídia no R2 / `cdn.rachao.app`)

---

## 1. Resumo

O rachao.app às vezes demora ~1 s para começar a responder a partir do Brasil, mesmo com a
aplicação respondendo em ~30 ms. A causa **não está no app, no banco nem no homelab**: está no
**ponto de presença (colo) da Cloudflare que atende o usuário**. Quando o ISP do usuário é roteado
para um colo nos EUA (Miami/Virgínia) em vez de São Paulo (GRU), cada requisição atravessa o
hemisfério duas vezes antes de chegar ao túnel, que fica em GRU. O problema é intermitente porque
depende do roteamento da Cloudflare para o ISP naquele momento, não de nada sob nosso controle.

---

## 2. Como o tráfego chega ao app (contexto)

Desde 2026-07 (ADR 0001 do homelab) a origem não tem porta inbound. Todo o tráfego público passa
por um **Cloudflare Tunnel**:

```
Usuário (BR) → DNS anycast → colo Cloudflare que atende o ISP do usuário
            → rede interna da Cloudflare → colo onde o túnel está registrado (GRU)
            → cloudflared (pod, 2 réplicas) → Traefik → rachao-frontend / rachao-api
```

Pontos que importam para latência:

- O `cloudflared` registra 4 conexões por réplica, todas em **gru02/gru05/gru08/gru17 (São Paulo)**.
  O túnel em si está no lugar certo.
- O TLS termina na borda da Cloudflare. O caminho borda → cloudflared é interno à Cloudflare.
- `rachao.app` (HTML SSR) e `api.rachao.app` (API v2) são **hosts distintos**. O navegador abre uma
  conexão TLS para cada um. Chamadas com `Authorization` para `api.rachao.app` exigem **preflight
  CORS** (OPTIONS) na primeira vez, cacheado por 24 h (`Access-Control-Max-Age: 86400`).
- `cdn.rachao.app` (R2) **não passa pelo túnel**: é servido direto da borda, com cache imutável.
- Assets `/_app/immutable/*` do frontend têm `Cache-Control: public, max-age=31536000, immutable`
  e são cacheados na borda pela Cloudflare (`cf-cache-status: HIT` após o primeiro acesso naquele
  colo). O HTML é sempre `DYNAMIC` (passa pelo túnel toda vez).

---

## 3. Evidências coletadas

### 3.1 Medições de 2026-08-21 (dia do sintoma)

Mesma máquina, mesmo IP (Claro/NET, `201.83.245.5`), mesmo horário:

| Camada | Medição | Veredito |
|---|---|---|
| Aplicação direto no Traefik (LAN, `--resolve` para 192.168.0.5) | HTML SSR **27 ms**; `/api/v2/health` **30 ms** | rápida |
| Node do homelab | CPU 28 %, RAM 30 %, sem restarts, worker/API ociosos | saudável |
| Túnel cloudflared | conexões registradas em gru02/05/08/17 | ok |
| Via Cloudflare, do Brasil | TTFB **0,87–1,07 s** em `rachao.app` e `api.rachao.app`; `cdn.rachao.app` 0,63–0,73 s | lento |
| Colo que atendia o cliente (`/cdn-cgi/trace`) | **`colo=MIA`** (Miami) | causa |
| Colo que atendia o próprio homelab saindo para a internet | **`colo=IAD`** (Virgínia) | mesma causa |
| Outro site no mesmo túnel (`dratatimayumi.com.br`) | TTFB 0,95 s | não é específico do rachao.app |
| Ping para a borda da Cloudflare | 6 ms | a conexão até a borda é boa |

Decomposição do TTFB de ~1 s pelo `curl -w`: `conn` ~0,13–0,31 s, `tls` ~0,38–0,52 s, `ttfb`
~1,0 s. Ou seja, cada round-trip (TCP, TLS, request) custava ~130–150 ms, coerente com ida e volta
Brasil ↔ Miami, e não com os 6 ms de ping local. O handshake TLS até a borda deveria ser local; o
fato de o TLS custar centenas de ms mostra que o "colo" era Miami, não uma borda em São Paulo.

Detalhe que isola a causa: **o servidor respondia em 27 ms** quando acessado sem a Cloudflare.
Toda a diferença (~950 ms) estava entre o cliente e o cloudflared.

### 3.2 Revalidação em 2026-09-07

Mesma máquina, mesmo IP:

| Item | Valor |
|---|---|
| `colo` | **GRU** |
| TTFB `rachao.app` | 0,086–0,099 s |
| TTFB `api.rachao.app/api/v2/health` | 0,10 s |
| Asset imutável | 1ª vez `MISS` (0,06 s), 2ª vez `HIT` |

Com o cliente atendido por GRU o app abre em ~100 ms. Isso confirma que o sintoma **é o roteamento
do ISP para um colo nos EUA**, e que ele varia ao longo do tempo. Um mesmo usuário pode ver o app
rápido num dia e lento em outro sem nenhum deploy no meio.

---

## 4. Por que a Cloudflare faz isso

- A rede da Cloudflare é **anycast**: o usuário chega ao colo que o roteamento BGP do ISP dele
  escolhe, não necessariamente o mais próximo.
- No **plano Free** a Cloudflare não garante atendimento no colo mais próximo. Para alguns ISPs
  brasileiros (historicamente Claro/NET e outros grandes), por custo de peering/trânsito no Brasil,
  o tráfego de zonas Free é frequentemente roteado para colos nos EUA (MIA, IAD, ATL). Zonas Pro e
  superiores tendem a ser servidas em GRU. Isso é um comportamento comercial da Cloudflare, não
  configurável na zona.
- O túnel só piora a geometria: no proxy tradicional o colo dos EUA falaria com a origem no Brasil
  por um único trecho de retorno. Com o túnel, o colo de entrada (MIA) precisa alcançar o colo onde
  o `cloudflared` está registrado (GRU) pela rede interna da Cloudflare, e depois o caminho volta
  por MIA. O trecho MIA ↔ GRU é pago duas vezes por request.
- A latência se multiplica por round-trip: TCP + TLS + request na primeira conexão, mais o preflight
  CORS na primeira chamada autenticada à API, mais cada chamada de API subsequente.

---

## 5. Impacto no app (onde a latência aparece)

O custo é **por round-trip**, então o que sofre é o primeiro carregamento e as páginas com muitas
chamadas sequenciais:

1. **Primeiro acesso** (`rachao.app`): DNS → TCP → TLS → HTML SSR. Com colo nos EUA, ~1 s antes do
   primeiro byte. Assets JS/CSS vêm da borda (cache HIT) e não pagam o túnel.
2. **Boot da SPA**: o store de auth chama `GET /auth/me` antes de liberar a tela. Nova conexão TLS
   para `api.rachao.app` + preflight CORS + request. Com colo nos EUA, mais ~1–1,5 s.
3. **Home autenticada** (`/`): `groups.list`, `myStats` (e, para admin, `players.list`,
   `signupStats`) em paralelo; depois `matches.list` **por grupo** em paralelo; depois `discover`.
   São pelo menos 3 "ondas" sequenciais de requests. Cada onda paga um round-trip completo.
4. **Navegação subsequente**: SPA, sem novo HTML. Só as chamadas de API pagam o overhead. O Service
   Worker faz `NetworkOnly` para API e só precache dos assets do build.
5. **Mídia** (`cdn.rachao.app`): avatares, fotos e vídeos vêm do R2 direto da borda. Pagam a
   latência até o colo dos EUA, mas não o trecho até GRU nem o túnel.

Fora do Brasil (ou com ISP roteado para GRU), nada disso é perceptível.

---

## 6. Caminhos possíveis

Ordenados do menor para o maior esforço/custo. Nenhum foi implementado.

### 6.1 Diagnosticar antes de gastar (custo zero)

- Abrir `https://rachao.app/cdn-cgi/trace` e olhar `colo=`. `GRU` = ok; `MIA`/`IAD`/`ATL` =
  sintoma presente.
- Comparar Wi-Fi (Claro/NET) com 4G de outra operadora. Se o 4G cair em GRU e voar, confirma que
  é roteamento por ISP.
- Registrar uma amostra ao longo de alguns dias (qual ISP, qual colo, qual TTFB) antes de decidir
  pagar algo. Uma sonda simples no Uptime Kuma medindo TTFB de `rachao.app` já mostra a variação.

### 6.2 Reduzir round-trips no app (sem mexer na infra)

Não resolve o roteamento, mas diminui quanto ele dói. Ganho proporcional ao número de ondas
sequenciais eliminadas:

- **Endpoint agregado de dashboard** (`GET /api/v2/dashboard`): devolver grupos + próximos rachões
  + stats numa chamada. Hoje a home faz 3 ondas sequenciais e N `matches.list` (uma por grupo).
- **Servir a API pelo mesmo host** (`rachao.app/api/*` via Traefik em vez de `api.rachao.app`):
  elimina uma conexão TLS extra e o preflight CORS inteiro. O Service Worker e o `api.ts` já
  suportam mesma origem (é o modo do dev). Exige mudar `VITE_API_URL` no build de prod, o Ingress e
  o CORS. Alternativa menor: manter `api.rachao.app` mas adicionar `<link rel="preconnect">` no HTML
  para abrir a conexão TLS da API em paralelo com o download dos assets.
- **Cache do `auth.me` no boot**: já existe `player` em localStorage. Renderizar com o cache e só
  bloquear se o token estiver expirado (hoje sempre aguarda a resposta da API).
- **Cache de leitura no Service Worker** para GETs da API com estratégia stale-while-revalidate
  (grupos, lista de partidas): a tela aparece com o dado anterior enquanto a rede atualiza.
  Hoje é `NetworkOnly`, decisão tomada por causa de um bug de CORS cross-origin; se a API for
  mesma origem, essa estratégia vira viável.
- **Cache na borda para GETs públicos** (`/matches/public/{hash}`, `/matches/discover`,
  páginas `/lp`, `/faq`, `/plans`): Cache Rules na Cloudflare com TTL curto (30–60 s). O link de
  partida compartilhado no WhatsApp abriria da borda sem tocar o túnel.

### 6.3 Pagar à Cloudflare para melhorar o roteamento

- **Cloudflare Pro** (~US$ 25/mês por zona): na prática, zonas Pro são atendidas nos colos
  brasileiros para a maioria dos ISPs. Não é garantia contratual, mas é o efeito mais relatado.
  Só a zona `rachao.app` precisa do upgrade; os outros domínios do túnel continuam Free.
- **Argo Smart Routing** (US$ 5/mês + US$ 0,10/GB): otimiza o caminho **dentro** da rede da
  Cloudflare (colo de entrada → colo do túnel). Não muda o colo de entrada; ajuda no trecho
  MIA ↔ GRU mas não elimina a travessia. Menos indicado que Pro para este caso.
- **Cloudflare Business/Enterprise**: garante roteamento, mas o custo não faz sentido para o
  estágio atual.

### 6.4 Mudar onde o tráfego entra (alterações de arquitetura)

- **Frontend estático fora do túnel**: publicar o build do SvelteKit em Cloudflare Pages (ou
  similar) e deixar só a API atrás do túnel. O HTML e os assets passariam a vir da borda; o túnel
  seria pago só nas chamadas de API. Perde-se o SSR das OG tags de `/match/[hash]` (usadas pelo
  preview do WhatsApp), que teria que ser refeito como Worker/função de edge, ou manter só essa rota
  no túnel.
- **Origem no Brasil sem túnel** (VPS/cloud em São Paulo, ou expor o homelab): resolve a
  latência, mas expor o homelab **contradiz a ADR 0001** (esconder IP residencial, fechar spoof do
  `X-Forwarded-For`). Mover a origem para uma VPS em SP resolve sem quebrar a ADR, mas reverte a
  decisão de rodar no homelab (ADR 0002) e reintroduz custo mensal.
- **Trocar a Cloudflare por outro proxy/CDN com PoP em SP no plano gratuito** (por exemplo,
  Bunny, Fastly, ou túnel via Tailscale Funnel/ngrok pago): mais trabalho de migração (DNS-01 do
  cert-manager, R2, Cache Rules) por um ganho que o plano Pro provavelmente entrega sozinho.

### 6.5 Conviver

O custo está concentrado no primeiro carregamento. Depois disso o app navega como SPA e os assets
ficam em cache. Para o público atual (grupos de amigos, uso esporádico) pode ser aceitável enquanto
não houver reclamação recorrente.

---

## 7. Recomendação registrada em 2026-08-21

1. Confirmar o padrão por ISP/colo antes de gastar (6.1).
2. Se confirmado e incômodo: testar **Cloudflare Pro** na zona `rachao.app` por um mês e medir de
   novo. É a alavanca que ataca a causa raiz (colo de entrada) com menor esforço.
3. Independente disso, as reduções de round-trip do item 6.2 (endpoint agregado de dashboard,
   API na mesma origem ou `preconnect`, cache na borda para GETs públicos) melhoram o app para
   todo mundo e valem como backlog.
4. Não expor a origem sem túnel.

---

## 8. Como reproduzir o diagnóstico

```bash
# 1. Colo que atende esta máquina
curl -s https://rachao.app/cdn-cgi/trace | grep -E "colo|ip="

# 2. Latência via Cloudflare (decomposta)
for i in 1 2 3; do
  curl -s -o /dev/null -w "conn=%{time_connect}s tls=%{time_appconnect}s ttfb=%{time_starttransfer}s\n" https://rachao.app
done

# 3. Latência da aplicação sem Cloudflare (só funciona na LAN do homelab)
curl -sk --resolve rachao.app:443:192.168.0.5 -o /dev/null -w "ttfb=%{time_starttransfer}s\n" https://rachao.app

# 4. Colos onde o túnel está registrado
ssh 192.168.0.5 "kubectl -n cloudflared logs -l app=cloudflared --tail=300 | grep 'Registered tunnel' | tail -8"

# 5. Cache de assets na borda (esperado: HIT a partir da 2ª requisição)
asset=$(curl -s https://rachao.app | grep -oE '/_app/immutable/[^"]+\.js' | head -1)
curl -sI "https://rachao.app$asset" | grep -i cf-cache-status
```

Leitura: se o passo 3 devolve dezenas de ms e o passo 2 devolve centenas de ms com `colo` fora do
Brasil no passo 1, o problema é o roteamento da Cloudflare, não o app.
