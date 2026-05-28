# PRD 048 — Guia de Estudo Go: construindo a `rachao-cli`

| Campo | Valor |
|---|---|
| **Versão** | 1.0 |
| **Status** | 📖 Documento de referência |
| **Autor** | thiagotn |
| **Data** | 2026-05-27 |
| **Referência** | [PRD 045 — Guia Go: football-api-go](045-guia-go-football-api.md) |
| **Continuidade** | Complemento pedagógico de PRD 045 — explora CLI em Go vs servidor HTTP |

---

> **Como usar este guia**
>
> Este documento é um livro de estudo estruturado em torno da implementação prática da `rachao-cli`, uma CLI para o rachao.app em Go. Ele complementa PRD 045 (que cobre o servidor HTTP) focando em aspectos únicos de CLIs: cobra, persistência local, formatação de output, testes de command. Leia na ordem para ter a progressão correta, mas cada capítulo também funciona como referência isolada.
>
> **Convenção de exemplos:** blocos `// rachao-cli: caminho/do/arquivo` indicam o arquivo exato onde o padrão aparecerá na implementação.

---

## Sumário

**Parte I — CLI em Go: fundamentos**

- **Cap. 1** — [Por que uma CLI em Go para o rachao.app](#cap-1)
- **Cap. 2** — [Cobra: framework de CLI](#cap-2)

**Parte II — Construindo a `rachao-cli` (5 casos de uso v1)**

- **Cap. 3** — [Estrutura de projeto e bootstrap](#cap-3) *(Fase 1)*
- **Cap. 4** — [Cliente HTTP: interface e implementação](#cap-4) *(Fase 1)*
- **Cap. 5** — [Persistência de configuração e tokens](#cap-5) *(Fase 1)*
- **Cap. 6** — [UC1: `rachao login`](#cap-6) *(Fase 2)*
- **Cap. 7** — [UC2: `rachao me`](#cap-7) *(Fase 2)*
- **Cap. 8** — [UC3: `rachao grupos`](#cap-8) *(Fase 2)*
- **Cap. 9** — [UC4: `rachao ranking`](#cap-9) *(Fase 2)*
- **Cap. 10** — [UC5: `rachao partidas`](#cap-10) *(Fase 2)*
- **Cap. 11** — [Formatação de output: tabelas e cores](#cap-11) *(Fase 1)*
- **Cap. 12** — [Testes de CLI](#cap-12) *(Fase 3)*

**Parte III — Referência e Roadmap**

- [Apêndice A — Árvore de comandos v1](#apendice-a)
- [Apêndice B — Roadmap v2+](#apendice-b)
- [Apêndice C — Comparação: Go vs Python (click/typer)](#apendice-c)

---

## Parte I — CLI em Go: Fundamentos

---

<a name="cap-1"></a>
## Capítulo 1 — Por que uma CLI em Go para o rachao.app

### 1.1 O problema que uma CLI resolve

A football-api-go existe — é um servidor HTTP robusto. Mas a forma mais fácil para um jogador **verificar seu ranking**, **confirmar presença em um rachão**, ou **listar suas estatísticas** é ainda via frontend web (SvelteKit) ou chamadas manuais de curl. Uma CLI muda isso:

- **Produtividade**: `rachao me` vs abrir o browser
- **Automação**: scripts shell podem integrar dados do rachao.app em dashboards ou notificações
- **Offline-friendliness**: a CLI cacheia tokens e dados locais
- **UX de desenvolvedor**: quem integra a API pode testar endpoints com a CLI antes de usar curl/Postman

### 1.2 Por que Go é uma boa escolha para uma CLI

| Aspecto | Go | Python (typer) |
|--------|----|----|
| **Tamanho do binário** | ~10MB (rachao-cli single binary) | ~50MB+ (Python + venv + deps) |
| **Velocidade de startup** | <10ms | ~500ms (Python interpreter) |
| **Distribuição** | Um arquivo executável (cross-platform) | Requer pip/venv, ou PyInstaller (complexo) |
| **Tipagem estática** | Nativa, compile-time | Opcional, runtime via mypy |
| **Concorrência nativa** | Goroutines, channels | asyncio, threads (GIL) |

Para uma CLI que será usada muitas vezes por dia, startup rápido é crítico. Uma goroutine leve permite fazer requisições paralelas (ex: carregar ranking + meu perfil simultaneamente).

### 1.3 Conceitos únicos de uma CLI vs servidor HTTP

| Conceito | Servidor HTTP | CLI |
|----------|----|----|
| **Persistência** | Banco de dados (PostgreSQL) | Arquivo local (`~/.rachao/config.json`) |
| **Autenticação** | Middleware HTTP | Função que carrega token do arquivo |
| **Saída** | JSON ou HTML (para client) | Texto formatado para terminal (tabelas, cores) |
| **Input** | Query params, JSON body | `os.Args`, flags, prompts interativos |
| **Testes** | `httptest.Server` | `cobra.Command.Execute()` com pipe de stdout |

### 1.4 Casos de uso que motivam a CLI

| # | Caso de uso | Comando | Conceito Go |
|---|---|---|---|
| 1 | Fazer login e salvar credencial | `rachao login` | `bufio` (prompt), `json`, `os.UserHomeDir` |
| 2 | Ver meu perfil e estatísticas | `rachao me` | Interface `APIClient`, struct tags JSON |
| 3 | Listar grupos e detalhes de membros | `rachao grupos list/detalhes` | Subcomandos cobra, slices, `tablewriter` |
| 4 | Ver ranking global (sem login) | `rachao ranking` | Flags cobra, `url.Values`, endpoint público |
| 5 | Listar partidas de um grupo | `rachao partidas list --grupo <id>` | Flags obrigatórias, formatação de tempo |

---

<a name="cap-2"></a>
## Capítulo 2 — Cobra: Framework de CLI

### 2.1 O que é Cobra

Cobra é um framework para construir CLIs profissionais em Go. Oferece:
- **Commands e Subcommands**: `rachao grupos` → `list`, `detalhes`
- **Flags (persistent e locais)**: `--json`, `--help`, `--version`
- **Argument parsing automático**: `rachao partidas 123` extrai `123` como argumento
- **Help gerado automaticamente**: `rachao --help`
- **Shell completions**: `rachao completion bash`

Equivalente em Python: `click`, `typer`, `argparse`.

### 2.2 Estrutura padrão de um Cobra command

```go
// rachao-cli: internal/commands/login.go

package commands

import (
    "fmt"
    "github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
    Use:   "login",
    Short: "Authenticate with WhatsApp number and password",
    Long:  "Interactively login, storing JWT tokens locally",
    RunE:  runLogin,  // RunE = Run Error — função que retorna error
}

func runLogin(cmd *cobra.Command, args []string) error {
    // lógica aqui
    fmt.Println("Logged in!")
    return nil
}

func init() {
    // Registra este command como filho do root
    rootCmd.AddCommand(loginCmd)
    
    // Flags específicas deste command
    loginCmd.Flags().StringVar(&password, "password", "", "Password (if not provided, will prompt)")
}
```

### 2.3 Root command e estrutura de tree

```go
// rachao-cli: cmd/rachao/main.go

package main

import (
    "github.com/thiagotn/football-manager/rachao-cli/internal/commands"
)

func main() {
    commands.Execute()
}
```

```go
// rachao-cli: internal/commands/root.go

package commands

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
    Use:     "rachao",
    Short:   "CLI for rachao.app",
    Long:    "Manage matches, groups, and stats from the terminal",
    Version: "0.1.0",
}

func Execute() error {
    return rootCmd.Execute()
}
```

### 2.4 Subcommands (grupos list, grupos detalhes)

```go
// rachao-cli: internal/commands/groups.go

package commands

var groupsCmd = &cobra.Command{
    Use:   "grupos",
    Short: "Manage groups",
}

var groupsListCmd = &cobra.Command{
    Use:   "list",
    Short: "List your groups",
    RunE:  runGroupsList,
}

var groupsDetailsCmd = &cobra.Command{
    Use:   "detalhes [group-id]",
    Short: "Show group details and members",
    Args:  cobra.ExactArgs(1),  // requer exatamente 1 argumento
    RunE:  runGroupsDetails,
}

func init() {
    // Registra subcommands
    groupsCmd.AddCommand(groupsListCmd)
    groupsCmd.AddCommand(groupsDetailsCmd)
    rootCmd.AddCommand(groupsCmd)
}
```

### 2.5 Flags: tipos e uso

```go
// Local flag — só vale para este command
var (
    formato   string
    verbose   bool
    matchYear int
)

matchesCmd.Flags().StringVar(&formato, "formato", "table", "Output format: table, json, csv")
matchesCmd.Flags().BoolVar(&verbose, "v", false, "Verbose output")

// Persistent flag — vale para este command E todos os filhos
rootCmd.PersistentFlags().StringVar(&apiURL, "api", "http://localhost:8080/api/v2", "API base URL")
```

---

## Parte II — Construindo a `rachao-cli`

---

<a name="cap-3"></a>
## Capítulo 3 — Estrutura de projeto e bootstrap

### 3.1 Layout de diretórios

```
rachao-cli/
├── cmd/
│   └── rachao/
│       └── main.go              # entrypoint único
├── internal/
│   ├── api/
│   │   ├── client.go            # interface APIClient + implementação
│   │   └── types.go             # structs de resposta da API
│   ├── config/
│   │   └── config.go            # carregar/salvar ~/.rachao/config.json
│   ├── commands/
│   │   ├── root.go              # root command
│   │   ├── login.go             # UC1
│   │   ├── me.go                # UC2
│   │   ├── groups.go            # UC3
│   │   ├── ranking.go           # UC4
│   │   ├── matches.go           # UC5
│   │   └── helpers.go           # funções compartilhadas (print table, etc)
│   └── ui/
│       ├── table.go             # tablewriter helpers
│       └── colors.go            # fatih/color helpers
├── tests/
│   └── commands/
│       ├── login_test.go
│       ├── me_test.go
│       └── groups_test.go
├── go.mod
├── go.sum
├── Makefile
└── .gitignore
```

### 3.2 `go.mod` inicial

```
module github.com/thiagotn/football-manager/rachao-cli

go 1.24

require (
    github.com/spf13/cobra v1.8.1
    github.com/olekukonko/tablewriter v0.0.5
    github.com/fatih/color v1.17.0
    golang.org/x/term v0.22.0
)
```

### 3.3 Bootstrapping: `cmd/rachao/main.go`

```go
// rachao-cli: cmd/rachao/main.go

package main

import (
    "fmt"
    "os"
    "github.com/thiagotn/football-manager/rachao-cli/internal/commands"
)

func main() {
    if err := commands.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, "Error:", err)
        os.Exit(1)
    }
}
```

### 3.4 Root command (`internal/commands/root.go`)

```go
// rachao-cli: internal/commands/root.go

package commands

import (
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:     "rachao",
    Short:   "CLI for rachao.app",
    Long:    "Manage matches, groups, and stats from the terminal.",
    Version: "0.1.0",
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    // Persistent flags que todos os commands herdam
    rootCmd.PersistentFlags().StringVar(
        &cfgAPIURL, "api", "http://localhost:8080/api/v2",
        "API base URL",
    )
}

var cfgAPIURL string // será compartilhado com commands filhos
```

---

<a name="cap-4"></a>
## Capítulo 4 — Cliente HTTP: interface e implementação

### 4.1 Filosofia: interface primeiro

Assim como PRD 045 Cap. 11 ensinava interfaces para testabilidade em handlers, aqui fazemos o mesmo: definir uma interface `APIClient` que pode ser mockada nos testes.

```go
// rachao-cli: internal/api/client.go

package api

import (
    "context"
    "io"
)

// APIClient define o contrato para chamadas à API
type APIClient interface {
    // Auth
    Login(ctx context.Context, whatsapp, password string) (*TokenResponse, error)
    GetMe(ctx context.Context) (*PlayerResponse, error)
    
    // Groups
    ListGroups(ctx context.Context) ([]GroupResponse, error)
    GetGroup(ctx context.Context, groupID string) (*GroupDetailResponse, error)
    
    // Matches
    ListMatches(ctx context.Context, groupID string) ([]MatchResponse, error)
    
    // Ranking
    GetRanking(ctx context.Context, rankType string, year, month *int) ([]RankingEntry, error)
}

// Implementação real com http.Client
type httpClient struct {
    baseURL string
    token   string
    client  *http.Client
}

func NewHTTPClient(baseURL, token string) APIClient {
    return &httpClient{
        baseURL: baseURL,
        token:   token,
        client:  &http.Client{Timeout: 10 * time.Second},
    }
}

// Login implementa APIClient.Login
func (c *httpClient) Login(ctx context.Context, whatsapp, password string) (*TokenResponse, error) {
    req := LoginRequest{WhatsApp: whatsapp, Password: password}
    var resp TokenResponse
    
    if err := c.do(ctx, "POST", "/auth/login", req, &resp); err != nil {
        return nil, err
    }
    return &resp, nil
}

// Helper: do() encapsula a lógica comum de request/response
func (c *httpClient) do(ctx context.Context, method, path string, body, respPtr interface{}) error {
    // monta URL, marshala body em JSON, faz request, unmarshal resposta
    // ...
}
```

### 4.2 Tipos de resposta (schemas)

```go
// rachao-cli: internal/api/types.go

package api

type TokenResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    TokenType    string `json:"token_type"`
    PlayerID     string `json:"player_id"`
    Name         string `json:"name"`
    Nickname     *string `json:"nickname"`
    Role         string `json:"role"`
    AvatarURL    *string `json:"avatar_url"`
    ChatEnabled  bool   `json:"chat_enabled"`
}

type PlayerResponse struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    WhatsApp string `json:"whatsapp"`
    Role     string `json:"role"`
}

type GroupResponse struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    CreatedBy   string `json:"created_by"`
    IsPublic    bool   `json:"is_public"`
    MemberCount int    `json:"member_count"`
}

type MatchResponse struct {
    ID            string `json:"id"`
    GroupID       string `json:"group_id"`
    Date          string `json:"match_date"`
    StartTime     string `json:"start_time"`
    Location      string `json:"location"`
    MaxPlayers    int    `json:"max_players"`
    ConfirmedCount int  `json:"confirmed_count"`
}

type RankingEntry struct {
    Position int    `json:"position"`
    Name     string `json:"name"`
    Nickname *string `json:"nickname"`
    Goals    int    `json:"goals"`
    Assists  int    `json:"assists"`
    Matches  int    `json:"matches"`
}
```

### 4.3 Context e timeouts

```go
// Todos os métodos do APIClient recebem context para:
// 1. Propagação de cancelamento (ex: user pressiona Ctrl+C)
// 2. Timeout automático (ex: 10s para resposta do servidor)

func (c *httpClient) do(ctx context.Context, method, path string, body, respPtr interface{}) error {
    // Criar request com contexto
    req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }
    
    // Autorização
    if c.token != "" {
        req.Header.Set("Authorization", "Bearer "+c.token)
    }
    req.Header.Set("Content-Type", "application/json")
    
    // Executar com timeout implícito (do http.Client)
    resp, err := c.client.Do(req)
    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()
    
    // Unmarshal resposta
    if err := json.NewDecoder(resp.Body).Decode(respPtr); err != nil {
        return fmt.Errorf("parse response: %w", err)
    }
    return nil
}
```

---

<a name="cap-5"></a>
## Capítulo 5 — Persistência de configuração e tokens

### 5.1 Estrutura do arquivo de configuração

```json
// ~/.rachao/config.json

{
  "api_url": "http://localhost:8080/api/v2",
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "player_id": "550e8400-e29b-41d4-a716-446655440000",
  "player_name": "João Silva",
  "token_expires_at": 1715612400
}
```

### 5.2 Gerenciar configuração com `os.UserHomeDir()`

```go
// rachao-cli: internal/config/config.go

package config

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"
)

type Config struct {
    APIURL         string `json:"api_url"`
    AccessToken    string `json:"access_token"`
    RefreshToken   string `json:"refresh_token"`
    PlayerID       string `json:"player_id"`
    PlayerName     string `json:"player_name"`
    TokenExpiresAt int64  `json:"token_expires_at"`
}

func configDir() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", fmt.Errorf("get home dir: %w", err)
    }
    return filepath.Join(home, ".rachao"), nil
}

func configPath() (string, error) {
    dir, err := configDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dir, "config.json"), nil
}

// Load lê a configuração do arquivo
func Load() (*Config, error) {
    path, err := configPath()
    if err != nil {
        return nil, err
    }
    
    // Se arquivo não existe, retorna erro específico
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return nil, fmt.Errorf("not logged in: run 'rachao login' first")
    }
    
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }
    
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }
    
    return &cfg, nil
}

// Save escreve a configuração no arquivo
func (c *Config) Save() error {
    path, err := configPath()
    if err != nil {
        return err
    }
    
    // Garantir que o diretório existe
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0700); err != nil {
        return fmt.Errorf("create config dir: %w", err)
    }
    
    data, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return fmt.Errorf("marshal config: %w", err)
    }
    
    if err := os.WriteFile(path, data, 0600); err != nil {
        return fmt.Errorf("write config: %w", err)
    }
    
    return nil
}

// IsTokenExpired verifica se o token expirou
func (c *Config) IsTokenExpired() bool {
    return time.Now().Unix() >= c.TokenExpiresAt
}

// Delete remove a configuração (logout)
func Delete() error {
    path, err := configPath()
    if err != nil {
        return err
    }
    return os.Remove(path)
}
```

### 5.3 Integração com commands

```go
// rachao-cli: internal/commands/helpers.go

package commands

import (
    "fmt"
    "github.com/thiagotn/football-manager/rachao-cli/internal/api"
    "github.com/thiagotn/football-manager/rachao-cli/internal/config"
)

// getClient retorna um APIClient autenticado ou erro
func getClient() (api.APIClient, error) {
    cfg, err := config.Load()
    if err != nil {
        return nil, fmt.Errorf("not logged in: %w", err)
    }
    
    return api.NewHTTPClient(cfg.APIURL, cfg.AccessToken), nil
}

// getClientOrPublic retorna um APIClient que pode estar autenticado ou anônimo
func getClientOrPublic() api.APIClient {
    cfg, err := config.Load()
    token := ""
    url := "http://localhost:8080/api/v2"
    
    if err == nil {
        token = cfg.AccessToken
        url = cfg.APIURL
    }
    
    return api.NewHTTPClient(url, token)
}
```

---

<a name="cap-6"></a>
## Capítulo 6 — UC1: `rachao login`

### 6.1 Fluxo do comando

```
rachao login
    ↓
Prompt: "WhatsApp (+55...): "
    ↓
Prompt: "Password: " [ocultada com term.ReadPassword]
    ↓
POST /api/v2/auth/login
    ↓
Salvar tokens em ~/.rachao/config.json
    ↓
"✓ Logged in as João Silva"
```

### 6.2 Implementação completa

```go
// rachao-cli: internal/commands/login.go

package commands

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "strings"
    
    "github.com/spf13/cobra"
    "golang.org/x/term"
    
    "github.com/thiagotn/football-manager/rachao-cli/internal/api"
    "github.com/thiagotn/football-manager/rachao-cli/internal/config"
)

var loginCmd = &cobra.Command{
    Use:   "login",
    Short: "Authenticate with WhatsApp and password",
    Long:  "Interactively login to rachao.app and store credentials locally.",
    RunE:  runLogin,
}

func runLogin(cmd *cobra.Command, args []string) error {
    // 1. Prompt para WhatsApp
    reader := bufio.NewReader(os.Stdin)
    fmt.Print("WhatsApp (+55...): ")
    whatsapp, err := reader.ReadString('\n')
    if err != nil {
        return fmt.Errorf("read whatsapp: %w", err)
    }
    whatsapp = strings.TrimSpace(whatsapp)
    
    // 2. Prompt para password (sem echo)
    fmt.Print("Password: ")
    passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
    if err != nil {
        return fmt.Errorf("read password: %w", err)
    }
    fmt.Println() // nova linha após password
    password := string(passwordBytes)
    
    // 3. Chamar API
    client := api.NewHTTPClient(cfgAPIURL, "")
    resp, err := client.Login(context.Background(), whatsapp, password)
    if err != nil {
        return fmt.Errorf("login failed: %w", err)
    }
    
    // 4. Salvar config
    cfg := &config.Config{
        APIURL:       cfgAPIURL,
        AccessToken:  resp.AccessToken,
        RefreshToken: resp.RefreshToken,
        PlayerID:     resp.PlayerID,
        PlayerName:   resp.Name,
        TokenExpiresAt: 0, // TODO: calcular da resposta (claims exp)
    }
    if err := cfg.Save(); err != nil {
        return fmt.Errorf("save config: %w", err)
    }
    
    // 5. Feedback
    fmt.Printf("✓ Logged in as %s\n", resp.Name)
    return nil
}

func init() {
    rootCmd.AddCommand(loginCmd)
}
```

### 6.3 Conceitos Go ilustrados

| Conceito | Uso | Arquivo |
|----------|-----|---------|
| `bufio.Reader` | Ler input do usuário linha a linha | `login.go` |
| `term.ReadPassword` | Ler senha sem exibir | `login.go` (requer `golang.org/x/term`) |
| `strings.TrimSpace` | Remove espaçamento em branco | `login.go` |
| `context.Background()` | Contexto que nunca expira | `login.go` |
| `&config.Config{}` | Struct init com zero values | `login.go` |

---

<a name="cap-7"></a>
## Capítulo 7 — UC2: `rachao me`

### 7.1 Fluxo do comando

```
rachao me
    ↓
GET /api/v2/auth/me (com Bearer token)
    ↓
Formatar resposta em seções:
    - Dados pessoais (nome, WhatsApp, role)
    - Estatísticas (partidas, gols, assistências)
    ↓
Exibir em terminal com cores
```

### 7.2 Implementação

```go
// rachao-cli: internal/commands/me.go

package commands

import (
    "context"
    "fmt"
    
    "github.com/spf13/cobra"
    
    "github.com/thiagotn/football-manager/rachao-cli/internal/ui"
)

var meCmd = &cobra.Command{
    Use:   "me",
    Short: "Show my profile and statistics",
    Long:  "Display authenticated player profile and match statistics.",
    RunE:  runMe,
}

func runMe(cmd *cobra.Command, args []string) error {
    // 1. Carregar client autenticado
    client, err := getClient()
    if err != nil {
        return err
    }
    
    // 2. Chamar API
    player, err := client.GetMe(context.Background())
    if err != nil {
        return fmt.Errorf("fetch profile: %w", err)
    }
    
    // 3. Formatar e exibir
    ui.PrintPlayerProfile(player)
    return nil
}

func init() {
    rootCmd.AddCommand(meCmd)
}
```

### 7.3 Formatação com cores (Cap. 11 preview)

```go
// rachao-cli: internal/ui/colors.go

package ui

import (
    "fmt"
    "github.com/fatih/color"
    "github.com/thiagotn/football-manager/rachao-cli/internal/api"
)

var (
    bold    = color.New(color.Bold).SprintFunc()
    cyan    = color.New(color.FgCyan).SprintFunc()
    green   = color.New(color.FgGreen).SprintFunc()
    yellow  = color.New(color.FgYellow).SprintFunc()
)

func PrintPlayerProfile(p *api.PlayerResponse) {
    fmt.Printf("\n%s\n", bold("=== MINHA CONTA ==="))
    fmt.Printf("%s %s\n", cyan("Nome:"), p.Name)
    if p.Nickname != nil && *p.Nickname != "" {
        fmt.Printf("%s %s\n", cyan("Apelido:"), *p.Nickname)
    }
    fmt.Printf("%s %s\n", cyan("WhatsApp:"), p.WhatsApp)
    fmt.Printf("%s %s\n", cyan("Função:"), roleEmoji(p.Role)+" "+p.Role)
    fmt.Println()
}

func roleEmoji(role string) string {
    switch role {
    case "admin":
        return "👑"
    default:
        return "⚽"
    }
}
```

---

<a name="cap-8"></a>
## Capítulo 8 — UC3: `rachao grupos`

### 8.1 Fluxo e subcomandos

```
rachao grupos
    ├── list        # listar meus grupos
    └── detalhes <id>  # detalhes + membros de um grupo
```

### 8.2 Implementação

```go
// rachao-cli: internal/commands/groups.go

package commands

import (
    "context"
    "fmt"
    "os"
    "strconv"
    
    "github.com/spf13/cobra"
    
    "github.com/thiagotn/football-manager/rachao-cli/internal/ui"
)

var groupsCmd = &cobra.Command{
    Use:   "grupos",
    Short: "Manage groups",
}

var groupsListCmd = &cobra.Command{
    Use:   "list",
    Short: "List your groups",
    RunE:  runGroupsList,
}

var groupsDetailsCmd = &cobra.Command{
    Use:   "detalhes [group-id]",
    Short: "Show group details and members",
    Args:  cobra.ExactArgs(1),
    RunE:  runGroupsDetails,
}

func runGroupsList(cmd *cobra.Command, args []string) error {
    client, err := getClient()
    if err != nil {
        return err
    }
    
    groups, err := client.ListGroups(context.Background())
    if err != nil {
        return fmt.Errorf("fetch groups: %w", err)
    }
    
    if len(groups) == 0 {
        fmt.Println("No groups found. Create one with 'rachao grupos create'.")
        return nil
    }
    
    ui.PrintGroupsTable(groups)
    return nil
}

func runGroupsDetails(cmd *cobra.Command, args []string) error {
    client, err := getClient()
    if err != nil {
        return err
    }
    
    groupID := args[0]
    
    group, err := client.GetGroup(context.Background(), groupID)
    if err != nil {
        return fmt.Errorf("fetch group: %w", err)
    }
    
    ui.PrintGroupDetail(group)
    return nil
}

func init() {
    groupsCmd.AddCommand(groupsListCmd)
    groupsCmd.AddCommand(groupsDetailsCmd)
    rootCmd.AddCommand(groupsCmd)
}
```

### 8.3 Formatação com tablewriter

```go
// rachao-cli: internal/ui/table.go

package ui

import (
    "fmt"
    "os"
    
    "github.com/olekukonko/tablewriter"
    
    "github.com/thiagotn/football-manager/rachao-cli/internal/api"
)

func PrintGroupsTable(groups []api.GroupResponse) {
    fmt.Printf("\n%s\n\n", bold("=== MEUS GRUPOS ==="))
    
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"ID", "Nome", "Público", "Membros"})
    table.SetBorder(true)
    table.SetRowLine(true)
    
    for _, g := range groups {
        isPublic := "✓"
        if !g.IsPublic {
            isPublic = "-"
        }
        table.Append([]string{
            g.ID[:8] + "...",
            g.Name,
            isPublic,
            fmt.Sprintf("%d", g.MemberCount),
        })
    }
    
    table.Render()
    fmt.Println()
}

func PrintGroupDetail(g *api.GroupDetailResponse) {
    fmt.Printf("\n%s %s\n\n", bold("=== GRUPO:"), g.Name)
    fmt.Printf("%s %s\n", cyan("ID:"), g.ID)
    fmt.Printf("%s %v\n", cyan("Público:"), g.IsPublic)
    fmt.Printf("%s %d membros\n\n", cyan("Membros:"), len(g.Members))
    
    // Tabela de membros
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Nome", "Posição", "Skill", "Role"})
    
    for _, m := range g.Members {
        skill := "-"
        if m.SkillStars > 0 {
            skill = fmt.Sprintf("%.1f", m.SkillStars)
        }
        table.Append([]string{
            m.PlayerName,
            stringOrDash(m.Position),
            skill,
            m.Role,
        })
    }
    
    table.Render()
    fmt.Println()
}

func stringOrDash(s *string) string {
    if s == nil || *s == "" {
        return "-"
    }
    return *s
}
```

---

<a name="cap-9"></a>
## Capítulo 9 — UC4: `rachao ranking`

### 9.1 Fluxo do comando

```
rachao ranking [--tipo top|flop] [--ano 2025] [--mes 5]
    ↓
GET /api/v2/ranking?type=top&year=2025&month=5
    ↓
Exibir ranking em tabela com medalhas (🥇 🥈 🥉)
```

Este é o **primeiro comando público** — não requer login.

### 9.2 Implementação

```go
// rachao-cli: internal/commands/ranking.go

package commands

import (
    "context"
    "fmt"
    "time"
    
    "github.com/spf13/cobra"
    
    "github.com/thiagotn/football-manager/rachao-cli/internal/ui"
)

var (
    rankTipo int = 0  // 0=top, 1=flop
    rankAno  int
    rankMes  int
)

var rankingCmd = &cobra.Command{
    Use:   "ranking",
    Short: "Show global ranking",
    Long:  "Display top/flop players globally (no login required).",
    RunE:  runRanking,
}

func runRanking(cmd *cobra.Command, args []string) error {
    // Client público (sem token necessário)
    client := getClientOrPublic()
    
    // Processar flags
    rankType := "top"
    if rankTipo == 1 {
        rankType = "flop"
    }
    
    var year, month *int
    if rankAno > 0 {
        year = &rankAno
        if rankMes > 0 {
            month = &rankMes
        }
    }
    
    entries, err := client.GetRanking(context.Background(), rankType, year, month)
    if err != nil {
        return fmt.Errorf("fetch ranking: %w", err)
    }
    
    ui.PrintRankingTable(entries, rankType)
    return nil
}

func init() {
    rootCmd.AddCommand(rankingCmd)
    
    now := time.Now()
    rankingCmd.Flags().IntVar(&rankTipo, "tipo", 0, "0=top (default), 1=flop")
    rankingCmd.Flags().IntVar(&rankAno, "ano", now.Year(), "Year filter")
    rankingCmd.Flags().IntVar(&rankMes, "mes", 0, "Month filter (requires year)")
}
```

### 9.3 Formatação com medalhas

```go
// rachao-cli: internal/ui/table.go (adição)

func PrintRankingTable(entries []api.RankingEntry, rankType string) {
    title := "RANKING GERAL"
    if rankType == "flop" {
        title = "PIOR RANKING (FLOP)"
    }
    fmt.Printf("\n%s\n\n", bold("=== "+title+" ==="))
    
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"", "Jogador", "Gols", "Assist.", "Partidas"})
    
    medals := []string{"🥇", "🥈", "🥉"}
    
    for i, e := range entries {
        medal := " "
        if i < len(medals) {
            medal = medals[i]
        }
        
        name := e.Name
        if e.Nickname != nil && *e.Nickname != "" {
            name = *e.Nickname
        }
        
        table.Append([]string{
            medal,
            name,
            fmt.Sprintf("%d", e.Goals),
            fmt.Sprintf("%d", e.Assists),
            fmt.Sprintf("%d", e.Matches),
        })
    }
    
    table.Render()
    fmt.Println()
}
```

---

<a name="cap-10"></a>
## Capítulo 10 — UC5: `rachao partidas`

### 10.1 Fluxo do comando

```
rachao partidas list --grupo <group-id>
    ↓
GET /api/v2/groups/{groupID}/matches
    ↓
Exibir tabela com datas, locais, presença confirmada
```

### 10.2 Implementação

```go
// rachao-cli: internal/commands/matches.go

package commands

import (
    "context"
    "fmt"
    
    "github.com/spf13/cobra"
    
    "github.com/thiagotn/football-manager/rachao-cli/internal/ui"
)

var matchesCmd = &cobra.Command{
    Use:   "partidas",
    Short: "Manage matches",
}

var matchesListCmd = &cobra.Command{
    Use:   "list",
    Short: "List matches in a group",
    RunE:  runMatchesList,
}

var (
    matchGroupID string  // flag --grupo (obrigatória)
)

func runMatchesList(cmd *cobra.Command, args []string) error {
    if matchGroupID == "" {
        return fmt.Errorf("--grupo is required")
    }
    
    client, err := getClient()
    if err != nil {
        return err
    }
    
    matches, err := client.ListMatches(context.Background(), matchGroupID)
    if err != nil {
        return fmt.Errorf("fetch matches: %w", err)
    }
    
    if len(matches) == 0 {
        fmt.Println("No matches scheduled.")
        return nil
    }
    
    ui.PrintMatchesTable(matches)
    return nil
}

func init() {
    matchesCmd.AddCommand(matchesListCmd)
    rootCmd.AddCommand(matchesCmd)
    
    matchesListCmd.Flags().StringVar(&matchGroupID, "grupo", "", "Group ID (required)")
    matchesListCmd.MarkFlagRequired("grupo")
}
```

### 10.3 Formatação de datas e status

```go
// rachao-cli: internal/ui/table.go (adição)

import "time"

func PrintMatchesTable(matches []api.MatchResponse) {
    fmt.Printf("\n%s\n\n", bold("=== PRÓXIMAS PARTIDAS ==="))
    
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Data", "Hora", "Local", "Confirmados", "Status"})
    
    for _, m := range matches {
        date, _ := time.Parse("2006-01-02", m.Date)
        dateStr := date.Format("02/01/2006") // formato pt-BR
        
        status := statusEmoji(m.ConfirmedCount, m.MaxPlayers)
        
        table.Append([]string{
            dateStr,
            m.StartTime,
            m.Location,
            fmt.Sprintf("%d/%d", m.ConfirmedCount, m.MaxPlayers),
            status,
        })
    }
    
    table.Render()
    fmt.Println()
}

func statusEmoji(confirmed, max int) string {
    if confirmed >= max {
        return "✅ Completo"
    } else if confirmed >= max*2/3 {
        return "⏳ Quase lá"
    } else {
        return "🔔 Confirmando"
    }
}
```

---

<a name="cap-11"></a>
## Capítulo 11 — Formatação de output: tabelas e cores

### 11.1 Filosofia de UI em CLI

Diferente de um servidor HTTP que retorna JSON, uma CLI deve apresentar dados de forma **legível ao humano**. Usamos:

- **Cores ANSI** via `fatih/color` — torna output legível sem sobrecarregar
- **Tabelas** via `tablewriter` — alinha colunas, adiciona bordas
- **Emojis** — feedback rápido (✓, ✗, 🥇, ⚽)
- **Formato pt-BR** para datas — não é ambíguo (`02/01/2025` = 2 de janeiro, não fevereiro)

### 11.2 Package `internal/ui/colors.go`

```go
// rachao-cli: internal/ui/colors.go

package ui

import (
    "fmt"
    "github.com/fatih/color"
)

// Funções de estilo reutilizáveis
var (
    bold       = color.New(color.Bold).SprintFunc()
    faint      = color.New(color.Faint).SprintFunc()
    cyan       = color.New(color.FgCyan).SprintFunc()
    green      = color.New(color.FgGreen).SprintFunc()
    yellow     = color.New(color.FgYellow).SprintFunc()
    red        = color.New(color.FgRed).SprintFunc()
    bgGreen    = color.New(color.BgGreen, color.FgBlack).SprintFunc()
    bgRed      = color.New(color.BgRed, color.FgBlack).SprintFunc()
)

// Success imprime mensagem de sucesso em verde
func Success(msg string) {
    fmt.Println(green("✓"), msg)
}

// Error imprime mensagem de erro em vermelho
func Error(msg string) {
    fmt.Println(red("✗"), msg)
}

// InfoSection imprime um título de seção em cyan/bold
func InfoSection(title string) {
    fmt.Printf("\n%s\n\n", bold(cyan("=== "+title+" ===")))
}
```

### 11.3 Package `internal/ui/table.go`

```go
// rachao-cli: internal/ui/table.go

package ui

import (
    "fmt"
    "os"
    
    "github.com/olekukonko/tablewriter"
)

// NewTable retorna uma tabela pré-configurada
func NewTable(headers []string) *tablewriter.Table {
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader(headers)
    table.SetBorder(true)
    table.SetRowLine(true)
    table.SetAlignment(tablewriter.ALIGN_LEFT)
    table.SetCenterSeparator("│")
    table.SetColumnSeparator("│")
    table.SetRowSeparator("─")
    return table
}

// RenderAndSpace renderiza tabela e imprime linha vazia depois
func RenderAndSpace(t *tablewriter.Table) {
    t.Render()
    fmt.Println()
}
```

### 11.4 Formatação de datas

```go
// rachao-cli: internal/ui/fmt.go

package ui

import "time"

// FormatDateBR formata data em formato pt-BR (DD/MM/YYYY)
func FormatDateBR(dateStr string) string {
    t, err := time.Parse("2006-01-02", dateStr)
    if err != nil {
        return dateStr // retorna original se inválido
    }
    return t.Format("02/01/2006")
}

// FormatTimeBR formata horário em formato pt-BR
func FormatTimeBR(t time.Time) string {
    return t.Format("15:04")
}

// DaysUntil retorna "em X dias" ou "hoje" ou "ontem"
func DaysUntil(dateStr string) string {
    t, _ := time.Parse("2006-01-02", dateStr)
    today := time.Now().Truncate(24 * time.Hour)
    target := t.Truncate(24 * time.Hour)
    diff := target.Sub(today).Hours() / 24
    
    switch {
    case diff == 0:
        return "hoje"
    case diff == 1:
        return "amanhã"
    case diff < 0:
        return "passado"
    default:
        return fmt.Sprintf("em %d dias", int(diff))
    }
}
```

---

<a name="cap-12"></a>
## Capítulo 12 — Testes de CLI

### 12.1 Filosofia de testes para CLI

Uma CLI é mais difícil de testar que um handler HTTP. Vamos testar:

1. **Command execution**: o command executa sem erro
2. **Output correctness**: a saída contém as informações esperadas
3. **Error handling**: mensagens de erro corretas

Usamos `cobra.Command.Execute()` com `bytes.Buffer` para capturar stdout.

### 12.2 Mock de `APIClient`

```go
// rachao-cli: tests/commands/helpers_test.go

package commands_test

import (
    "context"
    "github.com/thiagotn/football-manager/rachao-cli/internal/api"
)

// mockAPIClient implementa APIClient para testes
type mockAPIClient struct {
    loginFn      func(ctx context.Context, whatsapp, password string) (*api.TokenResponse, error)
    getMeFn      func(ctx context.Context) (*api.PlayerResponse, error)
    listGroupsFn func(ctx context.Context) ([]api.GroupResponse, error)
    // ... demais métodos
}

func (m *mockAPIClient) Login(ctx context.Context, whatsapp, password string) (*api.TokenResponse, error) {
    if m.loginFn != nil {
        return m.loginFn(ctx, whatsapp, password)
    }
    return nil, nil
}

// ... implementar os outros métodos de APIClient
```

### 12.3 Teste de command: `rachao me`

```go
// rachao-cli: tests/commands/me_test.go

package commands_test

import (
    "bytes"
    "context"
    "testing"
    
    "github.com/thiagotn/football-manager/rachao-cli/internal/api"
    "github.com/thiagotn/football-manager/rachao-cli/internal/commands"
)

func TestMeCommand_Success(t *testing.T) {
    // 1. Mock APIClient
    mockClient := &mockAPIClient{
        getMeFn: func(ctx context.Context) (*api.PlayerResponse, error) {
            return &api.PlayerResponse{
                ID:       "123",
                Name:     "João Silva",
                WhatsApp: "+5511999990000",
                Role:     "player",
            }, nil
        },
    }
    
    // 2. Injetar mock no comando (via função auxiliar)
    // nota: isso requer refatoração para aceitar APIClient como parâmetro
    // ver 12.4 para estratégia de injeção
    
    // 3. Executar comando e capturar output
    cmd := commands.NewMeCommand(mockClient)
    output := &bytes.Buffer{}
    cmd.SetOut(output)
    
    err := cmd.Execute()
    if err != nil {
        t.Fatalf("command failed: %v", err)
    }
    
    // 4. Verificar output
    outputStr := output.String()
    if !contains(outputStr, "João Silva") {
        t.Errorf("output missing player name. got: %s", outputStr)
    }
}

func contains(s, substr string) bool {
    return strings.Contains(s, substr)
}
```

### 12.4 Injeção de dependência para testes

Para testar um command sem chamar a API real, precisamos injetar o mock. Uma estratégia é criar a função que inicializa o command e aceita um `APIClient` opcional:

```go
// rachao-cli: internal/commands/me.go (refatorado para testes)

var (
    meCmd *cobra.Command
    meClient api.APIClient
)

func NewMeCommand(client api.APIClient) *cobra.Command {
    meClient = client // injetar mock
    return meCmd
}

var meCmd = &cobra.Command{
    Use:   "me",
    RunE:  runMe,
}

func runMe(cmd *cobra.Command, args []string) error {
    if meClient == nil {
        // em produção, carregar via getClient()
        var err error
        meClient, err = getClient()
        if err != nil {
            return err
        }
    }
    
    player, err := meClient.GetMe(context.Background())
    if err != nil {
        return err
    }
    
    ui.PrintPlayerProfile(player)
    return nil
}
```

### 12.5 Rodar testes

```bash
cd rachao-cli
go test ./tests/commands/... -v
```

---

## Parte III — Referência e Roadmap

---

<a name="apendice-a"></a>
## Apêndice A — Árvore de comandos v1.0

```
rachao
├── auth
│   └── login                           UC1: autenticar
├── me                                  UC2: perfil pessoal
├── grupos
│   ├── list                           UC3: listar grupos
│   └── detalhes <group-id>            UC3: detalhes de grupo
├── partidas
│   └── list --grupo <id>              UC5: listar partidas
└── ranking [--tipo top|flop] [--ano] [--mes]    UC4: ranking público
```

---

<a name="apendice-b"></a>
## Apêndice B — Roadmap v2+

### v2.0 (próxima fase)

- **Cap. 13** — `rachao partidas create` — criar nova partida interativamente
- **Cap. 14** — `rachao presença` — confirmar/recusar presença
- **Cap. 15** — `rachao times` — ver times sorteados, iniciar sorteio
- **Cap. 16** — `rachao chat` — chat com Claude via SSE no terminal
- **Cap. 17** — `rachao logout` — deletar credenciais locais

### v3.0+

- Configuração interativa (`rachao config`)
- Notificações de novas partidas via polling background
- Histórico local de comandos com atalhos
- Suporte a múltiplos perfis (logar em 2+ contas simultaneamente)
- Tab completion (`rachao completion bash|zsh`)

---

<a name="apendice-c"></a>
## Apêndice C — Comparação: Go vs Python

### CLI Framework: cobra vs click

| Aspecto | Go (cobra) | Python (click) |
|---------|-----------|----------------|
| **Definição** | Struct + function | Decorator + function |
| **Flags** | `.Flags().StringVar(...)` | `@click.option(...)` |
| **Args** | `cobra.ExactArgs(n)` | `@click.argument(...)` |
| **Subcommands** | `.AddCommand()` | `@click.group()` |
| **RunE** | Retorna `error` | Função normal |

**Exemplo Python (click):**
```python
@click.command()
@click.option('--name', prompt='Your name')
def hello(name):
    click.echo(f'Hello {name}')
```

**Equivalente em Go (cobra):**
```go
var nameCmd = &cobra.Command{
    Use: "hello",
    RunE: func(cmd *cobra.Command, args []string) error {
        reader := bufio.NewReader(os.Stdin)
        fmt.Print("Your name: ")
        name, _ := reader.ReadString('\n')
        fmt.Printf("Hello %s\n", name)
        return nil
    },
}
```

### HTTP Client: requests vs net/http

| Aspecto | Go (net/http) | Python (requests) |
|---------|---|---|
| **Fazer request** | `http.NewRequest`, `client.Do` | `requests.get(url, headers=...)` |
| **Timeout** | `client.Timeout` | `requests.get(url, timeout=10)` |
| **Headers** | `req.Header.Set(...)` | `headers={...}` |
| **JSON** | `json.NewDecoder(resp.Body)` | `resp.json()` |
| **Erro de request** | Retorna `error` | Levanta `requests.RequestException` |

### Persistência: json vs json

| Aspecto | Go | Python |
|---------|----|----|
| **Marshal** | `json.Marshal(v)` | `json.dumps(v)` |
| **Unmarshal** | `json.Unmarshal(data, &v)` | `json.loads(data)` |
| **File I/O** | `os.ReadFile` / `os.WriteFile` | `open(f).read()` / `f.write()` |
| **Home dir** | `os.UserHomeDir()` | `pathlib.Path.home()` |

---

## Resumo para o leitor

Ao final deste guia, você terá:

1. **Entendido CLIs em Go**: estrutura, cobra, persistência local
2. **Implementado 5 casos de uso reais**: login, perfil, grupos, ranking, partidas
3. **Explorado conceitos Go únicos**: `bufio`, `term.ReadPassword`, interface `APIClient`, testes com mocks
4. **Construído uma CLI pronta para crescer**: com testes, UI formatada, e arquitetura extensível

A CLI está pronta para ser estendida com novos comandos seguindo o mesmo padrão: interface → implementação → testes → formatação.

---

**Próximos passos:**
- Implementar Cap. 13+ do Roadmap (v2.0)
- Integrar com shell completions
- Publicar binário em releases do GitHub
- Documentar no README com exemplos de uso
