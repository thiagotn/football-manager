# Git Hooks

Git hooks automatizados para validação antes de commits e pushes.

## Hooks disponíveis

### `pre-push`
Valida vulnerabilidades de segurança nas dependências Python antes de fazer push.

**O que faz:**
- Executa `pip-audit` para detectar dependências com vulnerabilidades conhecidas
- Bloqueia o push se houver vulnerabilidades
- Fornece instruções para corrigir

**Quando roda:** `git push`

## Configuração

O arquivo `.gitconfig` já está configurado para usar este diretório:
```bash
git config core.hooksPath .githooks
```

Se precisar reconfigurar após clonar o repositório:
```bash
git config core.hooksPath .githooks
```

## Testando manualmente

```bash
# Testar o pre-push hook
cd football-api && poetry run pip-audit
```

## Desabilitando temporariamente

Se precisar fazer push sem passar pelo hook:
```bash
git push --no-verify
```

⚠️ **Nota:** Use apenas para situações excepcionais. Os hooks existem para prevenir que problemas de segurança cheguem ao repositório.

## Adicionando novos hooks

1. Criar novo arquivo em `.githooks/` com o nome do hook (ex: `pre-commit`)
2. Tornar executável: `chmod +x .githooks/pre-commit`
3. O git usará automaticamente (via `core.hooksPath`)

## Referência

- [Git Hooks Documentation](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks)
- [Conventional Hooks Standard](https://conventionalcommits.org/)
