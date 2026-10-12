# Arquitetura

## Visão geral

Este serviço implementa uma API de carteira e processamento de apostas em Go usando Uber Fx. O domínio é separado de I/O: `internal/domain` contém `Money`, `Wallet` e `WagerTransaction`, enquanto `internal/service` orquestra regras de negócio e persistência. O acesso ao banco e à mensageria são encapsulados em `internal/repository` e `internal/sqs`.

## Autenticação e autorização

A solução usa um IdP externo em um modelo OAuth 2.0/OIDC. O código aceita um token Bearer e, num ambiente local, pode operar em modo stub via `USE_STUB_AUTH=true` para permitir desenvolvimento sem um Keycloak real. Em produção, o token deve ser validado pelo issuer e audience configurados em `OIDC_ISSUER_URL` e `OIDC_AUDIENCE`; a identidade autenticada define o `providerId` permitido e as políticas de acesso usam esse valor no contexto da requisição.

## Representação monetária

A representação de `Money` usa `int64` em unidades mínimas (centavos) e escala fixa de duas casas decimais. A serialização externa usa o contrato `{"amount":"25.00","currency":"BRL"}`. Valores inválidos, NaN, Infinity, notação científica e escala maior que duas casas são rejeitados; a conversão para minor units é feita de forma explícita e sem arredondamento silencioso.

## Transações e concorrência

A carteira é a raiz do agregado financeiro. A atualização do saldo é feita no agregado e a persistência é atualizada por uma transação de repositório. Em produção, a estratégia recomendada é `UPDATE wallets SET balance_minor = ..., version = version + 1 WHERE id = ? AND version = ?` com retry limitado; isso evita lost updates e mantém o saldo consistente sem lock global. O único identificador da carteira é `(playerId, currency)` e o saldo nunca pode ficar negativo.

## Outbox e idempotência

A idempotência financeira é persistente por `(providerId, externalTransactionId)` e também pela chave de idempotência do cliente. O serviço armazena também um hash de negócio determinístico que exclui a chave de transporte e demais metadados não semânticos. A outbox do sistema garante que eventos de integração só sejam emitidos depois do commit da transação no banco.

## Execução local

O projeto inclui um `docker-compose.yml` com PostgreSQL e LocalStack para reproduzir o ambiente de produção. O `Dockerfile` compila a aplicação Go e usa a porta `8080` por padrão.
