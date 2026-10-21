// Driving port (porta primária): capacidades de negócio expostas aos
// adaptadores de entrada (HTTP, CLI, filas...).

/**
 * @typedef {import("../domain/product.js").Product} Product
 *
 * @typedef {object} CreateProductInput
 * @property {string} name
 * @property {number} price
 * @property {number} stock
 *
 * @typedef {object} ProductService
 * @property {() => Promise<Product[]>} list
 * @property {(id: string) => Promise<Product>} get
 * @property {(input: CreateProductInput) => Promise<Product>} create
 * @property {(id: string, percent: number) => Promise<Product>} applyDiscount
 */

export {};
