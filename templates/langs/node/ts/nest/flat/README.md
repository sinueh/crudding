# node-ts-nest-flat

Projeto Node.js gerado pelo crudding.

- Runtime: ts
- Framework: nest
- Arquitetura: flat

## Executar

```sh
npm install
npm start          # sobe em http://localhost:3000 (use PORT para mudar)
npm run dev        # reinicia a cada alteração
npm test
npm run typecheck
```

## Como o código está organizado

Controllers não dependem do framework: recebem um `HttpRequest` (`params`, `query`, `body`) e devolvem um `HttpResponse` (`status`, `body`). O arquivo `server.ts` é o único que conhece o nest e registra a tabela de rotas nele. Trocar de framework significa trocar só esse arquivo.

```text
src/
├── main.ts         # cria o store e sobe o servidor
├── server.ts       # adaptador do nest
├── tasks.ts        # modelo, armazenamento em memória, validação e rotas de tarefas
└── tasks.test.ts   # testes com node:test
```

## Endpoints

| Método | Rota | Descrição |
| --- | --- | --- |
| GET | `/tasks` | lista tarefas |
| GET | `/tasks/:id` | busca uma tarefa |
| POST | `/tasks` | cria (`{"title": "..."}`) |
| PATCH | `/tasks/:id/done` | marca como concluída |
| DELETE | `/tasks/:id` | remove |

