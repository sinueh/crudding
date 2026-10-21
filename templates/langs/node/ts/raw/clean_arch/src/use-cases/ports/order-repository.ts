import type { Order } from "../../entities/order.js";

// Porta de saída definida pela camada de casos de uso. A implementação fica
// em adapters/repositories, respeitando a regra de dependência.
export interface OrderRepository {
  findAll(): Promise<Order[]>;
  findById(id: string): Promise<Order | null>;
  save(order: Order): Promise<void>;
}
