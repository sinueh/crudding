# node-ts-fastify-layered

Projeto Node.js gerado pelo crudding.

- Runtime: ts
- Framework: fastify
- Arquitetura: layered

## Executar

```sh
npm install
npm start          # sobe em http://localhost:3000 (use PORT para mudar)
npm run dev        # reinicia a cada alteração
npm run typecheck
```

## Como o código está organizado

Controllers não dependem do framework: recebem um `HttpRequest` (`params`, `query`, `body`) e devolvem um `HttpResponse` (`status`, `body`). O arquivo `server.ts` é o único que conhece o fastify e registra a tabela de rotas nele. Trocar de framework significa trocar só esse arquivo.

Cada camada só conhece a camada logo abaixo: routes → controllers → services → models.

```text
src/
├── main.ts      # composição: instancia services e controllers
├── http/            # server (adaptador do fastify) e tipos HTTP
├── routes/          # tabela de rotas por recurso (router)
├── controllers/     # validação da entrada e tradução de erros para HTTP
├── services/        # regras de negócio e armazenamento em memória
└── models/          # formatos dos dados, parse do payload e erros de negócio
```

## Endpoints

| Método | Rota | Descrição |
| --- | --- | --- |
| GET | `/health` | status da API |
| GET | `/users` | lista usuários |
| GET | `/users/:id` | busca um usuário |
| POST | `/users` | cria (`{"name": "...", "email": "..."}`) |
| GET | `/orders?userId=1` | lista pedidos (filtro opcional) |
| GET | `/orders/:id` | busca um pedido |
| POST | `/orders` | cria (`{"userId": 1, "items": [{"product": "...", "quantity": 1, "unitPrice": 10}]}`) |
| POST | `/orders/:id/pay` | paga um pedido pendente |
| POST | `/orders/:id/cancel` | cancela um pedido não pago |

