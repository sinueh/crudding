import { randomUUID } from "node:crypto";
import { Order } from "../entities/order.js";
import { toOrderOutput } from "./order-output.js";

export class CreateOrder {
  #orders;

  constructor(orders) {
    this.#orders = orders;
  }

  async execute(input) {
    const order = Order.create({ id: randomUUID(), ...input });
    await this.#orders.save(order);
    return toOrderOutput(order);
  }
}
