# PRD — Publicação nas Lojas (Google Play & App Store)

**Versão:** 2.2
**Data:** 2026-03-31
**Status:** Fase 1 aprovada | Fase 2 em avaliação

---

## 1. Contexto

O rachao.app é hoje uma PWA (Progressive Web App) completa, construída em SvelteKit 2 + Svelte 5, com:

- Suporte a instalação via "Adicionar à tela inicial" (Android e iOS)
- Push notifications (Web Push / VAPID)
- Offline fallback via Service Worker (Workbox)
- Layout mobile-first responsivo
- Três idiomas (pt-BR, en, es)
- Stripe para pagamentos

Apesar de tecnicamente funcional como PWA, **não estar nas lojas** tem impacto direto em:
- Descoberta orgânica (usuários buscam apps nas lojas, não na web)
- Percepção de credibilidade ("tem na Play Store")
- Compartilhamento entre jogadores ("baixa o app")

---

## 2. Estratégia adotada — Duas fases

### Fase 1 — TWA Android (Google Play) · *aprovada*

Publicar o rachao.app na Google Play como um **TWA (Trusted Web Activity)** — sem Flutter, sem WebView customizado, sem reescrita de código.

**Usuários iOS** continuam sendo orientados a instalar via PWA (Safari → "Adicionar à tela inicial"), fluxo já existente e funcional no app.

### Fase 2 — Flutter nativo (Android + iOS) · *em avaliação*

Reescrever 100% das telas em Flutter (Dart), consumindo a mesma API REST. Publicar nas duas lojas com app verdadeiramente nativo. Avaliação depende do sucesso da Fase 1 e da demanda por features nativas.

---

## 3. Fase 1 — TWA Android

### O que é o TWA

O **Trusted Web Activity** é um mecanismo oficial do Android/Google que executa uma PWA **dentro do Chrome instalado no dispositivo** — não é um WebView, é o Chrome real. O resultado é:

- Performance idêntica ao browser (mesma engine V8, mesmo Service Worker)
- Web Push, offline, cache — tudo funciona exatamente como no browser
- Updates do web app chegam instantaneamente ao app da loja, sem nova submissão
- Google endossa explicitamente esta abordagem para PWAs maduras

O app gerado é um APK/AAB com ~1 MB que simplesmente aponta para a URL da PWA. A ferramenta oficial é o **Bubblewrap CLI** (mantido pela equipe do Chrome/Google).

### Prós

| | |
|---|---|
| ✅ Nenhuma linha de Kotlin, Java ou Flutter necessária | |
| ✅ Performance idêntica ao Chrome — mesma engine, não um WebView | |
| ✅ Service Worker, Web Push e offline funcionam nativamente | |
| ✅ Updates do web refletem imediatamente sem nova versão na loja | |
| ✅ Aprovação na Play Store bem estabelecida (Google recomenda o fluxo) | |
| ✅ Esforço mínimo: ~3,5 dias de trabalho | |
| ✅ Sem novo codebase para manter | |

### Contras / Limitações

| | |
|---|---|
| ❌ Apenas Android — iOS permanece no fluxo PWA | |
| ❌ Requer que o usuário tenha Chrome instalado (padrão em quase todos os Android) | |
| ❌ Não permite features nativas extras (biometria, widget de tela inicial) sem código nativo | |
| ❌ App depende da disponibilidade do servidor — sem conteúdo offline além do Service Worker atual | |

### iOS na Fase 1 — PWA melhorada

Sem app na App Store, a estratégia para iOS é **melhorar a experiência de instalação PWA** já existente:

- O `PwaSmartBanner` e `PwaInstallButton` já estão implementados
- Adicionar uma página ou modal de instrução clara para iOS ("Como instalar: Safari → compartilhar → Adicionar à tela inicial")
- Comunicar ativamente nos canais do app que a instalação iOS é via Safari
- Monitorar demanda: se a base de usuários iOS crescer significativamente, acelera a avaliação da Fase 2

---

## 4. Fase 2 — Flutter nativo (em avaliação)

### O que seria

Reimplementar 100% das telas do rachao.app em Flutter (Dart), consumindo a mesma API REST existente. O resultado seria um app verdadeiramente nativo em Android e iOS.

### Prós

