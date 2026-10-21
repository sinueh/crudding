import { parseNewOrder } from "../models/order.js";
import { parseId, respond } from "./respond.js";

// Recebe a requisição, valida a entrada e delega para o OrderService.
export class OrderController {
  #service;

  constructor(service) {
    this.#service = service;
  }

  list = (request) =>
    respond(() => {
      const { userId } = request.query;
      return this.#service.list(userId === undefined ? undefined : parseId(String(userId)));
    });

  getById = (request) => respond(() => this.#service.getById(parseId(request.params.id)));

  create = (request) => respond(() => this.#service.create(parseNewOrder(request.body)), 201);

  pay = (request) => respond(() => this.#service.pay(parseId(request.params.id)));

  cancel = (request) => respond(() => this.#service.cancel(parseId(request.params.id)));
}
