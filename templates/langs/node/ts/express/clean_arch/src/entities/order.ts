import { InvalidOrderError, OrderStateError } from "./errors.js";

export type OrderStatus = "pending" | "paid" | "cancelled";

export interface OrderItem {
  sku: string;
  quantity: number;
  unitPrice: number;
}

export interface OrderSnapshot {
  id: string;
  customerId: string;
  items: OrderItem[];
  status: OrderStatus;
  createdAt: string;
}

// Entidade: concentra as regras do pedido e protege seu próprio estado.
export class Order {
  #state: OrderSnapshot;

  constructor(snapshot: OrderSnapshot) {
    this.#state = { ...snapshot, items: snapshot.items.map((item) => ({ ...item })) };
  }

  static create(props: { id: string; customerId: string; items: OrderItem[] }): Order {
    if (props.customerId.trim() === "") {
      throw new InvalidOrderError("customerId é obrigatório");
    }
    if (props.items.length === 0) {
      throw new InvalidOrderError("o pedido precisa de ao menos um item");
    }
    for (const item of props.items) {
      if (item.sku.trim() === "" || !Number.isInteger(item.quantity) || item.quantity <= 0 || item.unitPrice <= 0) {
        throw new InvalidOrderError(`item inválido: ${JSON.stringify(item)}`);
      }
    }
    return new Order({ ...props, status: "pending", createdAt: new Date().toISOString() });
  }

  get id(): string {
    return this.#state.id;
  }

  get status(): OrderStatus {
    return this.#state.status;
  }

  get total(): number {
    const total = this.#state.items.reduce((sum, item) => sum + item.quantity * item.unitPrice, 0);
    return Math.round(total * 100) / 100;
  }

  pay(): void {
    if (this.#state.status !== "pending") {
      throw new OrderStateError(`pedido ${this.id} está ${this.#state.status} e não pode ser pago`);
    }
    this.#state.status = "paid";
  }

  cancel(): void {
    if (this.#state.status === "paid") {
      throw new OrderStateError(`pedido ${this.id} já foi pago e não pode ser cancelado`);
    }
    this.#state.status = "cancelled";
  }

  toSnapshot(): OrderSnapshot {
    return { ...this.#state, items: this.#state.items.map((item) => ({ ...item })) };
  }
}
