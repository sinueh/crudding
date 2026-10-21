import { InvalidProductError } from "./errors.js";

/**
 * @typedef {object} Product
 * @property {string} id
 * @property {string} name
 * @property {number} price
 * @property {number} stock
 */

// Fábrica que garante as invariantes do produto.
export function createProduct(props) {
  const name = props.name.trim();
  if (name.length < 3) {
    throw new InvalidProductError("o nome do produto precisa ter ao menos 3 caracteres");
  }
  if (!Number.isFinite(props.price) || props.price <= 0) {
    throw new InvalidProductError("o preço precisa ser maior que zero");
  }
  if (!Number.isInteger(props.stock) || props.stock < 0) {
    throw new InvalidProductError("o estoque precisa ser um inteiro não negativo");
  }
  return { ...props, name, price: Math.round(props.price * 100) / 100 };
}

export function applyDiscount(product, percent) {
  if (!Number.isFinite(percent) || percent <= 0 || percent >= 100) {
    throw new InvalidProductError("o desconto precisa estar entre 0 e 100");
  }
  return createProduct({ ...product, price: product.price * (1 - percent / 100) });
}
