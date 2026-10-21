import express, { type NextFunction, type Request, type Response } from "express";
import type { HttpResponse, Route } from "./types.js";

type RouterMethod = "get" | "post" | "put" | "patch" | "delete";

function send(response: Response, result: HttpResponse): void {
  response.status(result.status);
  if (result.body === undefined) {
    response.end();
    return;
  }
  response.json(result.body);
}

export function startServer(routes: Route[], port = Number(process.env.PORT ?? 3000)): Promise<void> {
  const app = express();
  const router = express.Router();
  app.use(express.json());

  for (const route of routes) {
    const method = route.method.toLowerCase() as RouterMethod;
    // Express 5 encaminha erros de handlers async para o error handler abaixo.
    router.route(route.path)[method](async (request: Request, response: Response) => {
      const result = await route.handler({
        params: request.params as Record<string, string>,
        query: request.query,
        body: request.body,
      });
      send(response, result);
    });
  }

  app.use(router);
  app.use((_request: Request, response: Response) => {
    send(response, { status: 404, body: { error: "rota não encontrada" } });
  });
  app.use((error: Error & { status?: number }, _request: Request, response: Response, _next: NextFunction) => {
    if (error.status === 400) {
      send(response, { status: 400, body: { error: "corpo JSON inválido" } });
      return;
    }
    console.error(error);
    send(response, { status: 500, body: { error: "erro interno" } });
  });

  return new Promise((resolve) => {
    app.listen(port, () => {
      console.log(`Listening on http://localhost:${port}`);
      resolve();
    });
  });
}
