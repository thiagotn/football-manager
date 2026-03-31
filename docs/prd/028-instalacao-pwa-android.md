# PRD — Revisão do Fluxo de Instalação PWA no Android

## 1. Contexto

O rachao.app é uma PWA (Progressive Web App) distribuída sem loja de aplicativos. No Android, o navegador Chrome exibe nativamente um popup de instalação ("Adicionar à tela inicial") quando determinados critérios técnicos são atendidos. O app já implementa o mecanismo padrão: captura o evento `beforeinstallprompt`, armazena o prompt diferido e o aciona quando o usuário clica em "Instalar App" no menu lateral.

O problema: esse fluxo **só funciona quando o próprio navegador decide disparar o evento** — o que depende de heurísticas internas do Chrome (frequência de visitas, engajamento, tempo na página) e não é garantido para todos os usuários. Na prática, muitos usuários nunca veem o botão de instalação e acabam usando o app no modo web responsivo, sem a experiência nativa.

---

## 2. Problema

### Dor atual
- O evento `beforeinstallprompt` **não é disparado de forma consistente** em todos os cenários Android/Chrome
- Navegadores alternativos (Samsung Internet, Firefox, Edge Android) têm comportamentos distintos — alguns não suportam o evento, outros têm critérios diferentes
- Usuários que chegam ao app por link direto (WhatsApp, por exemplo) podem estar em um **webview interno** do WhatsApp/Telegram, onde o evento nunca é disparado
- Sem o popup, o usuário não sabe que pode instalar o app — não há indicação visual de que a instalação é possível
- Resultado: parte relevante da base usa o app no modo browser, sem ícone na home screen, sem comportamento standalone, sem notificações push confiáveis

### Impacto
- Experiência degradada: barra do navegador visível, navegação diferente do app nativo
- Retenção menor: apps instalados têm maior taxa de retorno que bookmarks
- Notificações push menos confiáveis em modo browser vs standalone

---

## 3. Possibilidades

### Opção A — Instrução manual com deep link para o Chrome (recomendado como base)

**Como funciona:** Quando o evento `beforeinstallprompt` não está disponível (navegador incompatível, webview, limite de tentativas atingido), exibir um banner/modal com instruções para o usuário:
1. Abrir o link no **Chrome** (se não estiver)
2. Tocar nos três pontos (menu) → "Adicionar à tela inicial"

**Vantagens:**
- Funciona em 100% dos cenários Android — sempre há uma saída
- Zero dependência de evento do navegador
- Pode ser mostrado proativamente (ex: na primeira visita ou após 2 sessões)

**Desvantagens:**
- Mais fricção do que o popup nativo — exige ação manual do usuário
- Instruções visuais precisam ser claras (screenshots/GIF)

**Implementação:** Detectar se está em webview (via `user-agent`) ou se `beforeinstallprompt` nunca disparou após N sessões → exibir banner "Instale o app" com instruções passo a passo.

---

### Opção B — Detecção de webview + redirect para Chrome

**Como funciona:** Ao detectar que o usuário está em um webview (WhatsApp, Instagram, Telegram), exibir um banner pedindo para **abrir no Chrome**, com um botão que copia o link e instrui o usuário a abrir no navegador nativo.

**Detecção de webview Android:**
```
user-agent contém: "wv" (WebView oficial)
ou ausência de "Chrome/" no UA com presença de "Android"
ou "Instagram", "FBAN", "FB_IAB", "Twitter"
```

**Vantagens:**
- Resolve o caso mais comum (chegada via link de WhatsApp)
- Proativo — aparece antes mesmo do usuário tentar instalar

**Desvantagens:**
- Detecção de webview por UA é imperfeita (pode ter falsos positivos)
- O usuário precisa copiar o link e abrir manualmente — fricção alta

---

### Opção C — Página de download dedicada (`/download` ou `/instalar`)

**Como funciona:** Uma landing page específica para instalação, linkada nos pontos estratégicos do app (primeiro acesso, menu, comunicações via WhatsApp). A página detecta o dispositivo e exibe instruções personalizadas:

| Dispositivo | Instrução |
|-------------|-----------|
| Android + Chrome + `beforeinstallprompt` disponível | Botão "Instalar agora" (aciona o prompt nativo) |
| Android + Chrome + sem prompt | Instruções: menu ⋮ → "Adicionar à tela inicial" com GIF |
| Android + Samsung Internet | Instruções específicas para Samsung Internet |
| Android + webview | Instrução para abrir no Chrome com link copiável |
| iOS + Safari | Instruções existentes: Compartilhar → Adicionar à Tela de Início |
| iOS + outro browser | Instrução para abrir no Safari |
| Desktop | Instrução para usar Chrome/Edge com suporte a PWA |

**Vantagens:**
- URL compartilhável — pode ser enviada no WhatsApp do grupo para novos membros
- Centraliza toda a lógica de instalação em um lugar
- Pode ser enriquecida com screenshots/GIFs sem poluir o app principal

