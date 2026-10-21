import { InvalidProductError, ProductNotFoundError } from "../../domain/errors.js";

// Driving adapter: traduz HTTP para chamadas na porta ProductService.
export class ProductController {
  #service;

  constructor(service) {
    this.#service = service;
  }

  list = () => this.#handle(200, () => this.#service.list());

  get = (request) => this.#handle(200, () => this.#service.get(request.params.id));

  create = (request) =>
    this.#handle(201, () => {
      const body = request.body ?? {};
      return this.#service.create({
        name: String(body.name ?? ""),
        price: Number(body.price),
        stock: Number(body.stock ?? 0),
      });
    });

  applyDiscount = (request) =>
    this.#handle(200, () => {
      const body = request.body ?? {};
      return this.#service.applyDiscount(request.params.id, Number(body.percent));
    });

  async #handle(status, action) {
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
