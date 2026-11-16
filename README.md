# Tech Challenge — Oficina Mecânica · Billing Service

Microsserviço de **orçamento e pagamento** da oficina mecânica. Gera o orçamento a partir do diagnóstico, registra a aprovação ou a recusa do cliente, cobra pelo Mercado Pago e estorna quando a saga precisa compensar.

Parte do Tech Challenge da pós-graduação em Arquitetura de Software (FIAP SOAT) — Fase 4.

## Repositórios do projeto

| Repositório | Papel |
|---|---|
| [`oficina-os-service`](https://github.com/gaabriel165/oficina-os-service) | Ordens de serviço, clientes, veículos e orquestração da saga |
| **`oficina-billing-service`** (este) | Orçamento e pagamento |
| [`oficina-execution-service`](https://github.com/gaabriel165/oficina-execution-service) | Diagnóstico, execução, catálogo e estoque |
| [`oficina-lambda-auth`](https://github.com/gaabriel165/oficina-lambda-auth) | Autenticação por CPF e API Gateway |
| [`oficina-infra-k8s`](https://github.com/gaabriel165/oficina-infra-k8s) | VPC, EKS e base compartilhada |
| [`oficina-infra-db`](https://github.com/gaabriel165/oficina-infra-db) | Banco do OS Service |

## Stack

- Go 1.26, Gin, GORM
- PostgreSQL 16 (banco exclusivo deste serviço), migrations com golang-migrate
- Amazon SQS para comandos e respostas da saga
- OpenTelemetry (traces) e `slog` em JSON (logs)

## Comunicação

O serviço não expõe chamadas síncronas para os outros microsserviços. Ele consome comandos da fila `oficina-billing-commands` e responde na fila `oficina-saga-replies`. As mensagens de saída são gravadas na tabela `outbox_messages` na mesma transação da mudança de estado e publicadas em seguida; o consumidor registra cada mensagem processada para ignorar repetições.

Os contratos estão em [`messaging-contracts.md`](https://github.com/gaabriel165/oficina-os-service/blob/main/docs/architecture/messaging-contracts.md).

## Como executar

```bash
cp .env.example .env
docker-compose up --build
```

Sobem a aplicação (porta `8081`), o PostgreSQL e o ElasticMQ, que emula o SQS localmente.

```bash
curl http://localhost:8081/health
curl http://localhost:8081/ready
```

## Testes

```bash
make test/unit
make test/integration
make test/coverage
```

Os testes de integração usam testcontainers e precisam do Docker em execução.

## Estrutura de pastas

```
cmd/api                  ponto de entrada
configs                  leitura de variáveis de ambiente
internal/domain          entidades, portas e contratos de mensagens
internal/application     casos de uso e relay do outbox
internal/infrastructure  PostgreSQL, SQS e observabilidade
internal/api             handlers, middlewares e DTOs
migrations               migrations versionadas
```
