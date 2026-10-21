import type { OrderOutput } from "../../use-cases/order-output.js";

const currency = new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL" });

export interface OrderViewModel extends Omit<OrderOutput, "total"> {
  total: { amount: number; formatted: string };
}

// Converte o DTO do caso de uso no formato que a API devolve ao cliente.
export function presentOrder(order: OrderOutput): OrderViewModel {
  return { ...order, total: { amount: order.total, formatted: currency.format(order.total) } };
}
