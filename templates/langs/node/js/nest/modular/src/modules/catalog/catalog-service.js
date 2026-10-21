import { ConflictError, NotFoundError } from "../../shared/errors.js";

export class CatalogService {
  #products;

  constructor(products) {
    this.#products = products;
  }

  list() {
    return this.#products.findAll();
  }

  getById(id) {
    const product = this.#products.findById(id);
    if (!product) {
      throw new NotFoundError(`produto ${id} não encontrado`);
    }
    return product;
  }

  create(input) {
    if (this.list().some((product) => product.sku === input.sku)) {
      throw new ConflictError(`sku ${input.sku} já cadastrado`);
    }
    return this.#products.insert(input);
  }

  // API pública usada por outros módulos: baixa o estoque e devolve o produto.
  reserve(id, quantity) {
    const product = this.getById(id);
    if (product.stock < quantity) {
      throw new ConflictError(`estoque insuficiente para ${product.sku}`);
    }
    return this.#products.update({ ...product, stock: product.stock - quantity });
  }
}
