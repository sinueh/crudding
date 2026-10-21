import { OrderNotFoundError } from "./errors.js";
import { toOrderOutput } from "./order-output.js";

export class GetOrder {
  #orders;

  constructor(orders) {
    this.#orders = orders;
  }

  async execute(id) {
    const order = await this.#orders.findById(id);
    if (!order) {
      throw new OrderNotFoundError(id);
    }
    return toOrderOutput(order);
  }
}
