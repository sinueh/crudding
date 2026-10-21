import type { InMemoryCollection } from "../../shared/database.js";
import { NotFoundError } from "../../shared/errors.js";
import type { CatalogService } from "../catalog/index.js";
import type { Order, OrderLine, PlaceOrderItem } from "./order.js";

// O módulo orders depende apenas do que precisa da API pública do catalog.
export type Catalog = Pick<CatalogService, "getById" | "reserve">;

export class OrderService {
  #orders: InMemoryCollection<Order>;
  #catalog: Catalog;

  constructor(orders: InMemoryCollection<Order>, catalog: Catalog) {
    this.#orders = orders;
    this.#catalog = catalog;
  }

  list(): Order[] {
    return this.#orders.findAll();
  }

  getById(id: number): Order {
    const order = this.#orders.findById(id);
    if (!order) {
      throw new NotFoundError(`pedido ${id} não encontrado`);
    }
    return order;
  }

  place(items: PlaceOrderItem[]): Order {
    // Valida todos os produtos antes de reservar qualquer estoque.
    items.forEach((item) => this.#catalog.getById(item.productId));

    const lines: OrderLine[] = items.map((item) => {
      const product = this.#catalog.reserve(item.productId, item.quantity);
      return { productId: product.id, name: product.name, quantity: item.quantity, unitPrice: product.price };
    });
    const total = lines.reduce((sum, line) => sum + line.quantity * line.unitPrice, 0);

    return this.#orders.insert({ lines, total: Math.round(total * 100) / 100, createdAt: new Date().toISOString() });
  }
}
