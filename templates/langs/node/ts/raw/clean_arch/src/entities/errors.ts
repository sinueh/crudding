// Erros das regras de negócio corporativas. Não dependem de nenhuma camada externa.
export class InvalidOrderError extends Error {
  override name = "InvalidOrderError";
}

export class OrderStateError extends Error {
  override name = "OrderStateError";
}
