import { ValidationError } from "../../shared/errors.js";

export interface OrderLine {
  productId: number;
  name: string;
  quantity: number;
  unitPrice: number;
}

export interface Order {
  id: number;
  lines: OrderLine[];
  total: number;
  createdAt: string;
}

export interface PlaceOrderItem {
  productId: number;
  quantity: number;
}

export function parsePlaceOrder(input: unknown): PlaceOrderItem[] {
  const data = (input ?? {}) as Record<string, unknown>;
  if (!Array.isArray(data.items) || data.items.length === 0) {
    throw new ValidationError("o pedido precisa de ao menos um item");
  }
  return data.items.map((raw: unknown) => {
    const item = (raw ?? {}) as Record<string, unknown>;
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
