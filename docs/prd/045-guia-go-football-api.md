# PRD 045 — Guia de Estudo Go: construindo a `football-api-go`

| Campo | Valor |
|---|---|
| **Versão** | 1.0 |
| **Status** | 📖 Documento de referência |
| **Autor** | thiagotn |
| **Data** | 2026-05-20 |
| **Referência** | [PRD 044 — football-api-go](044-football-api-go.md) |

---

> **Como usar este guia**
>
> Este documento é um livro de estudo estruturado em torno da implementação prática da `football-api-go` (PRD 044). Cada capítulo introduz conceitos Go e os ancora em decisões reais do projeto. Leia na ordem para ter a progressão correta, mas cada capítulo também funciona como referência isolada.
>
> **Convenção de exemplos:** blocos `// football-api-go: internal/...` indicam o arquivo exato onde o padrão aparecerá na implementação real.

---

## Sumário

**Parte I — A linguagem**
1. [Por que Go para uma API de backend](#cap-1)
2. [Fundamentos: tipos, funções e structs](#cap-2)
3. [Interfaces e a filosofia de composição](#cap-3)
4. [Tratamento de erros idiomático](#cap-4)
5. [Concorrência: goroutines e channels](#cap-5)
6. [Toolchain: módulos, formatação e lint](#cap-6)

**Parte II — Construindo a API (seguindo as fases do PRD 044)**
7. [Estrutura de projeto e configuração](#cap-7) *(Fase 1)*
8. [HTTP com Chi: routers e middlewares](#cap-8) *(Fase 1)*
9. [Banco de dados com pgx/v5 + sqlc](#cap-9) *(Fase 1–2)*
10. [Autenticação: JWT e bcrypt](#cap-10) *(Fase 1)*
11. [Arquitetura: handlers, services e injeção de dependência](#cap-11) *(Fase 2)*
12. [Testes em Go: unitários e de integração](#cap-12) *(Fases 1–5)*
13. [Serviços externos: Stripe, Twilio, Supabase, Anthropic](#cap-13) *(Fases 3–4)*
14. [Middleware avançado: rate limit e feature flags](#cap-14) *(Fase 5)*
15. [Documentação: swaggo/swag + Mintlify](#cap-15) *(Fase 5)*
16. [CI/CD com GitHub Actions](#cap-16) *(Fase 5)*

**Parte III — Referência**
- [Apêndice A — Go vs Python/FastAPI: tabela de equivalências](#apendice-a)
- [Apêndice B — Padrões idiomáticos Go usados no projeto](#apendice-b)
- [Apêndice C — Glossário](#apendice-c)

---

## Parte I — A linguagem

---

<a name="cap-1"></a>
## Capítulo 1 — Por que Go para uma API de backend

### 1.1 O problema que Go resolve

A `football-api/` atual (Python/FastAPI) tem latência p95 de 80–120ms em endpoints simples. O objetivo da `football-api-go` é atingir ≤ 50ms com a mesma lógica e o mesmo banco de dados. Isso não é magia — é resultado de três características da linguagem:

**Compilação para binário nativo.** Go compila diretamente para código de máquina. Não há interpretador, não há JVM, não há GIL. O processo de startup leva menos de 10ms (vs. ~2s do FastAPI com uvicorn).

**Concorrência barata via goroutines.** Uma goroutine ocupa ~2KB de stack (vs. ~1MB de uma thread do OS). Um servidor Go pode tratar dezenas de milhares de conexões simultâneas sem o custo de context switching de threads.

**Tipagem estática + compilador.** Erros de tipo são capturados no `go build`, não em runtime. No contexto da football-api-go, isso significa que um campo ausente na query SQL é erro de compilação, não um `AttributeError` em produção às 2h da manhã.

### 1.2 O que Go deliberadamente não tem

Entender as ausências é tão importante quanto as features.

| Ausente | Por quê importa |
|---|---|
| Herança de classes | Go usa composição via embedding e interfaces. Liberdade de design sem diamante de herança. |
| Exceções (`try/catch`) | Erros são valores retornados explicitamente. Cada falha é tratada onde ocorre. |
| Generics (limitados antes de 1.18) | Presentes desde Go 1.18, mas com escopo intencional. Código Go favorece explicitação. |
| `null` como estado implícito | Ponteiros `nil` são explícitos. Interfaces nil têm semântica conhecida. |
| Overloading de funções | Uma função, uma assinatura. Facilita leitura de código alheio. |

### 1.3 Go no contexto da football-api-go

O projeto usa Go para exercitar precisamente os pontos fortes da linguagem:

- **Chi + net/http**: sem framework "mágico", cada request é um `http.Handler` — uma interface de 1 método
- **pgx/v5 + sqlc**: SQL explícito, resultados tipados, zero reflection em runtime
- **Goroutines no endpoint de chat**: SSE com Anthropic usa goroutine para streaming sem bloquear o servidor
- **Interfaces para testabilidade**: cada service tem uma interface → handler testável sem banco real

---

<a name="cap-2"></a>
## Capítulo 2 — Fundamentos: tipos, funções e structs

### 2.1 Tipos básicos

```go
// Tipos que aparecem frequentemente na football-api-go
var id       int64          // PKs do banco
var name     string         // nomes de jogadores, grupos
var score    float64        // skill_stars
var enabled  bool           // api_v2_enabled, chat_enabled
var created  time.Time      // timestamps

// Zero values: cada tipo tem valor padrão sem inicialização
var count int     // 0
var flag  bool    // false
var label string  // ""
```

**Zero value é uma feature, não um bug.** Structs recém-criadas já têm todos os campos inicializados com zero values — não é preciso inicializar campo a campo.

### 2.2 Declaração de variáveis

```go
// Três formas — use a mais curta aplicável:

// 1. var explícito (use quando o tipo não é óbvio pelo valor)
var timeout time.Duration = 30 * time.Second

// 2. Short declaration com := (mais comum dentro de funções)
player, err := repo.GetPlayer(ctx, id)

// 3. Constante
const APIPrefix = "/api/v2"
```

### 2.3 Structs

Structs são o principal mecanismo de agrupamento de dados em Go. Na football-api-go, cada entidade do banco tem uma struct correspondente.

```go
// football-api-go: internal/db/queries/players.sql.go (gerado pelo sqlc)
type Player struct {
    ID           int64          `json:"id"`
    WhatsApp     string         `json:"whatsapp"`
    Name         string         `json:"name"`
    Role         PlayerRole     `json:"role"`
    ApiV2Enabled bool           `json:"api_v2_enabled"`
    ChatEnabled  bool           `json:"chat_enabled"`
    CreatedAt    time.Time      `json:"created_at"`
    AvatarURL    *string        `json:"avatar_url,omitempty"` // ponteiro = nullable
}
```

**Struct tags** (as strings entre backticks) são metadados lidos em runtime por pacotes como `encoding/json`. `json:"id"` define o nome do campo no JSON. `omitempty` omite o campo se for zero value.

### 2.4 Funções

```go
// Assinatura padrão
func NomeFunc(param1 Tipo1, param2 Tipo2) (RetornoTipo, error) {
    // ...
    return valor, nil // nil = sem erro
}

// Múltiplos retornos — idioma fundamental de Go
func GetPlayer(ctx context.Context, id int64) (*Player, error) {
    row := db.QueryRow(ctx, "SELECT * FROM players WHERE id = $1", id)
    var p Player
    if err := row.Scan(&p.ID, &p.Name /*...*/); err != nil {
        return nil, fmt.Errorf("GetPlayer %d: %w", id, err)
    }
    return &p, nil
}
```

**Padrão de retorno:** quase toda função na football-api-go retorna `(Valor, error)`. Se erro != nil, o valor não deve ser usado.

### 2.5 Ponteiros

```go
// *T = ponteiro para T. &v = endereço de v. *p = valor apontado por p.

player := Player{Name: "Zico"}
ptr := &player      // *Player, aponta para player
ptr.Name = "Romário" // modifica o original (mesma memória)

// Por que usar ponteiros na football-api-go?
// 1. Para representar NULL do banco: *string, *int64, *time.Time
// 2. Para evitar cópia de structs grandes em parâmetros de função
// 3. Para métodos que modificam o receptor (receivers com *)
```

### 2.6 Slices e maps

```go
// Slice: lista dinâmica (análogo à list do Python)
players := []Player{}
players = append(players, Player{Name: "Pelé"})

// Map: dicionário (análogo ao dict do Python)
headers := map[string]string{
    "Content-Type": "application/json",
    "X-Request-ID": requestID,
}

// Verificar existência no map
value, ok := headers["Authorization"]
if !ok {
    // chave não existe
}
```

---

<a name="cap-3"></a>
## Capítulo 3 — Interfaces e a filosofia de composição

### 3.1 O que é uma interface em Go

Uma interface define um conjunto de métodos. Qualquer tipo que implemente esses métodos satisfaz a interface — **sem declaração explícita** (`implements` não existe em Go).

```go
// Uma interface de 1 método — o coração de net/http
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}

// Qualquer struct com ServeHTTP implementa Handler.
// Chi monta uma árvore de Handlers.
```

### 3.2 Interfaces para testabilidade — padrão central da football-api-go

Este é o padrão mais importante do projeto. Cada service tem uma interface, o handler depende da interface, os testes injetam um mock.

```go
// football-api-go: internal/services/auth_service.go

// Interface — define o contrato
type AuthService interface {
    Login(ctx context.Context, whatsapp, password string) (*LoginResponse, error)
    Register(ctx context.Context, req RegisterRequest) (*Player, error)
    RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
}

// Implementação real
type authService struct {
    db      *pgxpool.Pool
    queries *db.Queries  // sqlc gerado
}

func NewAuthService(pool *pgxpool.Pool, q *db.Queries) AuthService {
    return &authService{db: pool, queries: q}
}

// Handler — depende da INTERFACE, não da implementação
type authHandler struct {
    svc AuthService  // qualquer coisa que implemente AuthService
}
```

```go
// football-api-go: tests/unit/auth_test.go

// Mock implementa AuthService
type mockAuthService struct {
    loginFn func(ctx context.Context, w, p string) (*LoginResponse, error)
}

func (m *mockAuthService) Login(ctx context.Context, w, p string) (*LoginResponse, error) {
    return m.loginFn(ctx, w, p)
}
// ... outros métodos com implementação vazia ou panic

// No teste:
h := authHandler{
    svc: &mockAuthService{
        loginFn: func(_ context.Context, _, _ string) (*LoginResponse, error) {
            return &LoginResponse{Token: "fake-token"}, nil
        },
    },
}
```

### 3.3 Embedding — composição de structs

```go
// Em vez de herança, Go usa embedding.
// A football-api-go usa isso para erros padronizados:

// football-api-go: internal/apierror/errors.go
type APIError struct {
    Code    int    `json:"-"`
    Detail  string `json:"detail"`
}

func (e *APIError) Error() string { return e.Detail }

type NotFoundError struct {
    APIError          // embed: NotFoundError "herda" os campos e métodos
    Resource string
}

// NotFoundError satisfaz a interface error automaticamente
// porque APIError tem o método Error() string
```

---

<a name="cap-4"></a>
## Capítulo 4 — Tratamento de erros idiomático

### 4.1 Erro como valor

Em Go, erro não é uma exceção. É um valor de retorno como qualquer outro.

```go
// ❌ Não faça (anti-padrão: ignorar erro)
player, _ := repo.GetPlayer(ctx, id)

// ✅ Faça (tratar em cada nível)
player, err := repo.GetPlayer(ctx, id)
if err != nil {
    return nil, fmt.Errorf("authHandler.Login: %w", err)
}
```

### 4.2 Wrapping com `%w` — rastreabilidade de erros

```go
// %w "embrulha" o erro original, preservando o tipo para errors.Is / errors.As
func (s *authService) Login(ctx context.Context, whatsapp, password string) (*LoginResponse, error) {
    player, err := s.queries.GetPlayerByWhatsApp(ctx, whatsapp)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, &apierror.NotFoundError{Detail: "player not found"}
        }
        return nil, fmt.Errorf("authService.Login: %w", err)
    }
    // ...
}
```

### 4.3 Erros de domínio na football-api-go

O padrão espelha as exceções do FastAPI Python:

```go
// football-api-go: internal/apierror/errors.go

type APIError struct {
    Code   int    `json:"-"`
    Detail string `json:"detail"`
}
func (e *APIError) Error() string { return e.Detail }

// Equivalentes das exceções Python:
func NotFound(msg string) error  { return &APIError{Code: 404, Detail: msg} }
func Forbidden(msg string) error { return &APIError{Code: 403, Detail: msg} }
func Conflict(msg string) error  { return &APIError{Code: 409, Detail: msg} }
func Unprocessable(msg string) error { return &APIError{Code: 422, Detail: msg} }
func TooManyRequests() error     { return &APIError{Code: 429, Detail: "rate limit exceeded"} }
```

```go
// No middleware de recovery do Chi — captura qualquer panic e converte em 500:
// football-api-go: internal/middleware/recovery.go

func Recovery(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rec := recover(); rec != nil {
                renderError(w, fmt.Errorf("panic: %v", rec), http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

### 4.4 Função helper `renderError` — escrita de erro em JSON

```go
// football-api-go: internal/handlers/helpers.go

func renderError(w http.ResponseWriter, err error, fallbackCode int) {
    var apiErr *apierror.APIError
    if errors.As(err, &apiErr) {
        renderJSON(w, apiErr.Code, apiErr)
        return
    }
    renderJSON(w, fallbackCode, map[string]string{"detail": "internal error"})
}

func renderJSON(w http.ResponseWriter, code int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(v)
}
```

---

<a name="cap-5"></a>
## Capítulo 5 — Concorrência: goroutines e channels

### 5.1 Goroutines

Uma goroutine é uma função executada de forma concorrente. O custo de inicialização é mínimo (~2KB de stack, crescimento dinâmico).

```go
// Lançar uma goroutine:
go funcao()
go func() {
    // código anônimo concorrente
}()
```

### 5.2 Context — o mecanismo de cancelamento

`context.Context` é passado como primeiro parâmetro em **todas** as funções que fazem I/O na football-api-go. Serve para:
1. **Cancelamento**: quando o cliente desconecta, o context é cancelado, a query é abortada
2. **Timeout**: `context.WithTimeout` define prazo máximo para operações
3. **Propagação de valores**: o player autenticado é armazenado no context pelo middleware de auth

```go
// O context vive na request HTTP. Chi injeta na request:
func (h *groupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // context da request — cancelado se cliente desconectar

    id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
    group, err := h.svc.GetGroup(ctx, id) // ctx propagado até o banco
    if err != nil {
        renderError(w, err, http.StatusInternalServerError)
        return
    }
    renderJSON(w, http.StatusOK, group)
}
```

### 5.3 Goroutines no endpoint de chat (SSE)

O endpoint `POST /api/v2/chat` usa Server-Sent Events: o servidor mantém a conexão aberta e envia chunks de texto conforme o modelo de IA responde. Goroutines tornam isso natural:

```go
// football-api-go: internal/handlers/chat.go (simplificado)

func (h *chatHandler) Chat(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Configura headers SSE
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        renderError(w, errors.New("streaming not supported"), http.StatusInternalServerError)
        return
    }

    // Channel para receber chunks do Anthropic SDK
    chunks := make(chan string)
    errCh  := make(chan error, 1)

    go func() {
        defer close(chunks)
        // Chama Anthropic SDK com streaming
        stream, err := h.anthropic.CreateMessageStream(ctx, req)
        if err != nil {
            errCh <- err
            return
        }
        for stream.Next() {
            event := stream.Current()
            if event.Type == "content_block_delta" {
                chunks <- event.Delta.Text
            }
        }
        errCh <- stream.Err()
    }()

    // Loop principal: escreve chunks para o cliente enquanto chegam
    for {
        select {
        case chunk, ok := <-chunks:
            if !ok {
                fmt.Fprintf(w, "data: [DONE]\n\n")
                flusher.Flush()
                return
            }
            fmt.Fprintf(w, "data: %s\n\n", chunk)
            flusher.Flush()
        case err := <-errCh:
            if err != nil {
                fmt.Fprintf(w, "data: [ERROR]\n\n")
                flusher.Flush()
            }
            return
        case <-ctx.Done(): // cliente desconectou
            return
        }
    }
}
```

### 5.4 sync.Mutex — proteção de estado compartilhado

O rate limiter da football-api-go mantém contadores em memória. Múltiplas goroutines (requests concorrentes) leem e escrevem esses contadores — `sync.Mutex` garante acesso exclusivo:

```go
// football-api-go: internal/middleware/ratelimit.go

type ipRateLimiter struct {
    mu      sync.Mutex
    buckets map[string]*bucket
}

func (l *ipRateLimiter) Allow(ip string) bool {
    l.mu.Lock()         // exclusão mútua — apenas uma goroutine entra
    defer l.mu.Unlock() // liberado ao sair da função
    // ... leitura e escrita do bucket
}
```

---

<a name="cap-6"></a>
## Capítulo 6 — Toolchain: módulos, formatação e lint

### 6.1 Go Modules (`go.mod`)

```
module github.com/thiagotn/football-manager/football-api-go

go 1.24

require (
    github.com/go-chi/chi/v5    v5.2.1
    github.com/jackc/pgx/v5     v5.7.4
    github.com/golang-jwt/jwt/v5 v5.2.1
    // ...
)
```

- `go get pacote@versão` — adiciona dependência
- `go mod tidy` — remove imports não utilizados, baixa os que faltam
- `go mod vendor` — copia dependências para `vendor/` (opcional, preferido em CI)

### 6.2 Comandos essenciais

```bash
go build ./...          # compila tudo (verifica erros de tipo)
go test ./...           # roda todos os testes
go test -race ./...     # detecta race conditions (obrigatório em CI)
go vet ./...            # análise estática básica
gofmt -w .              # formata código (ou: goimports)
golangci-lint run       # lint avançado (configurado em .golangci.yml)
```

### 6.3 `golangci-lint` na football-api-go

O CI roda `golangci-lint` antes dos testes. Linters ativos:

```yaml
# football-api-go/.golangci.yml
linters:
  enable:
    - errcheck       # detecta erros ignorados (_, err pattern)
    - govet          # análise estática do compilador
    - staticcheck    # checks adicionais (SA*, S*, ST*)
    - gosec          # vulnerabilidades de segurança
    - misspell       # erros ortográficos em comentários
    - goimports      # imports organizados
```

### 6.4 `sqlc` — geração de código do banco

```bash
sqlc generate   # lê sqlc.yaml, processa sql/queries/*.sql, gera internal/db/queries/
```

```yaml
# football-api-go/sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "sql/queries/"
    schema: "../football-api/migrations/"  # reutiliza migrations Python
    gen:
      go:
        package: "db"
        out: "internal/db/queries"
        emit_json_tags: true
        emit_pointers_for_null_types: true
```

### 6.5 `air` — live-reload em desenvolvimento

```bash
air   # detecta mudanças em *.go e reinicia o servidor automaticamente
```

```toml
# football-api-go/.air.toml
[build]
cmd = "go build -o ./tmp/main ./cmd/server/main.go"
bin = "./tmp/main"
include_ext = ["go"]
exclude_dir = ["tests", "mintlify"]
```

---

## Parte II — Construindo a API

---

<a name="cap-7"></a>
## Capítulo 7 — Estrutura de projeto e configuração *(Fase 1)*

### 7.1 Convenção de layout: `cmd/` e `internal/`

Go não tem uma estrutura obrigatória, mas a comunidade consolidou um padrão:

```
football-api-go/
├── cmd/server/main.go    # entrypoint público — o que o binário executa
├── internal/             # código privado — não importável por outros módulos
│   ├── config/           # configuração
│   ├── db/               # acesso ao banco (pool + sqlc gerado)
│   ├── middleware/        # middlewares Chi
│   ├── handlers/          # handlers HTTP (um por domínio)
│   ├── services/          # lógica de negócio
│   └── apierror/          # tipos de erro padronizados
├── sql/queries/           # arquivos .sql para o sqlc
└── tests/                 # testes de integração (banco real)
```

**Por que `internal/`?** O compilador Go proíbe que outros módulos importem pacotes dentro de `internal/`. Isso garante que a football-api-go não se torna uma biblioteca acidentalmente — cada pacote é privado por default.

### 7.2 Configuração com `envconfig`

```go
// football-api-go: internal/config/config.go

package config

import "github.com/kelseyhightower/envconfig"

type Config struct {
    DatabaseURL   string `envconfig:"DATABASE_URL"   required:"true"`
    SecretKey     string `envconfig:"SECRET_KEY"      required:"true"`
    AppEnv        string `envconfig:"APP_ENV"         default:"development"`
    Port          int    `envconfig:"PORT"            default:"8080"`

    // Twilio
    TwilioAccountSID string `envconfig:"TWILIO_ACCOUNT_SID"`
    TwilioAuthToken  string `envconfig:"TWILIO_AUTH_TOKEN"`
    TwilioVerifySID  string `envconfig:"TWILIO_VERIFY_SID"`

    // Stripe
    StripeSecretKey     string `envconfig:"STRIPE_SECRET_KEY"`
    StripeWebhookSecret string `envconfig:"STRIPE_WEBHOOK_SECRET"`

    // Anthropic
    AnthropicAPIKey string `envconfig:"ANTHROPIC_API_KEY"`
    LLMModel        string `envconfig:"LLM_MODEL" default:"claude-haiku-4-5"`

    // VAPID
    VAPIDPrivateKey string `envconfig:"VAPID_PRIVATE_KEY"`
    VAPIDPublicKey  string `envconfig:"VAPID_PUBLIC_KEY"`
}

func Load() (*Config, error) {
    var c Config
    if err := envconfig.Process("", &c); err != nil {
        return nil, err
    }
    return &c, nil
}
```

**Equivalente Python:** `pydantic.BaseSettings`. A diferença é que o Go valida em runtime na inicialização — se `DATABASE_URL` não estiver definida, o servidor não sobe.

### 7.3 Entrypoint: `cmd/server/main.go`

```go
// football-api-go: cmd/server/main.go

package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/thiagotn/football-manager/football-api-go/internal/config"
    "github.com/thiagotn/football-manager/football-api-go/internal/db"
    "github.com/thiagotn/football-manager/football-api-go/internal/server"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("config: %v", err)
    }

    pool, err := db.NewPool(cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("db: %v", err)
    }
    defer pool.Close()

    router := server.NewRouter(cfg, pool)

    srv := &http.Server{
        Addr:         fmt.Sprintf(":%d", cfg.Port),
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 60 * time.Second, // 60s para SSE do chat
        IdleTimeout:  120 * time.Second,
    }

    // Graceful shutdown: espera requests em andamento terminarem
    go func() {
        log.Printf("server listening on :%d", cfg.Port)
        if err := srv.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("server: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit // bloqueia até receber sinal

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
    log.Println("server shutdown gracefully")
}
```

**Conceitos novos aqui:**
- `log.Fatalf` — loga e chama `os.Exit(1)`, usado apenas no `main`
- `defer` — executa a função quando o escopo retorna (aqui: fecha o pool ao sair do `main`)
- `os.Signal` + `signal.Notify` — recebe SIGTERM do Docker para graceful shutdown
- `chan os.Signal` — channel tipado para comunicação entre goroutines

---

<a name="cap-8"></a>
## Capítulo 8 — HTTP com Chi: routers e middlewares *(Fase 1)*

### 8.1 Por que Chi em vez de `net/http` puro

`net/http` já tem tudo que precisa para um servidor HTTP. Chi é apenas açúcar que adiciona:
- Parâmetros de URL (`/groups/{id}`) com `chi.URLParam`
- Grupos de rotas com middleware seletivo
- Métodos HTTP como `.Get()`, `.Post()`, etc.

Tudo retorna `http.Handler` — compatível com qualquer middleware da stdlib.

### 8.2 Estrutura do router

```go
// football-api-go: internal/server/router.go

package server

import (
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    internalMiddleware "github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

func NewRouter(cfg *config.Config, pool *pgxpool.Pool) http.Handler {
    q := db.New(pool) // queries sqlc

    // Instanciar services
    authSvc  := services.NewAuthService(pool, q, cfg)
    groupSvc := services.NewGroupService(pool, q)
    // ... demais services

    // Instanciar handlers
    authH  := handlers.NewAuthHandler(authSvc)
    groupH := handlers.NewGroupHandler(groupSvc)
    // ...

    r := chi.NewRouter()

    // Middlewares globais (todos os requests)
    r.Use(middleware.RequestID)      // X-Request-Id header
    r.Use(middleware.RealIP)         // X-Forwarded-For → r.RemoteAddr
    r.Use(middleware.Logger)         // log de cada request
    r.Use(internalMiddleware.CORS(cfg))
    r.Use(internalMiddleware.Recovery)

    // Health check — sem autenticação, sem api_v2_enabled gate
    r.Get("/api/v2/health", func(w http.ResponseWriter, r *http.Request) {
        renderJSON(w, 200, map[string]string{"status": "ok"})
    })

    // Rotas públicas — sem autenticação
    r.Route("/api/v2", func(r chi.Router) {
        r.Mount("/auth",    authH.PublicRoutes())
        r.Mount("/invites", inviteH.PublicRoutes())
        r.Mount("/ranking", rankingH.Routes())
        r.Mount("/matches", matchH.PublicRoutes())

        // Rotas autenticadas — JWT obrigatório + api_v2_enabled gate
        r.Group(func(r chi.Router) {
            r.Use(internalMiddleware.Auth(cfg.SecretKey, q))         // valida JWT
            r.Use(internalMiddleware.ApiV2Access(q))                 // verifica api_v2_enabled
            r.Mount("/groups",    groupH.AuthRoutes())
            r.Mount("/players",   playerH.AuthRoutes())
            r.Mount("/votes",     voteH.Routes())
            r.Mount("/finance",   financeH.Routes())
            r.Mount("/chat",      chatH.Routes())
            r.Mount("/mcp-tokens", mcpTokenH.Routes())
            // ...
        })

        // Rotas super-admin
        r.Group(func(r chi.Router) {
            r.Use(internalMiddleware.Auth(cfg.SecretKey, q))
            r.Use(internalMiddleware.RequireAdmin)
            r.Mount("/admin", adminH.Routes())
        })
    })

    return r
}
```

### 8.3 Anatomia de um handler

```go
// football-api-go: internal/handlers/groups.go

type groupHandler struct {
    svc services.GroupService
}

func NewGroupHandler(svc services.GroupService) *groupHandler {
    return &groupHandler{svc: svc}
}

func (h *groupHandler) AuthRoutes() http.Handler {
    r := chi.NewRouter()
    r.Get("/",         h.ListGroups)
    r.Post("/",        h.CreateGroup)
    r.Get("/{id}",     h.GetGroup)
    r.Patch("/{id}",   h.UpdateGroup)
    r.Delete("/{id}",  h.DeleteGroup)
    // ...
    return r
}

// @Summary     Get group by ID
// @Tags        groups
// @Security    BearerAuth
// @Param       id   path  int  true  "Group ID"
// @Success     200  {object}  schemas.GroupResponse
// @Failure     404  {object}  apierror.APIError
// @Router      /groups/{id} [get]
func (h *groupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
    // 1. Extrair parâmetros
    id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
    if err != nil {
        renderError(w, apierror.Unprocessable("invalid id"), 0)
        return
    }

    // 2. Extrair player do context (colocado pelo middleware Auth)
    player := middleware.PlayerFromCtx(r.Context())

    // 3. Chamar service
    group, err := h.svc.GetGroup(r.Context(), id, player.ID)
    if err != nil {
        renderError(w, err, http.StatusInternalServerError)
        return
    }

    // 4. Responder
    renderJSON(w, http.StatusOK, group)
}
```

### 8.4 Leitura de request body (JSON binding)

```go
// football-api-go: internal/handlers/helpers.go

func decodeJSON(r *http.Request, dst any) error {
    defer r.Body.Close()
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields() // retorna erro se campo inesperado
    if err := dec.Decode(dst); err != nil {
        return apierror.Unprocessable(fmt.Sprintf("invalid body: %v", err))
    }
    return nil
}

// Uso em um handler:
type CreateGroupRequest struct {
    Name     string `json:"name"`
    IsPublic bool   `json:"is_public"`
}

func (h *groupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
    var req CreateGroupRequest
    if err := decodeJSON(r, &req); err != nil {
        renderError(w, err, 0)
        return
    }
    // ...
}
```

### 8.5 Middleware de autenticação

```go
// football-api-go: internal/middleware/auth.go

type contextKey string
const playerKey contextKey = "player"

func Auth(secretKey string, q *db.Queries) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractBearerToken(r)
            if token == "" {
                renderError(w, apierror.Unauthorized(), 0)
                return
            }

            var player *db.Player
            var err error

            if strings.HasPrefix(token, "rachao_") {
                // MCP token — busca no banco
                player, err = q.GetPlayerByMCPToken(r.Context(), token)
            } else {
                // JWT — valida e extrai player_id
                player, err = validateJWT(r.Context(), q, token, secretKey)
            }

            if err != nil {
                renderError(w, apierror.Unauthorized(), 0)
                return
            }

            // Armazena player no context para handlers downstream
            ctx := context.WithValue(r.Context(), playerKey, player)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func PlayerFromCtx(ctx context.Context) *db.Player {
    return ctx.Value(playerKey).(*db.Player)
}
```

---

<a name="cap-9"></a>
## Capítulo 9 — Banco de dados com pgx/v5 + sqlc *(Fases 1–2)*

### 9.1 Por que pgx/v5

`database/sql` é a interface padrão do Go para bancos. pgx/v5 implementa essa interface **e** expõe uma API nativa mais eficiente:

- Suporte nativo a arrays PostgreSQL, JSONB, UUID, tipos customizados
- Connection pool embutido (`pgxpool`)
- Tipos Go gerados automaticamente para cada coluna (via sqlc)
- `pgx.ErrNoRows` — equivalente ao `sqlalchemy.orm.exc.NoResultFound` do Python

### 9.2 Pool de conexões

```go
// football-api-go: internal/db/pool.go

package db

import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(databaseURL string) (*pgxpool.Pool, error) {
    config, err := pgxpool.ParseConfig(databaseURL)
    if err != nil {
        return nil, fmt.Errorf("db.NewPool parse: %w", err)
    }

    config.MaxConns = 20                // máximo de conexões simultâneas
    config.MinConns = 2                 // mantém 2 abertas mesmo sem load
    config.MaxConnLifetime = time.Hour  // recria conexão após 1h

    pool, err := pgxpool.NewWithConfig(context.Background(), config)
    if err != nil {
        return nil, fmt.Errorf("db.NewPool connect: %w", err)
    }

    if err := pool.Ping(context.Background()); err != nil {
        return nil, fmt.Errorf("db.NewPool ping: %w", err)
    }

    return pool, nil
}
```

### 9.3 sqlc: SQL primeiro, código depois

O fluxo sqlc é:
1. Escreva SQL normal em `sql/queries/*.sql`
2. Adicione comentários com nome e tipo da query
3. `sqlc generate` cria funções Go tipadas em `internal/db/queries/`

```sql
-- football-api-go: sql/queries/players.sql

-- name: GetPlayerByID :one
SELECT * FROM players WHERE id = $1;

-- name: GetPlayerByWhatsApp :one
SELECT * FROM players WHERE whatsapp = $1;

-- name: ListPlayers :many
SELECT id, name, whatsapp, role, created_at, api_v2_enabled
FROM players
WHERE role != 'admin'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdatePlayerApiV2Access :exec
UPDATE players
SET api_v2_enabled = $2
WHERE id = $1;

-- name: CreatePlayer :one
INSERT INTO players (name, whatsapp, password_hash, role)
VALUES ($1, $2, $3, $4)
RETURNING *;
```

**sqlc gera automaticamente:**

```go
// football-api-go: internal/db/queries/players.sql.go (NÃO EDITAR — gerado)

func (q *Queries) GetPlayerByID(ctx context.Context, id int64) (Player, error) {
    row := q.db.QueryRow(ctx, getPlayerByID, id)
    var p Player
    err := row.Scan(&p.ID, &p.WhatsApp, &p.Name, &p.Role, &p.CreatedAt, &p.ApiV2Enabled)
    return p, err
}

func (q *Queries) ListPlayers(ctx context.Context, arg ListPlayersParams) ([]Player, error) {
    rows, err := q.db.Query(ctx, listPlayers, arg.Limit, arg.Offset)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var players []Player
    for rows.Next() {
        var p Player
        if err := rows.Scan(&p.ID, &p.Name /*...*/); err != nil {
            return nil, err
        }
        players = append(players, p)
    }
    return players, rows.Err()
}
```

### 9.4 Transações

```go
// football-api-go: internal/services/group_service.go

// Criar grupo + adicionar criador como admin — em transação
func (s *groupService) CreateGroup(ctx context.Context, req CreateGroupRequest, playerID int64) (*db.Group, error) {
    tx, err := s.pool.Begin(ctx)
    if err != nil {
        return nil, fmt.Errorf("CreateGroup begin tx: %w", err)
    }
    defer tx.Rollback(ctx) // rollback automático se Commit não for chamado

    q := s.queries.WithTx(tx) // mesmas queries, mas dentro da transação

    group, err := q.CreateGroup(ctx, db.CreateGroupParams{
        Name:      req.Name,
        CreatedBy: playerID,
    })
    if err != nil {
        return nil, fmt.Errorf("CreateGroup insert: %w", err)
    }

    _, err = q.AddGroupMember(ctx, db.AddGroupMemberParams{
        GroupID:  group.ID,
        PlayerID: playerID,
        IsAdmin:  true,
    })
    if err != nil {
        return nil, fmt.Errorf("CreateGroup add member: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return nil, fmt.Errorf("CreateGroup commit: %w", err)
    }
    return &group, nil
}
```

### 9.5 Migrations com golang-migrate

```bash
# Aplicar todas as migrations (apontando para as migrations Python)
migrate -path ../football-api/migrations \
        -database "postgres://user:pass@localhost/rachao?sslmode=disable" up

# Verificar versão atual
migrate -path ../football-api/migrations \
        -database "..." version
```

```makefile
# football-api-go: Makefile
migrate:
    migrate -path ../football-api/migrations \
            -database "$(DATABASE_URL)" up
```

---

<a name="cap-10"></a>
## Capítulo 10 — Autenticação: JWT e bcrypt *(Fase 1)*

### 10.1 bcrypt para senhas

```go
// football-api-go: internal/services/auth_service.go

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12 // mesmo custo da API Python

func hashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
    return string(bytes), err
}

func checkPassword(password, hash string) bool {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
```

**Cross-compatibilidade com Python:** o hash `$2b$12$...` gerado pelo Python `passlib` é compatível com Go `bcrypt`. Um usuário criado na Python API pode autenticar na Go API e vice-versa.

### 10.2 JWT com `golang-jwt/jwt/v5`

```go
// football-api-go: internal/services/auth_service.go

type Claims struct {
    PlayerID int64  `json:"player_id"`
    Role     string `json:"role"`
    jwt.RegisteredClaims
}

func (s *authService) GenerateTokenPair(player *db.Player) (*TokenPair, error) {
    // Access token: 15 minutos
    accessClaims := Claims{
        PlayerID: player.ID,
        Role:     string(player.Role),
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).
        SignedString([]byte(s.secretKey))
    if err != nil {
        return nil, err
    }

    // Refresh token: 7 dias
    refreshClaims := Claims{
        PlayerID: player.ID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
        },
    }
    refreshToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).
        SignedString([]byte(s.secretKey))

    return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *authService) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{},
        func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method")
            }
            return []byte(s.secretKey), nil
        })
    if err != nil {
        return nil, err
    }
    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }
    return nil, errors.New("invalid token")
}
```

**Cross-compatibilidade com Python:** o `SECRET_KEY` é o mesmo. HS256 é o mesmo algoritmo. Claims `player_id` e `role` são os mesmos. Um token gerado em `/api/v1/auth/login` (Python) é aceito em `/api/v2/auth/me` (Go) — e vice-versa.

---

<a name="cap-11"></a>
## Capítulo 11 — Arquitetura: handlers, services e injeção de dependência *(Fase 2)*

### 11.1 As três camadas

```
Request HTTP
    ↓
Handler         — decodifica request, chama service, serializa response
    ↓
Service         — regras de negócio, orquestração entre repositories
    ↓
sqlc Queries    — acesso ao banco (gerado, não tem lógica de negócio)
    ↓
PostgreSQL
```

**Por que essa separação importa:** no teste unitário, substituímos o service por um mock. A handler continua funcionando sem banco real.

### 11.2 Injeção de dependência manual (sem framework)

Go não tem container de DI como Spring ou FastAPI `Depends`. A injeção é feita manualmente no `NewRouter`:

```go
// Construção do grafo de dependências no main/router:

// Nível 3: banco
pool := db.NewPool(cfg.DatabaseURL)
q    := db.New(pool)

// Nível 2: services (recebem pool + queries)
authSvc   := services.NewAuthService(pool, q, cfg)
groupSvc  := services.NewGroupService(pool, q)
matchSvc  := services.NewMatchService(pool, q)

// Nível 1: handlers (recebem services via interface)
authH  := handlers.NewAuthHandler(authSvc)
groupH := handlers.NewGroupHandler(groupSvc)
matchH := handlers.NewMatchHandler(matchSvc)

// Nível 0: router (recebe handlers)
r.Mount("/groups", groupH.AuthRoutes())
```

### 11.3 Algoritmo snake-draft (port de Python para Go)

O `team_builder.py` é um dos services mais ricos da football-api — ordena jogadores por skill e distribui entre times em modo serpentina. O port para Go demonstra bem a diferença de estilo:

```go
// football-api-go: internal/services/team_builder.go

type DrawPlayer struct {
    ID         int64
    Name       string
    SkillStars float64
    IsGoalie   bool
    TeamIndex  int // atribuído pelo snake-draft
}

type TeamBuilderService interface {
    DrawTeams(players []DrawPlayer, numTeams int) ([][]DrawPlayer, error)
}

type teamBuilderService struct{}

func (s *teamBuilderService) DrawTeams(players []DrawPlayer, numTeams int) ([][]DrawPlayer, error) {
    if numTeams < 2 {
        return nil, apierror.Unprocessable("minimum 2 teams required")
    }

    // Separar goleiros
    goalies := filterGoalies(players, true)
    field   := filterGoalies(players, false)

    // Ordenar campo por skill desc
    sort.Slice(field, func(i, j int) bool {
        return field[i].SkillStars > field[j].SkillStars
    })

    // Snake-draft: 0,1,2,2,1,0,0,1,2,...
    teams := make([][]DrawPlayer, numTeams)
    direction := 1
    teamIdx   := 0
    for _, p := range field {
        p.TeamIndex = teamIdx
        teams[teamIdx] = append(teams[teamIdx], p)
        teamIdx += direction
        if teamIdx == numTeams {
            teamIdx = numTeams - 1
            direction = -1
        } else if teamIdx < 0 {
            teamIdx = 0
            direction = 1
        }
    }

    // Distribuir goleiros round-robin
    for i, g := range goalies {
        g.TeamIndex = i % numTeams
        teams[i%numTeams] = append(teams[i%numTeams], g)
    }

    return teams, nil
}
```

---

<a name="cap-12"></a>
## Capítulo 12 — Testes em Go: unitários e de integração *(Fases 1–5)*

### 12.1 Filosofia de testes em Go

Go tem testing embutido na stdlib. Não há nenhum framework necessário — `go test` já é suficiente para tudo. `testify` adiciona assertions mais legíveis.

```bash
go test ./internal/...              # todos os testes unitários
go test ./internal/... -v           # verbose (mostra cada test)
go test ./internal/... -run TestAuth # filtra por nome
go test ./internal/... -cover       # cobertura de linhas
go test -race ./...                  # detecta race conditions
```

### 12.2 Teste unitário de handler

```go
// football-api-go: tests/unit/auth_test.go

package unit_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestAuthHandler_Login_Success(t *testing.T) {
    // 1. Arrange: mock do service
    mockSvc := &mockAuthService{
        loginFn: func(ctx context.Context, whatsapp, password string) (*services.LoginResponse, error) {
            return &services.LoginResponse{
                AccessToken:  "access-token",
                RefreshToken: "refresh-token",
                Player:       &db.Player{ID: 1, Name: "Zico"},
            }, nil
        },
    }

    h := handlers.NewAuthHandler(mockSvc)
    r := chi.NewRouter()
    r.Post("/auth/login", h.Login)

    // 2. Act: fazer request
    body := `{"whatsapp":"+5511999990000","password":"senha123"}`
    req  := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    // 3. Assert: verificar resposta
    require.Equal(t, http.StatusOK, w.Code)
    var resp services.LoginResponse
    err := json.NewDecoder(w.Body).Decode(&resp)
    require.NoError(t, err)
    assert.Equal(t, "access-token", resp.AccessToken)
    assert.Equal(t, "Zico", resp.Player.Name)
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
    mockSvc := &mockAuthService{
        loginFn: func(_ context.Context, _, _ string) (*services.LoginResponse, error) {
            return nil, apierror.Forbidden("invalid credentials")
        },
    }

    h := handlers.NewAuthHandler(mockSvc)
    r := chi.NewRouter()
    r.Post("/auth/login", h.Login)

    req := httptest.NewRequest(http.MethodPost, "/auth/login",
        bytes.NewBufferString(`{"whatsapp":"+55...","password":"wrong"}`))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusForbidden, w.Code)
}
```

### 12.3 Helpers compartilhados entre testes

```go
// football-api-go: tests/unit/helpers_test.go

// fakePlayer retorna um player autenticado para injetar no context
func fakePlayer(opts ...func(*db.Player)) *db.Player {
    p := &db.Player{
        ID:           999,
        Name:         "Test Player",
        Role:         db.PlayerRoleUser,
        ApiV2Enabled: true,
    }
    for _, opt := range opts {
        opt(p)
    }
    return p
}

// withPlayer cria um request com player no context (simula middleware Auth)
func withPlayer(r *http.Request, p *db.Player) *http.Request {
    ctx := context.WithValue(r.Context(), middleware.PlayerContextKey, p)
    return r.WithContext(ctx)
}

// asAdmin cria um player com role=admin
func asAdmin() func(*db.Player) {
    return func(p *db.Player) { p.Role = db.PlayerRoleAdmin }
}
```

### 12.4 Teste de integração (banco real)

```go
// football-api-go: tests/integration/auth_integration_test.go

package integration_test

import (
    "context"
    "os"
    "testing"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/stretchr/testify/require"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
    // Setup: conecta ao banco de teste (definido em DATABASE_URL)
    var err error
    testPool, err = pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
    if err != nil {
        panic("integration test: db connection failed: " + err.Error())
    }
    defer testPool.Close()

    os.Exit(m.Run())
}

func TestAuth_RegisterAndLogin(t *testing.T) {
    ctx := context.Background()
    q   := db.New(testPool)

    // Limpa dados de teste ao final
    t.Cleanup(func() {
        testPool.Exec(ctx, "DELETE FROM players WHERE whatsapp = '+5511777770000'")
    })

    // Registra jogador
    authSvc := services.NewAuthService(testPool, q, testCfg())
    player, err := authSvc.Register(ctx, services.RegisterRequest{
        WhatsApp: "+5511777770000",
        Name:     "Integration Test",
        Password: "senha123",
    })
    require.NoError(t, err)
    require.NotZero(t, player.ID)

    // Autentica
    resp, err := authSvc.Login(ctx, "+5511777770000", "senha123")
    require.NoError(t, err)
    require.NotEmpty(t, resp.AccessToken)

    // Verifica que token é válido
    claims, err := authSvc.ValidateToken(resp.AccessToken)
    require.NoError(t, err)
    require.Equal(t, player.ID, claims.PlayerID)
}
```

### 12.5 Teste do middleware `api_v2_access`

```go
// football-api-go: tests/unit/middleware_test.go

func TestApiV2Access_Blocks_DisabledPlayer(t *testing.T) {
    player := fakePlayer(func(p *db.Player) { p.ApiV2Enabled = false })

    protected := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK) // nunca deve chegar aqui
    })

    handler := middleware.ApiV2Access(mockQueries)(protected)

    req := httptest.NewRequest(http.MethodGet, "/api/v2/groups", nil)
    req  = withPlayer(req, player)
    w   := httptest.NewRecorder()

    handler.ServeHTTP(w, req)

    assert.Equal(t, http.StatusForbidden, w.Code)
    assert.Contains(t, w.Body.String(), "API_V2_NOT_ENABLED")
}

func TestApiV2Access_Allows_Admin(t *testing.T) {
    player := fakePlayer(asAdmin(), func(p *db.Player) { p.ApiV2Enabled = false })
    // Admin sempre passa, mesmo com api_v2_enabled = false

    protected := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })

    handler := middleware.ApiV2Access(mockQueries)(protected)

    req := httptest.NewRequest(http.MethodGet, "/api/v2/admin/stats", nil)
    req  = withPlayer(req, player)
    w   := httptest.NewRecorder()

    handler.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)
}
```

---

<a name="cap-13"></a>
## Capítulo 13 — Serviços externos: Stripe, Twilio, Supabase, Anthropic *(Fases 3–4)*

### 13.1 Padrão de wrapper para services externos

Todos os serviços externos seguem o mesmo padrão:
1. Interface que define o contrato
2. Implementação real que chama o SDK
3. Mock que implementa a interface — usado nos testes

```go
// Padrão aplicado ao Twilio:

// Interface
type OTPService interface {
    SendOTP(ctx context.Context, whatsapp string) error
    VerifyOTP(ctx context.Context, whatsapp, code string) (bool, error)
}

// Implementação real
type twilioService struct {
    accountSID string
    authToken  string
    verifySID  string
}

func (s *twilioService) SendOTP(ctx context.Context, whatsapp string) error {
    client := twilio.NewRestClientWithParams(twilio.ClientParams{
        Username: s.accountSID,
        Password: s.authToken,
    })
    params := &openapi.CreateVerificationParams{}
    params.SetTo(whatsapp)
    params.SetChannel("whatsapp") // ou "sms"
    _, err := client.VerifyV2.CreateVerification(s.verifySID, params)
    return err
}

// Mock para testes — não faz chamada real ao Twilio
type mockOTPService struct {
    sendFn   func(ctx context.Context, w string) error
    verifyFn func(ctx context.Context, w, code string) (bool, error)
}
```

### 13.2 Stripe: webhook com HMAC-SHA256

```go
// football-api-go: internal/handlers/webhooks.go

func (h *webhookHandler) PaymentWebhook(w http.ResponseWriter, r *http.Request) {
    const maxBodyBytes = int64(65536)
    r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
    payload, err := io.ReadAll(r.Body)
    if err != nil {
        renderError(w, apierror.Unprocessable("body too large"), 0)
        return
    }

    // Verificação HMAC — essencial para segurança
    event, err := webhook.ConstructEvent(payload,
        r.Header.Get("Stripe-Signature"),
        h.cfg.StripeWebhookSecret)
    if err != nil {
        // HMAC inválido: possivelmente replay attack
        renderError(w, apierror.Forbidden("invalid signature"), 0)
        return
    }

    switch event.Type {
    case "checkout.session.completed":
        // processar assinatura confirmada
    case "customer.subscription.deleted":
        // processar cancelamento
    }

    w.WriteHeader(http.StatusOK)
}
```

**Por que verificar HMAC:** qualquer pessoa na internet pode fazer POST para `/webhooks/payment`. O header `Stripe-Signature` contém um HMAC-SHA256 calculado com o `STRIPE_WEBHOOK_SECRET` que só a Stripe e o servidor conhecem.

### 13.3 Supabase Storage: upload de avatar

```go
// football-api-go: internal/services/storage.go

type StorageService interface {
    UploadAvatar(ctx context.Context, playerID int64, contentType string, data []byte) (string, error)
    DeleteAvatar(ctx context.Context, playerID int64) error
}

type supabaseStorage struct {
    url     string
    apiKey  string
    bucket  string
}

func (s *supabaseStorage) UploadAvatar(ctx context.Context, playerID int64, contentType string, data []byte) (string, error) {
    path := fmt.Sprintf("avatars/%d", playerID)
    url  := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.url, s.bucket, path)

    req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
    req.Header.Set("Authorization", "Bearer "+s.apiKey)
    req.Header.Set("Content-Type", contentType)
    req.Header.Set("x-upsert", "true") // sobrescreve se já existir

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("storage upload: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("storage upload: status %d", resp.StatusCode)
    }

    publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.url, s.bucket, path)
    return publicURL, nil
}
```

---

<a name="cap-14"></a>
## Capítulo 14 — Middleware avançado: rate limit e feature flags *(Fase 5)*

### 14.1 Rate limiter de login (5 tentativas/IP/15 min)

```go
// football-api-go: internal/middleware/ratelimit.go

type bucket struct {
    count    int
    resetAt  time.Time
}

type loginRateLimiter struct {
    mu      sync.Mutex
    buckets map[string]*bucket
    limit   int
    window  time.Duration
}

func NewLoginRateLimiter() *loginRateLimiter {
    l := &loginRateLimiter{
        buckets: make(map[string]*bucket),
        limit:   5,
        window:  15 * time.Minute,
    }
    // Goroutine de limpeza: remove buckets expirados a cada minuto
    go func() {
        ticker := time.NewTicker(time.Minute)
        for range ticker.C {
            l.cleanup()
        }
    }()
    return l
}

func (l *loginRateLimiter) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip := r.RemoteAddr
        if !l.allow(ip) {
            w.Header().Set("Retry-After", "900") // 15 min em segundos
            renderError(w, apierror.TooManyRequests(), 0)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func (l *loginRateLimiter) allow(ip string) bool {
    l.mu.Lock()
    defer l.mu.Unlock()

    b, ok := l.buckets[ip]
    if !ok || time.Now().After(b.resetAt) {
        l.buckets[ip] = &bucket{count: 1, resetAt: time.Now().Add(l.window)}
        return true
    }
    if b.count >= l.limit {
        return false
    }
    b.count++
    return true
}
```

### 14.2 Middleware `api_v2_access` — feature flag por usuário

```go
// football-api-go: internal/middleware/api_v2_access.go

func ApiV2Access(q *db.Queries) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            player := PlayerFromCtx(r.Context())
            if player == nil {
                // Não autenticado — deixa passar (endpoint público)
                // O middleware Auth já bloqueou antes se o endpoint exige auth
                next.ServeHTTP(w, r)
                return
            }

            // Super admin sempre tem acesso
            if player.Role == db.PlayerRoleAdmin {
                next.ServeHTTP(w, r)
                return
            }

            // Usuário comum: verificar flag
            if !player.ApiV2Enabled {
                renderJSON(w, http.StatusForbidden, map[string]string{
                    "detail": "API_V2_NOT_ENABLED",
                })
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

**Sequência de middlewares no router:**
```
request → Auth (valida JWT, popula ctx) → ApiV2Access (verifica flag) → handler
```

O `ApiV2Access` depende do player estar no context, por isso vem **depois** do `Auth`.

---

<a name="cap-15"></a>
## Capítulo 15 — Documentação: swaggo/swag + Mintlify *(Fase 5)*

### 15.1 Annotations swaggo nos handlers

```go
// football-api-go: internal/handlers/groups.go

// @Summary     List groups
// @Description Returns all groups the authenticated player is a member of
// @Tags        groups
// @Security    BearerAuth
// @Success     200  {array}   schemas.GroupResponse
// @Failure     401  {object}  apierror.APIError
// @Router      /groups [get]
func (h *groupHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
    // ...
}

// @Summary     Create group
// @Tags        groups
// @Security    BearerAuth
// @Param       body  body  schemas.CreateGroupRequest  true  "Group data"
// @Success     201   {object}  schemas.GroupResponse
// @Failure     422   {object}  apierror.APIError
// @Router      /groups [post]
func (h *groupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
    // ...
}
```

### 15.2 Geração do OpenAPI

```bash
# Instalar a ferramenta (apenas uma vez)
go install github.com/swaggo/swag/cmd/swag@latest

# Gerar openapi.yaml a partir das annotations
swag init -g cmd/server/main.go \
          --output mintlify/ \
          --outputTypes yaml \
          --parseInternal
```

### 15.3 Configuração Mintlify (`mint.json`)

```json
// football-api-go: mintlify/mint.json
{
  "name": "rachao.app API",
  "logo": {
    "dark": "/logo/dark.svg",
    "light": "/logo/light.svg"
  },
  "favicon": "/favicon.ico",
  "colors": {
    "primary": "#22c55e",
    "light": "#4ade80",
    "dark": "#15803d"
  },
  "topbarLinks": [
    { "name": "GitHub", "url": "https://github.com/thiagotn/football-manager" }
  ],
  "anchors": [
    { "name": "API Reference", "icon": "code", "url": "api-reference" }
  ],
  "navigation": [
    {
      "group": "Getting Started",
      "pages": ["quickstart", "authentication"]
    },
    {
      "group": "Architecture",
      "pages": ["architecture"]
    },
    {
      "group": "API Reference",
      "pages": ["api-reference/overview"]
    }
  ],
  "openapi": "openapi.yaml",
  "baseUrl": "https://api.rachao.app"
}
```

### 15.4 `make docs` no Makefile

```makefile
# football-api-go: Makefile

docs:
    swag init -g cmd/server/main.go --output mintlify/ --outputTypes yaml --parseInternal
    mintlify dev mintlify/  # preview local em http://localhost:3000
```

---

<a name="cap-16"></a>
## Capítulo 16 — CI/CD com GitHub Actions *(Fase 5)*

### 16.1 Estrutura do workflow

O arquivo `.github/workflows/api-go.yml` (detalhado no PRD 044) segue uma cadeia de dependências:

```
lint → unit-tests → integration-tests → build (+ push GHCR)
```

Cada job só roda se o anterior passou. Isso garante que nunca se faça push de uma imagem que falha no lint ou nos testes.

### 16.2 Cache de módulos Go

```yaml
# Dentro de um job que usa Go:
- uses: actions/setup-go@v5
  with:
    go-version: '1.24'
    cache: true           # cache automático de go modules
    cache-dependency-path: football-api-go/go.sum
```

O cache reduz o tempo de download das dependências de ~45s para ~3s nas execuções subsequentes.

### 16.3 Service containers para integração

```yaml
# No job integration-tests:
services:
  postgres:
    image: postgres:16-alpine
    env:
      POSTGRES_DB: rachao_test
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    options: >-
      --health-cmd pg_isready
      --health-interval 10s
      --health-timeout 5s
      --health-retries 5
```

O `options: --health-cmd pg_isready` garante que o step de testes só começa após o PostgreSQL estar pronto para receber conexões.

### 16.4 Dockerfile multi-stage

```dockerfile
# football-api-go: Dockerfile

# Stage dev — com air para live-reload
FROM golang:1.24-alpine AS dev
RUN go install github.com/cosmtrek/air@latest
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
CMD ["air"]

# Stage builder — compila o binário estático
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \       # remove símbolos de debug → binário menor
    -o /app/server \
    ./cmd/server/main.go

# Stage production — imagem mínima (~20MB total)
FROM scratch AS production
COPY --from=builder /app/server /server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8080
USER 65534:65534          # nobody:nogroup — usuário não-root
ENTRYPOINT ["/server"]
```

**`CGO_ENABLED=0`:** desabilita o Cgo (bindings C). Permite compilar um binário estático que roda na imagem `scratch` (sem nenhuma dependência de SO). Resultado: imagem Docker de ~20–25MB.

---

## Parte III — Referência

---

<a name="apendice-a"></a>
## Apêndice A — Go vs Python/FastAPI: tabela de equivalências

| Conceito | Python / FastAPI | Go / Chi + sqlc |
|---|---|---|
| Roteamento | `@app.get("/groups")` | `r.Get("/groups", h.ListGroups)` |
| Path params | `id: int = Path(...)` | `chi.URLParam(r, "id")` |
| Body parsing | `req: CreateGroupRequest` (Pydantic) | `json.NewDecoder(r.Body).Decode(&req)` |
| Validação | Pydantic validators | Manual + `apierror.Unprocessable()` |
| Autenticação | `player = Depends(CurrentPlayer)` | `middleware.PlayerFromCtx(r.Context())` |
| Erro 404 | `raise NotFoundError("msg")` | `return nil, apierror.NotFound("msg")` |
| Banco | `await db.execute(query)` | `q.GetPlayer(ctx, id)` (sqlc gerado) |
| Transação | `async with db.begin()` | `tx, _ := pool.Begin(ctx); defer tx.Rollback(ctx)` |
| ORM | SQLAlchemy models | sqlc gera structs a partir de SQL |
| Migrations | `migrations/NNN_desc.sql` (mesma) | golang-migrate aponta para o mesmo diretório |
| Testes | `pytest` + `httpx.AsyncClient` | `go test` + `httptest.NewRecorder` |
| Mock de deps | `monkeypatch` / `pytest-mock` | Interface + struct que implementa a interface |
| Settings | `pydantic.BaseSettings` | `envconfig.Process` |
| Startup/lifecycle | `@app.on_event("startup")` | `func main()` + `defer pool.Close()` |
| Erro de tipo | Em runtime (`AttributeError`) | Em compilação (`go build`) |
| Concorrência | `async def` + `await` | Goroutines + `context.Context` |
| SSE | `EventSourceResponse` (sse-starlette) | `http.Flusher` + goroutine + channel |
| Deploy | `uvicorn app.main:app` | Binário estático `./server` |

---

<a name="apendice-b"></a>
## Apêndice B — Padrões idiomáticos Go usados no projeto

### B.1 Accept interfaces, return structs

```go
// ✅ Parâmetros de função aceitam interfaces (flexibilidade, testabilidade)
func NewGroupHandler(svc GroupService) *groupHandler { ... }

// ✅ Construtores retornam structs concretas (clareza de tipo)
func NewAuthService(...) *authService { ... }

// ❌ Evitar retornar interfaces (dificulta type assertion)
func NewAuthService(...) AuthService { ... } // não necessário
```

### B.2 Table-driven tests

```go
func TestTeamBuilder(t *testing.T) {
    tests := []struct {
        name      string
        players   int
        numTeams  int
        wantErr   bool
    }{
        {"2 teams, 8 players", 8, 2, false},
        {"3 teams, 10 players", 10, 3, false},
        {"invalid: 1 team", 5, 1, true},
        {"invalid: more teams than players", 2, 3, true},
    }

    svc := services.NewTeamBuilderService()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            players := makePlayers(tt.players)
            _, err := svc.DrawTeams(players, tt.numTeams)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### B.3 Funções opcionais com functional options

```go
// Padrão para configurar structs com muitos parâmetros opcionais:

type serverOptions struct {
    readTimeout  time.Duration
    writeTimeout time.Duration
    maxBodySize  int64
}

type Option func(*serverOptions)

func WithReadTimeout(d time.Duration) Option {
    return func(o *serverOptions) { o.readTimeout = d }
}

func NewServer(handler http.Handler, opts ...Option) *http.Server {
    o := &serverOptions{
        readTimeout:  15 * time.Second, // defaults
        writeTimeout: 60 * time.Second,
        maxBodySize:  5 << 20, // 5MB
    }
    for _, opt := range opts {
        opt(o)
    }
    return &http.Server{
        Handler:      handler,
        ReadTimeout:  o.readTimeout,
        WriteTimeout: o.writeTimeout,
    }
}
```

### B.4 `errors.Is` vs `errors.As`

```go
// errors.Is: verifica se o erro É um valor específico (sentinel errors)
if errors.Is(err, pgx.ErrNoRows) { ... }

// errors.As: verifica se o erro É DO TIPO, extraindo o valor
var apiErr *apierror.APIError
if errors.As(err, &apiErr) {
    // apiErr agora tem Code e Detail populados
    renderJSON(w, apiErr.Code, apiErr)
}
```

### B.5 `defer` para recursos

```go
// Padrão: abrir + defer fechar/liberar imediatamente após
file, err := os.Open(path)
if err != nil { return err }
defer file.Close() // garante fechamento mesmo em caso de panic

rows, err := db.Query(ctx, query)
if err != nil { return err }
defer rows.Close()
```

---

<a name="apendice-c"></a>
## Apêndice C — Glossário

| Termo | Definição |
|---|---|
| **goroutine** | Função executada concorrentemente. Gerenciada pelo runtime Go, não pelo OS. Custo ~2KB de stack. |
| **channel** | Canal tipado para comunicação entre goroutines. `make(chan T)`. |
| **interface** | Conjunto de assinaturas de métodos. Qualquer tipo que implementa os métodos satisfaz a interface — sem `implements`. |
| **embedding** | Inclusão de um tipo dentro de outro struct. Os campos e métodos do tipo embutido ficam disponíveis diretamente. |
| **nil** | Zero value de ponteiros, interfaces, maps, slices, channels e funções. Diferente de zero value de tipos concretos. |
| **defer** | Adia a execução de uma função para quando o escopo retornar. Executado em LIFO (último definido, primeiro executado). |
| **context.Context** | Mecanismo de cancelamento e propagação de valores em chamadas de funções. Sempre primeiro parâmetro em funções com I/O. |
| **pgxpool** | Pool de conexões PostgreSQL do pgx/v5. Gerencia múltiplas conexões simultâneas com o banco. |
| **sqlc** | Ferramenta de geração de código. Transforma SQL com annotations em funções Go tipadas. |
| **golangci-lint** | Orquestrador de linters Go. Roda múltiplos analisadores estáticos em paralelo. |
| **httptest.NewRecorder** | Implementação de `http.ResponseWriter` para testes. Captura status code, headers e body sem servidor real. |
| **scratch** | Imagem Docker vazia. Usada como base para binários Go estáticos — resulta em imagens de 20–30MB. |
| **CGO_ENABLED=0** | Desabilita Cgo na compilação. Permite criar binário estático compatível com imagem `scratch`. |
| **swaggo/swag** | Ferramenta que extrai annotations `// @Summary` dos handlers Go e gera `openapi.yaml`. |
| **Mintlify** | Plataforma de documentação técnica com MDX, playground OpenAPI interativo e geração de `/llms.txt`. |
| **api_v2_enabled** | Flag por jogador que controla acesso à `/api/v2`. Permite rollout por amostragem controlado pelo super admin. |

---

*Guia mantido em sincronia com o PRD 044. Ao implementar cada fase, revisite o capítulo correspondente para consultar padrões e exemplos.*
