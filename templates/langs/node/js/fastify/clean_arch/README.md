# node-js-fastify-clean_arch

Projeto Node.js gerado pelo crudding.

- Runtime: js
- Framework: fastify
- Arquitetura: clean_arch

## Executar

```sh
npm install
npm start          # sobe em http://localhost:3000 (use PORT para mudar)
npm run dev        # reinicia a cada alteração
```

## Como o código está organizado

Controllers não dependem do framework: recebem um `HttpRequest` (`params`, `query`, `body`) e devolvem um `HttpResponse` (`status`, `body`). O arquivo `server.js` é o único que conhece o fastify e registra a tabela de rotas nele. Trocar de framework significa trocar só esse arquivo.

As dependências apontam sempre para dentro: frameworks → adapters → use-cases → entities.

```text
src/
├── entities/        # entidade Order com suas regras e erros
├── use-cases/       # CreateOrder, GetOrder, ListOrders, PayOrder, DTOs e a porta OrderRepository
├── adapters/
│   ├── controllers/ # converte HTTP em chamadas aos casos de uso
│   ├── presenters/  # formata a saída (ex.: total em BRL)
│   └── repositories/# implementação em memória da porta OrderRepository
└── frameworks/
    ├── main.js    # composition root
    └── http/        # rotas e server do fastify
```

## Endpoints

| Método | Rota | Descrição |
| --- | --- | --- |
| GET | `/health` | status da API |
| GET | `/orders` | lista pedidos |
| GET | `/orders/:id` | busca um pedido |
| POST | `/orders` | cria (`{"customerId": "c1", "items": [{"sku": "A", "quantity": 1, "unitPrice": 10}]}`) |
| POST | `/orders/:id/pay` | paga um pedido pendente |