**Desvantagens:**
- Requer criação de uma nova rota e manutenção do conteúdo
- Usuário precisa navegar até lá (a não ser que seja exibida automaticamente)

---

### Opção D — Banner persistente "Instale o app" na dashboard

**Como funciona:** Exibir um banner discreto no topo da dashboard (ou como card fixo) para usuários que ainda não instalaram o app, direcionando para o fluxo correto. O banner some após a instalação ou após o usuário fechar manualmente (persistido em localStorage).

```
┌─────────────────────────────────────────────────────┐
│ 📲 Instale o app para uma experiência melhor    [X] │
│    Acesso rápido, funciona offline, notificações     │
│    [Instalar agora]                                  │
└─────────────────────────────────────────────────────┘
```

**Vantagens:**
- Visibilidade garantida — aparece no fluxo principal, não depende de o usuário encontrar o menu
- Pode acionar o prompt nativo quando disponível, e cair para instruções quando não

**Desvantagens:**
- Intrusivo se não for bem dosado (mostrar apenas nas primeiras N sessões)
- Precisa de lógica de dismissal e persistência

---

## 4. Recomendação

A abordagem mais eficaz é **combinar as opções A + D + C**:

1. **Opção D (banner na dashboard):** Primeira barreira — visibilidade garantida para todos os usuários não instalados. Quando clicado, aciona o prompt nativo se disponível, ou abre instruções contextuais.

2. **Opção A (instruções manuais):** Fallback universal quando o prompt nativo não está disponível. Instrução clara com o caminho menu ⋮ → "Adicionar à tela inicial".

3. **Opção B (detecção de webview):** Caso especial para chegada via WhatsApp/Instagram — banner imediato pedindo para abrir no Chrome antes de qualquer outra coisa.

4. **Opção C (página `/instalar`):** URL compartilhável para ser enviada no WhatsApp de grupos, facilitando a adoção em massa de novos membros.

A **Opção C sozinha** é o menor esforço com maior alcance: uma página simples, linkável, que pode ser enviada pelo admin no grupo do WhatsApp: `"Baixe o app: rachao.app/instalar"`.

---

## 5. Detalhes Técnicos Relevantes

### Detecção de contexto atual (`pwaInstall.ts`)
O store já detecta:
- `canInstall`: evento `beforeinstallprompt` disponível (Android/Chrome favorável)
- `isIos`: iOS Safari fora de standalone
- `isStandalone`: já instalado

Precisaria adicionar:
- `isWebview`: user-agent indica webview
- `installDismissed`: usuário já fechou o banner (localStorage)
- `sessionCount`: número de sessões (para dosagem do banner)

### Detecção de webview (Android)
```typescript
function isAndroidWebview(): boolean {
  const ua = navigator.userAgent;
  return /wv\)/.test(ua) ||           // Chrome WebView oficial
    /FBAN|FBAV|Instagram/.test(ua) || // Facebook/Instagram
    /Twitter/.test(ua) ||
    (/Android/.test(ua) && !/Chrome\//.test(ua) && !/Firefox\//.test(ua));
}
```

### Critérios do Chrome para `beforeinstallprompt`
O Chrome requer:
- HTTPS ✅ (rachao.app já usa)
- Web App Manifest com `name`, `icons`, `start_url`, `display: standalone` ✅
- Service Worker registrado ✅
- **Engajamento mínimo**: O usuário precisou interagir com o domínio por pelo menos 30 segundos

O critério de engajamento é o principal bloqueador — usuários que chegam via link direto e não exploram o app não atingem o threshold.

---

## 6. Critérios de Aceitação (para implementação futura)

- [ ] Usuários em webview (WhatsApp/Instagram) veem instruções para abrir no Chrome antes de qualquer outra coisa
- [ ] Usuários em Chrome Android sem `beforeinstallprompt` veem instruções com o caminho manual ⋮ → "Adicionar à tela inicial"
- [ ] Existe uma URL `/instalar` ou `/download` compartilhável, com instruções por dispositivo
- [ ] O banner na dashboard é exibido nas primeiras 3 sessões de usuários não instalados e pode ser dispensado
- [ ] O banner não aparece para usuários já em modo standalone
- [ ] Toda a lógica usa apenas detecção client-side (sem dependência de APIs externas)

---

## 7. O que NÃO está no escopo

- **App nativo (Play Store):** Exigiria React Native ou Flutter, custo e manutenção incompatíveis com o estágio atual do produto
- **TWA (Trusted Web Activity):** Publicar o PWA na Play Store como TWA é uma alternativa válida a médio prazo, mas requer conta de desenvolvedor Google ($25 único) e processo de publicação — pode ser avaliado separadamente
- **Geração de APK:** Ferramentas como PWABuilder geram um APK a partir do PWA para distribuição direta (`.apk` via link), mas abre questões de confiança do usuário (instalar APK de fonte desconhecida)
