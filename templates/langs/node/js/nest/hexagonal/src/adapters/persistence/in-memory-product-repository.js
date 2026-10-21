// Driven adapter: implementa a porta de persistência em memória. Um adaptador
// Postgres/Mongo implementaria a mesma interface sem mudar o domínio.
export class InMemoryProductRepository {
  #products = new Map();

  async findAll() {
    return [...this.#products.values()];
  }

  async findById(id) {
    return this.#products.get(id) ?? null;
  }

  async save(product) {
    this.#products.set(product.id, { ...product });
  }
}
