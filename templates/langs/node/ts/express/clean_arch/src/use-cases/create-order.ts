import { randomUUID } from "node:crypto";
import { type OrderItem, Order } from "../entities/order.js";
import { type OrderOutput, toOrderOutput } from "./order-output.js";
import type { OrderRepository } from "./ports/order-repository.js";

export interface CreateOrderInput {
  customerId: string;
  items: OrderItem[];
}

export class CreateOrder {
  #orders: OrderRepository;

  constructor(orders: OrderRepository) {
    this.#orders = orders;
  }

  async execute(input: CreateOrderInput): Promise<OrderOutput> {
    const order = Order.create({ id: randomUUID(), ...input });
    await this.#orders.save(order);
    return toOrderOutput(order);
  }
}
