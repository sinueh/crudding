import { OrderNotFoundError } from "./errors.js";
import { type OrderOutput, toOrderOutput } from "./order-output.js";
import type { OrderRepository } from "./ports/order-repository.js";

export class GetOrder {
  #orders: OrderRepository;

  constructor(orders: OrderRepository) {
    this.#orders = orders;
  }

  async execute(id: string): Promise<OrderOutput> {
    const order = await this.#orders.findById(id);
    if (!order) {
      throw new OrderNotFoundError(id);
    }
    return toOrderOutput(order);
  }
}
