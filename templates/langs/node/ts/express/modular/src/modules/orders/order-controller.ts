import type { HttpRequest, HttpResponse } from "../../shared/http/types.js";
import { parseId, respond } from "../../shared/respond.js";
import { parsePlaceOrder } from "./order.js";
import type { OrderService } from "./order-service.js";

export class OrderController {
  #service: OrderService;

  constructor(service: OrderService) {
    this.#service = service;
  }

  list = (): Promise<HttpResponse> => respond(() => this.#service.list());

  getById = (request: HttpRequest): Promise<HttpResponse> =>
    respond(() => this.#service.getById(parseId(request.params.id)));

  place = (request: HttpRequest): Promise<HttpResponse> =>
    respond(() => this.#service.place(parsePlaceOrder(request.body)), 201);
}
