const currency = new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL" });

// Converte o DTO do caso de uso no formato que a API devolve ao cliente.
export function presentOrder(order) {
  return { ...order, total: { amount: order.total, formatted: currency.format(order.total) } };
}
