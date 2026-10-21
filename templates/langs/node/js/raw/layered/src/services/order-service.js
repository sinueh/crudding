import { ConflictError, NotFoundError } from "../models/errors.js";

// Regras de negócio de pedidos: total calculado no servidor e transições de status.
export class OrderService {
  #orders = new Map();
  #nextId = 1;
  #users;

  constructor(users) {
    this.#users = users;
  }

  list(userId) {
    const orders = [...this.#orders.values()];
    return userId === undefined ? orders : orders.filter((order) => order.userId === userId);
  }

  getById(id) {
    const order = this.#orders.get(id);
    if (!order) {
      throw new NotFoundError(`pedido ${id} não encontrado`);
    }
    return order;
  }

  create(input) {
    this.#users.getById(input.userId);
    const total = input.items.reduce((sum, item) => sum + item.quantity * item.unitPrice, 0);
    const order = {
      id: this.#nextId++,
      userId: input.userId,
      items: input.items,
      total: Math.round(total * 100) / 100,
      status: "pending",
    };
    this.#orders.set(order.id, order);
    return order;
  }

  pay(id) {
    const order = this.getById(id);
    if (order.status !== "pending") {
      throw new ConflictError(`pedido ${id} está ${order.status} e não pode ser pago`);
    }
    order.status = "paid";
    return order;
  }

  cancel(id) {
    const order = this.getById(id);
    if (order.status === "paid") {
      throw new ConflictError(`pedido ${id} já foi pago e não pode ser cancelado`);
    }
    order.status = "cancelled";
    return order;
  }
}
