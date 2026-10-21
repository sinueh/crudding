import { randomUUID } from "node:crypto";
import type { ProductRepository } from "../ports/product-repository.js";
import type { CreateProductInput, ProductService } from "../ports/product-service.js";
import { ProductNotFoundError } from "./errors.js";
import { type Product, applyDiscount, createProduct } from "./product.js";

// Implementa a driving port usando apenas a driven port do repositório.
export class ProductManager implements ProductService {
  #repository: ProductRepository;

  constructor(repository: ProductRepository) {
    this.#repository = repository;
  }

  list(): Promise<Product[]> {
    return this.#repository.findAll();
  }

  async get(id: string): Promise<Product> {
    const product = await this.#repository.findById(id);
    if (!product) {
      throw new ProductNotFoundError(id);
    }
    return product;
  }

  async create(input: CreateProductInput): Promise<Product> {
    const product = createProduct({ id: randomUUID(), ...input });
    await this.#repository.save(product);
    return product;
  }

  async applyDiscount(id: string, percent: number): Promise<Product> {
    const discounted = applyDiscount(await this.get(id), percent);
    await this.#repository.save(discounted);
    return discounted;
  }
}
