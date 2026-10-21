import type { Product } from "../../domain/product.js";
import type { ProductRepository } from "../../ports/product-repository.js";

// Driven adapter: implementa a porta de persistência em memória. Um adaptador
// Postgres/Mongo implementaria a mesma interface sem mudar o domínio.
export class InMemoryProductRepository implements ProductRepository {
  #products = new Map<string, Product>();

  async findAll(): Promise<Product[]> {
    return [...this.#products.values()];
  }

  async findById(id: string): Promise<Product | null> {
    return this.#products.get(id) ?? null;
  }

  async save(product: Product): Promise<void> {
    this.#products.set(product.id, { ...product });
  }
}
