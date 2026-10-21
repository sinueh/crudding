import express from "express";

function send(response, result) {
  response.status(result.status);
  if (result.body === undefined) {
    response.end();
    return;
  }
  response.json(result.body);
}

export function startServer(routes, port = Number(process.env.PORT ?? 3000)) {
  const app = express();
  const router = express.Router();
  app.use(express.json());

  for (const route of routes) {
    const method = route.method.toLowerCase();
    // Express 5 encaminha erros de handlers async para o error handler abaixo.
    router.route(route.path)[method](async (request, response) => {
      const result = await route.handler({
        params: request.params,
        query: request.query,
        body: request.body,
      });
      send(response, result);
    });
  }

  app.use(router);
  app.use((_request, response) => {
    send(response, { status: 404, body: { error: "rota não encontrada" } });
  });
  app.use((error, _request, response, _next) => {
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
