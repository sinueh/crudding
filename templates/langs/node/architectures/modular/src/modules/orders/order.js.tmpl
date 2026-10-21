import { ValidationError } from "../../shared/errors.js";

export function parsePlaceOrder(input) {
  const data = input ?? {};
  if (!Array.isArray(data.items) || data.items.length === 0) {
    throw new ValidationError("o pedido precisa de ao menos um item");
  }
  return data.items.map((raw) => {
    const item = raw ?? {};
    const productId = Number(item.productId);
    const quantity = Number(item.quantity);
    if (!Number.isInteger(productId) || productId <= 0) {
      throw new ValidationError("item.productId inválido");
    }
    if (!Number.isInteger(quantity) || quantity <= 0) {
      throw new ValidationError("item.quantity deve ser um inteiro positivo");
    }
    return { productId, quantity };
  });
}
