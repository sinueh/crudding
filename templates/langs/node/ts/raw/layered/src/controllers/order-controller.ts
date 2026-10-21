import { parseNewOrder } from "../models/order.js";
import type { OrderService } from "../services/order-service.js";
import type { HttpRequest, HttpResponse } from "../http/types.js";
import { parseId, respond } from "./respond.js";

// Recebe a requisição, valida a entrada e delega para o OrderService.
export class OrderController {
  #service: OrderService;

  constructor(service: OrderService) {
    this.#service = service;
  }

  list = (request: HttpRequest): Promise<HttpResponse> =>
    respond(() => {
      const { userId } = request.query;
      return this.#service.list(userId === undefined ? undefined : parseId(String(userId)));
    });

  getById = (request: HttpRequest): Promise<HttpResponse> =>
    respond(() => this.#service.getById(parseId(request.params.id)));

  create = (request: HttpRequest): Promise<HttpResponse> =>
    respond(() => this.#service.create(parseNewOrder(request.body)), 201);

  pay = (request: HttpRequest): Promise<HttpResponse> => respond(() => this.#service.pay(parseId(request.params.id)));

  cancel = (request: HttpRequest): Promise<HttpResponse> =>
    respond(() => this.#service.cancel(parseId(request.params.id)));
}
