import { InvalidOrderError, OrderStateError } from "../../entities/errors.js";
import type { OrderItem } from "../../entities/order.js";
import type { CreateOrder } from "../../use-cases/create-order.js";
import { OrderNotFoundError } from "../../use-cases/errors.js";
import type { GetOrder } from "../../use-cases/get-order.js";
import type { ListOrders } from "../../use-cases/list-orders.js";
import type { PayOrder } from "../../use-cases/pay-order.js";
import { presentOrder } from "../presenters/order-presenter.js";

// Contratos próprios do controller: a camada de frameworks se adapta a eles,
// e não o contrário.
export interface ControllerRequest {
  params: Record<string, string>;
  body: unknown;
}

export interface ControllerResponse {
  status: number;
  body?: unknown;
}

export interface OrderUseCases {
  createOrder: CreateOrder;
  getOrder: GetOrder;
  listOrders: ListOrders;
  payOrder: PayOrder;
}

export class OrderController {
  #useCases: OrderUseCases;

  constructor(useCases: OrderUseCases) {
    this.#useCases = useCases;
  }

  list = (): Promise<ControllerResponse> =>
    this.#handle(200, async () => (await this.#useCases.listOrders.execute()).map(presentOrder));

  get = (request: ControllerRequest): Promise<ControllerResponse> =>
    this.#handle(200, async () => presentOrder(await this.#useCases.getOrder.execute(request.params.id)));

  create = (request: ControllerRequest): Promise<ControllerResponse> =>
    this.#handle(201, async () => {
      const body = (request.body ?? {}) as { customerId?: unknown; items?: unknown };
      const items: OrderItem[] = Array.isArray(body.items)
        ? body.items.map((item: Record<string, unknown>) => ({
            sku: String(item.sku ?? ""),
            quantity: Number(item.quantity),
            unitPrice: Number(item.unitPrice),
          }))
        : [];
      const output = await this.#useCases.createOrder.execute({ customerId: String(body.customerId ?? ""), items });
      return presentOrder(output);
    });

  pay = (request: ControllerRequest): Promise<ControllerResponse> =>
    this.#handle(200, async () => presentOrder(await this.#useCases.payOrder.execute(request.params.id)));

  async #handle(status: number, action: () => Promise<unknown>): Promise<ControllerResponse> {
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
