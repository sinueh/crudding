export class OrderNotFoundError extends Error {
  override name = "OrderNotFoundError";

  constructor(id: string) {
    super(`pedido ${id} não encontrado`);
  }
}
