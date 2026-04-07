# Android TWA — Estado atual

> Este arquivo documenta o **estado corrente** do projeto Android.
> Atualizar sempre que o twa-manifest.json, versão do app ou estrutura mudar.

---

## Status

- [ ] `bubblewrap init` executado localmente — projeto Gradle ainda não gerado
- [ ] `assetlinks.json` preenchido com SHA-256 real da keystore
- [ ] Build local (`bubblewrap build`) testado com sucesso
- [ ] App testado em dispositivo físico Android
- [ ] GitHub Secrets configurados (ver abaixo)
- [ ] Primeira submissão na Play Store (track: internal)

---

## Versão atual do app

| Campo | Valor |
|-------|-------|
| `appVersionCode` | 1 (incrementar a cada release) |
| `appVersionName` | 1.0.0 (semver: major.minor.patch) |
| Package name | `app.rachao` |

> Incrementar `appVersionCode` a cada novo release no `twa-manifest.json`.

---

## Como gerar o projeto pela primeira vez (local)

Pré-requisitos: Node.js, Java JDK 17, Android SDK.

```bash
# 1. Gerar keystore (uma única vez — guardar com segurança)
keytool -genkeypair -v \
  -keystore rachao-release.keystore \
  -alias rachao \
  -keyalg RSA -keysize 2048 -validity 10000

# 2. Extrair SHA-256 para o assetlinks.json
keytool -list -v \
  -keystore rachao-release.keystore \
  -alias rachao

# 3. Instalar Bubblewrap
npm install -g @bubblewrap/cli

# 4. Inicializar projeto TWA
cd football-android
bubblewrap init --manifest https://rachao.app/manifest.webmanifest
# Configurar:
#   Package name: app.rachao
#   App name: rachao.app
#   Launch URL: https://rachao.app/
#   Apontar para rachao-release.keystore

# 5. Build local
bubblewrap build

# 6. Instalar em dispositivo físico para teste
bubblewrap install
```

---

## GitHub Secrets necessários

| Secret | Como obter |
|--------|-----------|
| `ANDROID_KEYSTORE_BASE64` | `base64 -w 0 rachao-release.keystore` |
| `ANDROID_KEY_ALIAS` | `rachao` |
| `ANDROID_KEY_PASSWORD` | Senha definida no `keytool` |
| `ANDROID_STORE_PASSWORD` | Senha definida no `keytool` |
| `GOOGLE_PLAY_SERVICE_ACCOUNT_JSON` | Play Console → Setup → API access → Create service account (role: Release Manager) |

---

## Deploy via GitHub Actions

```bash
# Track interno (primeira submissão)
gh workflow run build-twa.yml -f track=internal

# Promover para produção após validação
gh workflow run build-twa.yml -f track=production
```

---

## O que é commitado vs ignorado

```
football-android/
├── ✅ twa-manifest.json     ← fonte da verdade (sem credenciais)
├── ✅ android/              ← projeto Gradle gerado pelo Bubblewrap
│   ├── ✅ app/build.gradle
│   ├── ✅ app/src/
│   ├── ✅ gradlew
│   └── ❌ app/build/        ← gitignore
├── ✅ CLAUDE.md
├── ❌ *.keystore            ← NUNCA commitar
└── ❌ android/.gradle/      ← gitignore
```