| | |
|---|---|
| ✅ Performance nativa máxima nas duas plataformas | |
| ✅ App Store iOS sem risco de rejeição por "WebView vazio" | |
| ✅ Push notifications via FCM (Android) e APNs (iOS) — confiáveis em ambas as plataformas | |
| ✅ Acesso a APIs nativas: biometria (Face ID/Touch ID), widgets de tela inicial, câmera | |
| ✅ Experiência polida: gestos nativos, animações fluidas, scroll nativo | |
| ✅ Shorebird para OTA updates — patches sem esperar revisão das lojas | |
| ✅ Independente da conectividade para conteúdo já cacheado | |

### Contras / Riscos

| | |
|---|---|
| ❌ **Custo muito alto**: ~20 telas com lógica complexa, equivale a meses de trabalho solo | |
| ❌ **Dois codebases para manter**: qualquer nova feature precisa ser implementada duas vezes (web + Flutter) | |
| ❌ i18n precisa ser reimplementado (3 idiomas, ~1.200 chaves de tradução) | |
| ❌ Stripe no iOS: Apple exige IAP para bens digitais (15–30% de comissão) — mitigação: bloquear compra no app e direcionar ao site ✅ | |
| ❌ Requer domínio de Dart/Flutter além do stack atual (Python/Svelte) | |
| ❌ CI/CD mais complexo: build iOS exige runner macOS no GitHub Actions (~10× custo do Linux) | |
| ❌ Certificados, provisioning profiles, TestFlight, App Store Connect — overhead de processo | |

### Critérios para aprovação da Fase 2

A Fase 2 deve ser iniciada apenas se um ou mais dos seguintes critérios forem atendidos:

- Base de usuários iOS relevante e crescente, sem conversão adequada via PWA
- Demanda por features que a web não suporta (biometria, widget, câmera, offline total)
- Crescimento suficiente do produto que justifique o investimento de manutenção de dois codebases
- Disponibilidade de tempo para o projeto (estimativa: 3–5 meses de trabalho solo)

### Shorebird na Fase 2

Se a Fase 2 for aprovada, o **Shorebird** será a estratégia de entrega de atualizações OTA:

- Permite distribuir correções de código Dart diretamente aos dispositivos **sem passar pela revisão das lojas**
- Funciona substituindo a Dart VM por um runtime próprio que verifica patches em background
- Critério de uso: `shorebird patch` para bug fixes e ajustes de UI; `shorebird release` para mudanças com plugins nativos ou bump de versão visível

| Limitação | Detalhe |
|-----------|---------|
| Apenas código Dart | Mudanças em plugins nativos (Kotlin/Swift) ou novos assets ainda exigem release pela loja |
| Apple — zona cinzenta | Guideline 2.5.2 proíbe download de executável. Shorebird argumenta equivalência ao CodePush (aceito pela Apple). Risco baixo na prática |
| Runtime próprio | Usa fork da Dart VM — pode haver delay na adoção de novas versões do Flutter |

**Pricing Shorebird (referência 2026):**

| Plano | Patches/mês | Preço |
|-------|------------|-------|
| Free | 5.000 patch installs | $0 |
| Team | 50.000 patch installs | ~$20/mês |

---

## 5. Desafios críticos

### 5.1 Digital Asset Links (Fase 1 — obrigatório)

O TWA exige um arquivo de verificação hospedado no servidor para provar que o app pertence ao domínio:

```
https://rachao.app/.well-known/assetlinks.json
```

Sem isso, o TWA exibe uma barra de URL (vira WebView normal) ou é rejeitado.

**Solução no monorepo:** o arquivo fica em `football-frontend/static/.well-known/assetlinks.json`. O SvelteKit serve arquivos de `static/` diretamente na raiz — sem configuração de Nginx/Traefik. Fica versionado no repo e qualquer mudança na keystore é um PR normal.

### 5.2 Qualidade do manifest.webmanifest (Fase 1)

O Bubblewrap lê o manifest da PWA. Para aprovação na Play Store, o manifest precisa ter:
- `name` e `short_name` preenchidos
- `icons` com pelo menos 512×512 (já existe: `/logo-512.png` e `/logo-maskable-512.png`)
- `display: standalone`
- `start_url` definida
- `theme_color` e `background_color`

O manifest atual do rachao.app já atende todos esses requisitos.

### 5.3 Deep Links — App Links Android (Fase 1)

Para que links `https://rachao.app/match/[hash]` compartilhados no WhatsApp abram o app da Play Store (e não o browser), é necessário configurar **Android App Links** — o mesmo arquivo `assetlinks.json` cobre isso.

Sem App Links, o comportamento é: link abre no Chrome, não no app instalado.

