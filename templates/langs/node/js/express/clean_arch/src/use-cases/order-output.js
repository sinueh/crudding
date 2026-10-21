// DTO de saída: os casos de uso nunca expõem a entidade para fora.
export function toOrderOutput(order) {
  return { ...order.toSnapshot(), total: order.total };
}
