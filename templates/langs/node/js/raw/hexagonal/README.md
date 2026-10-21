# node-js-raw-hexagonal

Projeto Node.js gerado pelo crudding.

- Runtime: js
- Framework: raw
- Arquitetura: hexagonal

## Executar

```sh
npm install
npm start          # sobe em http://localhost:3000 (use PORT para mudar)
npm run dev        # reinicia a cada alteração
```

## Como o código está organizado

Controllers não dependem do framework: recebem um `HttpRequest` (`params`, `query`, `body`) e devolvem um `HttpResponse` (`status`, `body`). O arquivo `server.js` é o único que conhece o raw e registra a tabela de rotas nele. Trocar de framework significa trocar só esse arquivo.

O domínio fica no centro e se comunica com o mundo externo apenas por portas (interfaces). Adaptadores implementam ou consomem essas portas.

```text
src/
├── entrypoints/main.js   # composition root: liga adaptadores às portas
├── domain/                 # entidade Product, regras e ProductManager (implementa a driving port)
├── ports/                  # ProductService (driving) e ProductRepository (driven)
└── adapters/
    ├── http/               # driving adapter: controller, rotas e server do raw
    └── persistence/        # driven adapter: repositório em memória
```

## Endpoints

| Método | Rota | Descrição |
| --- | --- | --- |
| GET | `/health` | status da API |
| GET | `/products` | lista produtos |
| GET | `/products/:id` | busca um produto |
| POST | `/products` | cria (`{"name": "...", "price": 10, "stock": 5}`) |
| POST | `/products/:id/discount` | aplica desconto (`{"percent": 10}`) |

