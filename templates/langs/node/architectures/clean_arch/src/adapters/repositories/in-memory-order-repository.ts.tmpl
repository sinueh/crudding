import { Order, type OrderSnapshot } from "../../entities/order.js";
import type { OrderRepository } from "../../use-cases/ports/order-repository.js";

// Implementa a porta OrderRepository guardando snapshots, como faria um
// repositório real ao mapear a entidade para linhas de uma tabela.
export class InMemoryOrderRepository implements OrderRepository {
  #rows = new Map<string, OrderSnapshot>();

  async findAll(): Promise<Order[]> {
    return [...this.#rows.values()].map((row) => new Order(row));
  }

  async findById(id: string): Promise<Order | null> {
    const row = this.#rows.get(id);
    return row ? new Order(row) : null;
  }

  async save(order: Order): Promise<void> {
    this.#rows.set(order.id, order.toSnapshot());
  }
}
