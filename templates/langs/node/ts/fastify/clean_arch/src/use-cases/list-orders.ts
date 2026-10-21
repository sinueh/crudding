import { type OrderOutput, toOrderOutput } from "./order-output.js";
import type { OrderRepository } from "./ports/order-repository.js";

export class ListOrders {
  #orders: OrderRepository;

  constructor(orders: OrderRepository) {
    this.#orders = orders;
  }

  async execute(): Promise<OrderOutput[]> {
    const orders = await this.#orders.findAll();
    return orders.map(toOrderOutput);
  }
}
