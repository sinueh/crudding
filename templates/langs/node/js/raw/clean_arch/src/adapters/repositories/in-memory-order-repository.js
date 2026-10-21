import { Order } from "../../entities/order.js";

// Implementa a porta OrderRepository guardando snapshots, como faria um
// repositório real ao mapear a entidade para linhas de uma tabela.
export class InMemoryOrderRepository {
  #rows = new Map();

  async findAll() {
    return [...this.#rows.values()].map((row) => new Order(row));
  }

  async findById(id) {
    const row = this.#rows.get(id);
    return row ? new Order(row) : null;
  }

  async save(order) {
    this.#rows.set(order.id, order.toSnapshot());
  }
}
