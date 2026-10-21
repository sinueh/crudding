import { InvalidProductError, ProductNotFoundError } from "../../domain/errors.js";
import type { ProductService } from "../../ports/product-service.js";
import type { HttpRequest, HttpResponse } from "./types.js";

// Driving adapter: traduz HTTP para chamadas na porta ProductService.
export class ProductController {
  #service: ProductService;

  constructor(service: ProductService) {
    this.#service = service;
  }

  list = (): Promise<HttpResponse> => this.#handle(200, () => this.#service.list());

  get = (request: HttpRequest): Promise<HttpResponse> => this.#handle(200, () => this.#service.get(request.params.id));

  create = (request: HttpRequest): Promise<HttpResponse> =>
    this.#handle(201, () => {
      const body = (request.body ?? {}) as Record<string, unknown>;
      return this.#service.create({
        name: String(body.name ?? ""),
        price: Number(body.price),
        stock: Number(body.stock ?? 0),
      });
    });

  applyDiscount = (request: HttpRequest): Promise<HttpResponse> =>
    this.#handle(200, () => {
      const body = (request.body ?? {}) as Record<string, unknown>;
      return this.#service.applyDiscount(request.params.id, Number(body.percent));
    });

  async #handle(status: number, action: () => Promise<unknown>): Promise<HttpResponse> {
    try {
      return { status, body: await action() };
    } catch (error) {
      if (error instanceof InvalidProductError) {
        return { status: 422, body: { error: error.message } };
      }
      if (error instanceof ProductNotFoundError) {
        return { status: 404, body: { error: error.message } };
      }
      throw error;
    }
  }
}
