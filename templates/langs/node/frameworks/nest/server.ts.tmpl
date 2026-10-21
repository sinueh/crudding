import "reflect-metadata";
import { Controller, Delete, Get, Module, Patch, Post, Put, Req, Res } from "@nestjs/common";
import { NestFactory } from "@nestjs/core";
import type { Request, Response } from "express";
import type { HttpMethod, Route } from "./types.js";

const methodDecorators: Record<HttpMethod, (path: string) => MethodDecorator> = {
  GET: Get,
  POST: Post,
  PUT: Put,
  PATCH: Patch,
  DELETE: Delete,
};

// Monta um controller Nest a partir da tabela de rotas. Os decorators são
// aplicados manualmente para que o mesmo código funcione em JS e em TS.
function createController(routes: Route[]): new () => object {
  class RoutesController {}

  routes.forEach((route, index) => {
    const name = `route${index}`;
    Object.defineProperty(RoutesController.prototype, name, {
      configurable: true,
      writable: true,
      value: async (request: Request, response: Response) => {
        const result = await route.handler({
          params: request.params as Record<string, string>,
          query: request.query,
          body: request.body,
        });
        response.status(result.status);
        if (result.body === undefined) {
          response.end();
        } else {
          response.json(result.body);
        }
      },
    });
    const descriptor = Object.getOwnPropertyDescriptor(RoutesController.prototype, name)!;
    methodDecorators[route.method](route.path)(RoutesController.prototype, name, descriptor);
    Req()(RoutesController.prototype, name, 0);
    Res()(RoutesController.prototype, name, 1);
  });

  Controller()(RoutesController);
  return RoutesController;
}

export async function startServer(routes: Route[], port = Number(process.env.PORT ?? 3000)): Promise<void> {
  class AppModule {}
  Module({ controllers: [createController(routes)] })(AppModule);

  const app = await NestFactory.create(AppModule);
  await app.listen(port);
  console.log(`Listening on http://localhost:${port}`);
}
