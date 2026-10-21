import { randomUUID } from "node:crypto";
import { ProductNotFoundError } from "./errors.js";
import { applyDiscount, createProduct } from "./product.js";

// Implementa a driving port usando apenas a driven port do repositório.
export class ProductManager {
  #repository;

  constructor(repository) {
    this.#repository = repository;
  }

  list() {
    return this.#repository.findAll();
  }

  async get(id) {
    const product = await this.#repository.findById(id);
    if (!product) {
      throw new ProductNotFoundError(id);
    }
    return product;
  }

  async create(input) {
    const product = createProduct({ id: randomUUID(), ...input });
    await this.#repository.save(product);
    return product;
  }

  async applyDiscount(id, percent) {
    const discounted = applyDiscount(await this.get(id), percent);
    await this.#repository.save(discounted);
    return discounted;
  }
}
