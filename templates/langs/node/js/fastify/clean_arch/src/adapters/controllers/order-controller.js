import { InvalidOrderError, OrderStateError } from "../../entities/errors.js";
import { OrderNotFoundError } from "../../use-cases/errors.js";
import { presentOrder } from "../presenters/order-presenter.js";

export class OrderController {
  #useCases;

  constructor(useCases) {
    this.#useCases = useCases;
  }

  list = () => this.#handle(200, async () => (await this.#useCases.listOrders.execute()).map(presentOrder));

  get = (request) =>
    this.#handle(200, async () => presentOrder(await this.#useCases.getOrder.execute(request.params.id)));

  create = (request) =>
    this.#handle(201, async () => {
      const body = request.body ?? {};
      const items = Array.isArray(body.items)
        ? body.items.map((item) => ({
            sku: String(item.sku ?? ""),
            quantity: Number(item.quantity),
            unitPrice: Number(item.unitPrice),
          }))
        : [];
      const output = await this.#useCases.createOrder.execute({ customerId: String(body.customerId ?? ""), items });
      return presentOrder(output);
    });

  pay = (request) =>
    this.#handle(200, async () => presentOrder(await this.#useCases.payOrder.execute(request.params.id)));

  async #handle(status, action) {
    try {
      return { status, body: await action() };
    } catch (error) {
      if (error instanceof InvalidOrderError) {
        return { status: 422, body: { error: error.message } };
      }
      if (error instanceof OrderNotFoundError) {
        return { status: 404, body: { error: error.message } };
      }
      if (error instanceof OrderStateError) {
        return { status: 409, body: { error: error.message } };
      }
      throw error;
    }
  }
}
