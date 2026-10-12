# Desafio Backend

Este projeto implementa uma API Go com Uber Fx para processamento distribuído de carteiras e apostas.

## Rodando localmente

```bash
docker compose up --build
```

Em seguida, a API estará disponível em `http://localhost:8080`.

## Endpoints principais

- `POST /wallets`
- `GET /wallets/:walletId`
- `POST /wagering/transactions`
- `GET /health/live`
- `GET /health/ready`

## Testes

```bash
go test ./...
```
