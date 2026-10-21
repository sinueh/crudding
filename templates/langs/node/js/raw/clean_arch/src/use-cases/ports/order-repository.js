// Porta de saída definida pela camada de casos de uso. A implementação fica
// em adapters/repositories, respeitando a regra de dependência.

/**
 * @typedef {import("../../entities/order.js").Order} Order
 *
 * @typedef {object} OrderRepository
 * @property {() => Promise<Order[]>} findAll
 * @property {(id: string) => Promise<Order | null>} findById
 * @property {(order: Order) => Promise<void>} save
 */

export {};
