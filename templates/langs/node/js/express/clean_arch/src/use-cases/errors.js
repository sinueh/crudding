export class OrderNotFoundError extends Error {
  name = "OrderNotFoundError";

  constructor(id) {
    super(`pedido ${id} não encontrado`);
  }
}
