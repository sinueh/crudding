import { NotFoundError } from "../../shared/errors.js";

// O módulo orders depende apenas da API pública do catalog (getById e reserve).
export class OrderService {
  #orders;
  #catalog;

  constructor(orders, catalog) {
    this.#orders = orders;
    this.#catalog = catalog;
  }

  list() {
    return this.#orders.findAll();
  }

  getById(id) {
    const order = this.#orders.findById(id);
    if (!order) {
      throw new NotFoundError(`pedido ${id} não encontrado`);
    }
    return order;
  }

  place(items) {
    // Valida todos os produtos antes de reservar qualquer estoque.
    items.forEach((item) => this.#catalog.getById(item.productId));

    const lines = items.map((item) => {
      const product = this.#catalog.reserve(item.productId, item.quantity);
      return { productId: product.id, name: product.name, quantity: item.quantity, unitPrice: product.price };
    });
    const total = lines.reduce((sum, line) => sum + line.quantity * line.unitPrice, 0);

    return this.#orders.insert({ lines, total: Math.round(total * 100) / 100, createdAt: new Date().toISOString() });
  }
}
