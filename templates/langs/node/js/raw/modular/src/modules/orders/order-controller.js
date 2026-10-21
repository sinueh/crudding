import { parseId, respond } from "../../shared/respond.js";
import { parsePlaceOrder } from "./order.js";

export class OrderController {
  #service;

  constructor(service) {
    this.#service = service;
  }

  list = () => respond(() => this.#service.list());

  getById = (request) => respond(() => this.#service.getById(parseId(request.params.id)));

  place = (request) => respond(() => this.#service.place(parsePlaceOrder(request.body)), 201);
}
