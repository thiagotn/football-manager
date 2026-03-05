# football-e2e

Testes end-to-end do rachao.app usando [Playwright](https://playwright.dev/python/) + [pytest](https://docs.pytest.org/).

---

## Arquitetura

```
football-e2e/
├── conftest.py          # Fixtures globais: login, contextos autenticados
├── pages/               # Page Object Model — abstração das páginas da UI
│   ├── login_page.py
│   ├── dashboard_page.py
│   ├── group_page.py
│   └── match_page.py
└── tests/               # Suites de testes por domínio
    ├── test_auth.py      # Login, logout, redirecionamentos de acesso
    ├── test_groups.py    # Grupos: abas, convite, adicionar membro
    ├── test_matches.py   # Rachões: listagem, status, navegação
    ├── test_players.py   # Jogadores: listagem, edição, busca
    └── test_attendance.py # Partidas: acesso público, presença
```

### Page Object Model (POM)

Cada página da aplicação é representada por uma classe em `pages/`. Os testes interagem somente com os métodos dessas classes, nunca com seletores diretamente. Isso centraliza a manutenção: se um componente mudar, apenas o POM é atualizado.

```python
# uso no teste
gp = GroupPage(page)
gp.tab_members()
gp.invite_button().click()
```

### Fixtures de autenticação

O login é feito uma única vez por sessão (`scope="session"`) e o estado resultante (cookies + localStorage com o token JWT) é salvo em disco e reutilizado por todos os testes que usam `admin_page`. Isso elimina overhead de autenticação repetida.

```
conftest.py
└── admin_storage_state (session)  ← faz login uma vez, salva estado
    └── admin_page (function)      ← novo contexto por teste, carrega estado salvo
```

---

## Pré-requisitos

- Python 3.11+
- Stack local rodando (`docker compose up` em `football-api/`)

---

## Executar localmente

### 1. Instalar dependências

```bash
cd football-e2e
pip install -e .
playwright install chromium
```

### 2. Configurar ambiente

```bash
cp .env.example .env
# edite se necessário (padrão já aponta para localhost:3000)
```

### 3. Rodar os testes

```bash
# todos os testes
pytest tests/ -v

# suite específica
pytest tests/test_auth.py -v

# com screenshots em falha
pytest tests/ -v --screenshot=only-on-failure --output=test-results/

# modo headed (vê o browser abrindo)
pytest tests/ --headed
```

---

## Variáveis de ambiente

| Variável          | Descrição                      | Padrão                    |
|-------------------|--------------------------------|---------------------------|
| `BASE_URL`        | URL base do frontend           | `http://localhost:3000`   |
| `ADMIN_WHATSAPP`  | WhatsApp do usuário admin      | `11999990000`             |
| `ADMIN_PASSWORD`  | Senha do usuário admin         | `admin123`                |

---

## Suites de testes

| Arquivo                | Cenários cobertos                                              |
|------------------------|----------------------------------------------------------------|
| `test_auth.py`         | Login válido/inválido, logout, redirect sem autenticação       |
| `test_groups.py`       | 3 abas do grupo, modal convite, modal adicionar membro        |
| `test_matches.py`      | Aba Próximos/Últimos, navegação para detalhes, status         |
| `test_players.py`      | Listagem, busca, modais editar e resetar senha                |
| `test_attendance.py`   | Acesso público à partida, contagem confirmados, compartilhar  |

---

## CI/CD

Os testes rodam automaticamente no GitHub Actions (`.github/workflows/e2e.yml`) em todo push para `main` que altere `football-frontend/`, `football-api/` ou `football-e2e/`. Também podem ser disparados manualmente via **Actions → E2E Tests → Run workflow**.

Em caso de falha, screenshots são salvas como artifact e ficam disponíveis por 7 dias.
