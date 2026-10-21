import Fastify from "fastify";
import type { Route } from "./types.js";

export async function startServer(routes: Route[], port = Number(process.env.PORT ?? 3000)): Promise<void> {
  const app = Fastify({ logger: true });

  // Aceita POST sem corpo mesmo com content-type JSON (ex.: /orders/:id/pay).
  app.removeContentTypeParser("application/json");
  app.addContentTypeParser("application/json", { parseAs: "string" }, (_request, body, done) => {
    if (body === "") {
      done(null, undefined);
      return;
    }
    try {
      done(null, JSON.parse(body as string));
    } catch {
      done(Object.assign(new Error("corpo JSON inválido"), { statusCode: 400 }), undefined);
    }
  });

  for (const route of routes) {
    app.route({
      method: route.method,
      url: route.path,
      handler: async (request, reply) => {
        const result = await route.handler({
          params: request.params as Record<string, string>,
          query: request.query as Record<string, unknown>,
          body: request.body,
        });
        return reply.status(result.status).send(result.body);
      },
    });
  }

  app.setNotFoundHandler((_request, reply) => {
    reply.status(404).send({ error: "rota não encontrada" });
  });
  app.setErrorHandler((error: Error & { statusCode?: number }, request, reply) => {
    if (error.statusCode && error.statusCode < 500) {
      reply.status(error.statusCode).send({ error: error.message });
      return;
    }
    request.log.error(error);
    reply.status(500).send({ error: "erro interno" });
  });

  await app.listen({ host: "0.0.0.0", port });
}
