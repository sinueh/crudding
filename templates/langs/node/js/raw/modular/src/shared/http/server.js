import { createServer } from "node:http";

class InvalidJsonError extends Error {}

// Converte "/users/:id" em uma expressão regular que captura os parâmetros.
function compile(route) {
  const keys = [];
  const source = route.path.replace(/\/+$/, "").replace(/:(\w+)/g, (_match, key) => {
    keys.push(key);
    return "([^/]+)";
  });
  return { ...route, keys, pattern: new RegExp(`^${source}/?$`) };
}

async function readBody(request) {
  const chunks = [];
  for await (const chunk of request) {
    chunks.push(chunk);
  }
  const raw = Buffer.concat(chunks).toString("utf8");
  if (raw === "") {
    return undefined;
  }
  try {
    return JSON.parse(raw);
  } catch {
    throw new InvalidJsonError("corpo JSON inválido");
  }
}

function send(response, result) {
  if (result.body === undefined) {
    response.writeHead(result.status).end();
    return;
  }
  response.writeHead(result.status, { "content-type": "application/json" });
  response.end(JSON.stringify(result.body));
}

export function startServer(routes, port = Number(process.env.PORT ?? 3000)) {
  const compiled = routes.map(compile);

  const server = createServer(async (request, response) => {
    const url = new URL(request.url ?? "/", "http://localhost");
    const method = request.method ?? "GET";

    for (const route of compiled) {
      const match = route.method === method ? route.pattern.exec(url.pathname) : null;
      if (!match) {
        continue;
      }
      const params = Object.fromEntries(route.keys.map((key, index) => [key, decodeURIComponent(match[index + 1])]));
      try {
        const body = await readBody(request);
        send(response, await route.handler({ params, query: Object.fromEntries(url.searchParams), body }));
      } catch (error) {
        if (error instanceof InvalidJsonError) {
          send(response, { status: 400, body: { error: error.message } });
          return;
        }
        console.error(error);
        send(response, { status: 500, body: { error: "erro interno" } });
      }
      return;
    }

    send(response, { status: 404, body: { error: "rota não encontrada" } });
  });

  return new Promise((resolve) => {
    server.listen(port, () => {
      console.log(`Listening on http://localhost:${port}`);
      resolve();
    });
  });
}
