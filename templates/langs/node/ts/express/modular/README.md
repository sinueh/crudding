# node-ts-express-modular

Projeto Node.js gerado pelo crudding.

- Runtime: ts
- Framework: express
- Arquitetura: modular

## Executar

```sh
npm install
npm start          # sobe em http://localhost:3000 (use PORT para mudar)
npm run dev        # reinicia a cada alteração
npm run typecheck
```

## Como o código está organizado

Controllers não dependem do framework: recebem um `HttpRequest` (`params`, `query`, `body`) e devolvem um `HttpResponse` (`status`, `body`). O arquivo `server.ts` é o único que conhece o express e registra a tabela de rotas nele. Trocar de framework significa trocar só esse arquivo.

O código é agrupado por funcionalidade. Cada módulo expõe sua API pública em `index.ts`, e outros módulos só importam de lá.

```text
src/
├── main.ts           # monta os módulos e sobe o servidor
├── modules/
│   ├── catalog/        # produtos e estoque (model, service, controller, routes)
│   └── orders/         # pedidos; usa a API pública do catalog para reservar estoque
└── shared/
    ├── database.ts   # coleção em memória usada como "tabela"
    ├── errors.ts     # erros de negócio
    ├── respond.ts    # tradução de erros para HTTP
    └── http/           # server (adaptador do express) e tipos HTTP
```

## Endpoints

| Método | Rota | Descrição |
| --- | --- | --- |
| GET | `/health` | status da API |
| GET | `/products` | lista produtos |
| GET | `/products/:id` | busca um produto |
| POST | `/products` | cria (`{"sku": "...", "name": "...", "price": 10, "stock": 5}`) |
| GET | `/orders` | lista pedidos |
| GET | `/orders/:id` | busca um pedido |
| POST | `/orders` | cria e reserva estoque (`{"items": [{"productId": 1, "quantity": 2}]}`) |

