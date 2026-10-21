import type { InMemoryCollection } from "../../shared/database.js";
import { ConflictError, NotFoundError } from "../../shared/errors.js";
import type { NewProduct, Product } from "./product.js";

export class CatalogService {
  #products: InMemoryCollection<Product>;

  constructor(products: InMemoryCollection<Product>) {
    this.#products = products;
  }

  list(): Product[] {
    return this.#products.findAll();
  }

  getById(id: number): Product {
    const product = this.#products.findById(id);
    if (!product) {
      throw new NotFoundError(`produto ${id} não encontrado`);
    }
    return product;
  }

  create(input: NewProduct): Product {
    if (this.list().some((product) => product.sku === input.sku)) {
      throw new ConflictError(`sku ${input.sku} já cadastrado`);
    }
    return this.#products.insert(input);
  }

  // API pública usada por outros módulos: baixa o estoque e devolve o produto.
  reserve(id: number, quantity: number): Product {
    const product = this.getById(id);
    if (product.stock < quantity) {
      throw new ConflictError(`estoque insuficiente para ${product.sku}`);
    }
    return this.#products.update({ ...product, stock: product.stock - quantity });
  }
}