### 5.4 Política da Apple — App Store (relevante apenas na Fase 2)

A Apple tem uma política explícita ([Guideline 4.2](https://developer.apple.com/app-store/review/guidelines/#minimum-functionality)) contra apps que são "apenas uma visão de um website". Na Fase 2, com Flutter nativo, este risco não existe — o app será genuinamente nativo.

### 5.5 Pagamentos no iOS (Fase 2 — se aprovada)

A Apple exige IAP para bens digitais com comissão de 15–30%. **Decisão tomada:** bloquear o fluxo de assinatura dentro do app iOS e direcionar ao site (estratégia Spotify/Netflix). Nenhum impacto no backend.

### 5.6 Push Notifications no iOS (Fase 2 — se aprovada)

Web Push não funciona em apps nativos iOS. Seria necessário:
- FCM (Firebase Cloud Messaging) para Android
- APNs (Apple Push Notification service) para iOS
- Nova migration no backend para suportar tokens FCM além de Web Push (VAPID)
- Atualizar `push.py` para rotear por tipo de subscription

**Decisão tomada para Fase 2:** implementar push nativo em versão futura, não no MVP.

---

## 6. Arquitetura do monorepo

### Estrutura atual vs proposta

```
football-manager/                          ← raiz do monorepo
│
├── football-api/                          ← backend (FastAPI)
├── football-frontend/                     ← PWA (SvelteKit)
│   └── static/
│       ├── .well-known/
│       │   └── assetlinks.json  ← NOVO   ← servido em /.well-known/assetlinks.json
│       └── (ícones, manifest, etc.)
├── football-e2e/                          ← testes E2E (Playwright)
├── football-android/            ← NOVO   ← TWA Android (Bubblewrap)
│   ├── twa-manifest.json        ← NOVO   ← config do TWA (commitado, sem credenciais)
│   ├── android/                 ← NOVO   ← projeto Android gerado pelo Bubblewrap
│   │   ├── app/
│   │   │   ├── build.gradle
│   │   │   └── src/
│   │   ├── build.gradle
│   │   ├── gradle/
│   │   ├── gradlew
│   │   └── settings.gradle
│   ├── CLAUDE.md                ← NOVO   ← estado atual do projeto Android
│   └── .gitignore               ← NOVO   ← ignora *.keystore e android/app/build/
│
├── football-flutter/  ← Fase 2, se aprovada
│
├── .github/
│   └── workflows/
│       ├── main.yml                       ← pipeline existente (API + Frontend + E2E)
│       ├── deploy-monitoring.yml          ← existente
│       └── build-twa.yml        ← NOVO   ← build + deploy Play Store
│
├── docs/
│   └── prd/
│       └── publicacao-lojas.md            ← este arquivo
├── CLAUDE.md
└── Makefile
```

### Convenção de nomes

O projeto segue o prefixo `football-*` para todos os sub-projetos:

| Diretório | Tecnologia | Plataforma |
|-----------|-----------|------------|
| `football-api/` | FastAPI | Servidor |
| `football-frontend/` | SvelteKit | Web / PWA |
| `football-e2e/` | Playwright | Testes |
| `football-android/` | Bubblewrap / TWA | Android (Play Store) |
| `football-flutter/` *(Fase 2)* | Flutter | Android + iOS |

### O que é commitado vs ignorado em `football-android/`

```
football-android/
│
├── ✅ twa-manifest.json          ← fonte da verdade da configuração TWA
├── ✅ android/                   ← projeto Gradle gerado pelo Bubblewrap
│   ├── ✅ app/build.gradle
│   ├── ✅ app/src/               ← código gerado (não editar manualmente)
│   ├── ✅ gradlew
│   └── ❌ app/build/             ← saída do build (gitignore)
├── ✅ CLAUDE.md
├── ❌ *.keystore                 ← NUNCA commitar (vai para GitHub Secrets)
└── ❌ android/.gradle/           ← cache do Gradle (gitignore)
```

**`football-android/.gitignore`:**
```gitignore
*.keystore
*.jks
android/.gradle/
android/app/build/
android/local.properties
```

### Relação entre os workflows

```
.github/workflows/

main.yml  (workflow_dispatch)
  ├── detect-changes  (paths: football-api/**, football-frontend/**, football-e2e/**)
  ├── unit-tests      (se api mudou)
  ├── e2e-tests       (se api/frontend/e2e mudou)
  ├── build-images    (docker: api + frontend)
  └── deploy-vps      (docker-compose.prod.yml)

build-twa.yml  (workflow_dispatch, independente)
  └── build           (ubuntu-latest)
      ├── decode keystore (from GitHub Secret)
      ├── bubblewrap build
      ├── upload artifact (AAB)
      └── upload-google-play (track: internal/alpha/beta/production)
```

Os dois pipelines são **completamente independentes**. Um deploy da web não aciona o TWA — o conteúdo do app TWA é a PWA hospedada, então uma atualização do web chega ao app Android automaticamente sem nenhuma ação no pipeline do TWA.

---

## 7. Checklist — Fase 1 (TWA Android)

### Pré-requisitos

- [ ] Conta de desenvolvedor Google Play ($25, pagamento único)
- [ ] Keystore Android para assinatura do app:
  ```bash
  keytool -genkeypair -v -keystore rachao-release.keystore \
    -alias rachao -keyalg RSA -keysize 2048 -validity 10000
  ```
- [ ] Node.js instalado (para Bubblewrap)
- [ ] Java JDK instalado (para geração do APK/AAB)
- [ ] Android SDK instalado (ou Android Studio)

---

### Servidor — Digital Asset Links

- [ ] Obter o SHA-256 fingerprint da keystore de release:
  ```bash
  keytool -list -v -keystore rachao-release.keystore -alias rachao
  ```
- [ ] Criar `football-frontend/static/.well-known/assetlinks.json`:
  ```json
  [{
    "relation": ["delegate_permission/common.handle_all_urls"],
    "target": {
      "namespace": "android_app",
      "package_name": "app.rachao.twa",
      "sha256_cert_fingerprints": ["AA:BB:CC:..."]
    }
  }]
  ```
- [ ] Verificar: `curl https://rachao.app/.well-known/assetlinks.json`
- [ ] Validar com a ferramenta do Google: [Statement List Generator](https://developers.google.com/digital-asset-links/tools/generator)

---

### Bubblewrap — Geração do app

- [ ] Criar o diretório `football-android/` na raiz do monorepo
- [ ] Instalar Bubblewrap CLI:
  ```bash
  npm install -g @bubblewrap/cli
  ```
- [ ] Inicializar o projeto TWA dentro de `football-android/`:
  ```bash
  cd football-android
  bubblewrap init --manifest https://rachao.app/manifest.webmanifest
  ```
  Configurar durante o init:
  - Package name: `app.rachao.twa`
  - App name: `rachao.app`
  - Launch URL: `https://rachao.app/`
  - Apontar para a keystore gerada
- [ ] Criar `football-android/.gitignore` (ver modelo acima)
- [ ] Criar `football-android/CLAUDE.md` com estado inicial do projeto
- [ ] Revisar `twa-manifest.json` gerado (ícones, cores, orientação)
- [ ] Buildar o AAB (formato exigido pela Play Store):
  ```bash
  bubblewrap build
  ```
- [ ] Verificar que o build gerou `app-release-bundle.aab`
- [ ] Commitar `football-android/` (sem o keystore)

---

### Testes

- [ ] Instalar o APK em dispositivo físico Android:
  ```bash
  bubblewrap install
  ```
- [ ] Verificar que a barra de URL do Chrome **não aparece** (confirma que Digital Asset Links está correto)
- [ ] Testar fluxo completo no app:
  - [ ] Login com WhatsApp + OTP
  - [ ] Cadastro de novo usuário
  - [ ] Visualização de partida
  - [ ] Confirmação de presença
  - [ ] Push notification (Web Push deve funcionar normalmente)
  - [ ] Fluxo de assinatura (Stripe)
  - [ ] Troca de idioma
  - [ ] Modo offline (Service Worker)
- [ ] Testar abertura via deep link: enviar `https://rachao.app/match/[hash]` pelo WhatsApp e confirmar que abre o app
- [ ] Testar em pelo menos 2 versões do Android (API 28+ recomendado)

---

### GitHub Actions — Build e Deploy automatizado

O build do TWA roda em `ubuntu-latest` (sem macOS). O `twa-manifest.json` gerado pelo `bubblewrap init` é commitado; a partir daí, `bubblewrap build` é completamente non-interactive.

#### GitHub Secrets necessários

| Secret | Descrição |
|--------|-----------|
| `ANDROID_KEYSTORE_BASE64` | Keystore codificado em base64: `base64 -w 0 rachao-release.keystore` |
| `ANDROID_KEY_ALIAS` | Alias da chave (ex: `rachao`) |
| `ANDROID_KEY_PASSWORD` | Senha da chave |
| `ANDROID_STORE_PASSWORD` | Senha do keystore |
| `GOOGLE_PLAY_SERVICE_ACCOUNT_JSON` | JSON da service account com permissão na Play Console |

> A service account é criada em **Google Play Console → Setup → API access → Create service account** com role "Release Manager".

#### Workflow: `.github/workflows/build-twa.yml`

```yaml
name: Build & Deploy TWA Android

on:
  workflow_dispatch:
    inputs:
      track:
        description: 'Play Store track'
        required: true
        default: 'internal'
        type: choice
        options: [internal, alpha, beta, production]

jobs:
  build:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Set up Java 17
        uses: actions/setup-java@v4
        with:
          java-version: '17'
          distribution: 'temurin'

      - name: Install Bubblewrap CLI
        run: npm install -g @bubblewrap/cli

      - name: Decode keystore
        run: |
          echo "${{ secrets.ANDROID_KEYSTORE_BASE64 }}" \
            | base64 --decode > football-android/rachao-release.keystore

      - name: Inject signing config into twa-manifest.json
        working-directory: football-android
        run: |
          jq \
            --arg path "rachao-release.keystore" \
            --arg alias "${{ secrets.ANDROID_KEY_ALIAS }}" \
            --arg keypass "${{ secrets.ANDROID_KEY_PASSWORD }}" \
            --arg storepass "${{ secrets.ANDROID_STORE_PASSWORD }}" \
            '.signingKey.path = $path |
             .signingKey.alias = $alias |
             .signingKey.keypassword = $keypass |
             .signingKey.storepassword = $storepass' \
            twa-manifest.json > twa-manifest.tmp.json
          mv twa-manifest.tmp.json twa-manifest.json

      - name: Build AAB
        working-directory: football-android
        env:
          ANDROID_HOME: /usr/local/lib/android/sdk
        run: bubblewrap build --skipPwaValidation

      - name: Upload AAB artifact
        uses: actions/upload-artifact@v4
        with:
          name: rachao-twa-${{ github.sha }}
          path: football-android/app-release-bundle.aab
          retention-days: 30

      - name: Upload to Play Store
        uses: r0adkll/upload-google-play@v1
        with:
          serviceAccountJsonPlainText: ${{ secrets.GOOGLE_PLAY_SERVICE_ACCOUNT_JSON }}
          packageName: app.rachao.twa
          releaseFiles: football-android/app-release-bundle.aab
          track: ${{ github.event.inputs.track }}
          status: completed
```

#### Como usar

```bash
# Deploy para track interno
gh workflow run build-twa.yml -f track=internal

# Promover para produção (após validação no internal/alpha/beta)
gh workflow run build-twa.yml -f track=production
```

#### Fluxo de versão

O TWA não tem versionamento próprio de código — o conteúdo é a PWA. A versão do APK é gerenciada no `twa-manifest.json`:

```json
{
  "appVersionCode": 1,
  "appVersionName": "1.0.0"
}
```

A cada novo release, incrementar `appVersionCode` no `twa-manifest.json`, commitar e rodar o workflow.

#### Checklist do workflow

- [ ] Gerar keystore e encodar em base64:
  ```bash
  keytool -genkeypair -v -keystore rachao-release.keystore \
    -alias rachao -keyalg RSA -keysize 2048 -validity 10000
  base64 -w 0 rachao-release.keystore
  ```
- [ ] Adicionar os 5 secrets no repositório GitHub
- [ ] Criar service account no Google Play Console e baixar o JSON
- [ ] Commitar `football-android/twa-manifest.json` (sem credenciais)
- [ ] Rodar o workflow pela primeira vez com `track: internal`
- [ ] Validar AAB em dispositivo físico antes de promover para produção

---

### Google Play Console

- [ ] Criar conta na [Google Play Console](https://play.google.com/console)
- [ ] Criar novo app → tipo: "App" → gratuito
- [ ] Configurar ficha da loja:
  - [ ] Título: `rachao.app` (30 caracteres)
  - [ ] Descrição curta (80 caracteres) em pt-BR, en, es
  - [ ] Descrição completa (4.000 caracteres) em pt-BR, en, es
  - [ ] Ícone de alta resolução (512×512 PNG, sem transparência)
  - [ ] Feature graphic (1024×500 PNG)
  - [ ] Screenshots (mínimo 2, recomendado 4–8): celular, tablet opcional
  - [ ] Categoria: Esportes
  - [ ] Classificação de conteúdo: responder questionário (deve resultar em "Livre")
  - [ ] Política de privacidade: `https://rachao.app/privacy`
- [ ] Configurar preço: Gratuito
- [ ] Configurar países de distribuição (Brasil inicialmente)
- [ ] Upload do AAB em track interno via workflow
- [ ] Adicionar e-mails de testadores internos
- [ ] Testar via track interno no dispositivo físico
- [ ] Promover para produção
- [ ] Aguardar revisão (geralmente 1–3 dias úteis)

---

### Pós-publicação Fase 1

- [ ] Monitorar Android Vitals (crashes, ANRs) na Play Console
- [ ] Configurar resposta a avaliações na loja
- [ ] Adicionar link "Disponível no Google Play" no site (`/lp`, footer)
- [ ] Atualizar banners de instalação no app para detectar se veio da Play Store
- [ ] Documentar o processo de nova versão: incrementar `appVersionCode` + rodar workflow

---

## 8. Decisões consolidadas

| # | Questão | Decisão |
|---|---------|---------|
| 1 | **Abordagem Fase 1** | ✅ TWA Android via Bubblewrap — sem Flutter |
| 2 | **iOS Fase 1** | ✅ PWA existente — orientar instalação via Safari |
| 3 | **Abordagem Fase 2** | ⏳ Em avaliação — Flutter nativo 100% se critérios forem atingidos |
| 4 | **Push iOS** | ⏸️ Fase 2, versão futura |
| 5 | **Pagamentos iOS** | ✅ Bloquear no app, direcionar ao site (estratégia Spotify/Netflix) |
| 6 | **OTA Updates (Fase 2)** | ✅ Shorebird |
| 7 | **Build iOS (Fase 2)** | ✅ GitHub Actions `macos-latest` — sem Mac local |
| 8 | **Prazo** | ✅ Sem prazo fixo — projeto solo |

---

## 9. Estimativa de esforço

### Fase 1 — TWA Android

| Tarefa | Esforço |
|--------|---------|
| Digital Asset Links no frontend (`static/.well-known/`) | 0,5 dia |
| Bubblewrap setup + build + testes locais | 1 dia |
| GitHub Actions workflow + secrets + service account | 0,5 dia |
| Ficha da loja (textos, screenshots, ícones) | 1 dia |
| Submissão + acompanhamento da revisão | 0,5 dia |
| **Total de trabalho** | **~3,5 dias** |
| Revisão Google Play | 1–3 dias úteis |

### Fase 2 — Flutter nativo (estimativa preliminar)

| Tarefa | Esforço |
|--------|---------|
| Setup projeto Flutter + CI/CD (GitHub Actions macOS + Shorebird) | 2–3 dias |
| Reescrita das ~20 telas principais | 8–12 semanas |
| i18n (3 idiomas, ~1.200 chaves) | 1 semana |
| Integração push nativo (FCM + APNs) + backend | 1 semana |
| Testes + ajustes pré-submissão | 2–3 semanas |
| Fichas nas lojas (Play + App Store) | 2 dias |
| Revisão App Store (TestFlight → produção) | 1–3 semanas |
| **Total estimado** | **~4–5 meses solo** |

---

## 10. Riscos

| Risco | Fase | Prob. | Impacto | Mitigação |
|-------|------|-------|---------|-----------|
| Digital Asset Links mal configurado (barra URL aparece) | 1 | Média | Médio | Validar com ferramenta Google antes de submeter |
| Play Store rejeitar por política de conteúdo | 1 | Baixa | Médio | Preencher corretamente classificação de conteúdo e política de privacidade |
| Deep links não funcionam após publicação | 1 | Baixa | Médio | Testar App Links antes de publicar em produção |
| iOS base cresce e PWA install não converte | 1 | Média | Médio | Acelera avaliação da Fase 2 |
| App Store rejeitar por falta de valor nativo | 2 | Baixa | Alto | App Flutter nativo não tem este problema — telas 100% reescritas |
| Dois codebases dificulta manutenção solo | 2 | Alta | Alto | Critério de entrada: só avançar com base de usuários e demanda que justifique |
| Shorebird rejeitado pela Apple (guideline 2.5.2) | 2 | Baixa | Alto | Amplamente adotado no mercado; monitorar política |
| Custo runner macOS no GitHub Actions | 2 | Baixa | Baixo | Pipeline manual (`workflow_dispatch`) minimiza execuções |
