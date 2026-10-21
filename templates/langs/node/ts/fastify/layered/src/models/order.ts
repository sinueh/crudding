import { ValidationError } from "./errors.js";

export type OrderStatus = "pending" | "paid" | "cancelled";

export interface OrderItem {
  product: string;
  quantity: number;
  unitPrice: number;
}

export interface Order {
  id: number;
  userId: number;
  items: OrderItem[];
  total: number;
  status: OrderStatus;
}

export interface NewOrder {
  userId: number;
  items: OrderItem[];
}

function parseItem(input: unknown): OrderItem {
  const data = (input ?? {}) as Record<string, unknown>;
  const product = typeof data.product === "string" ? data.product.trim() : "";
  const quantity = Number(data.quantity);
  const unitPrice = Number(data.unitPrice);

  if (product === "") {
    throw new ValidationError("item.product é obrigatório");
  }
  if (!Number.isInteger(quantity) || quantity <= 0) {
    throw new ValidationError("item.quantity deve ser um inteiro positivo");
  }
  if (!Number.isFinite(unitPrice) || unitPrice <= 0) {
    throw new ValidationError("item.unitPrice deve ser maior que zero");
  }
  return { product, quantity, unitPrice };
}

// Valida o payload recebido e devolve um NewOrder normalizado.
export function parseNewOrder(input: unknown): NewOrder {
  const data = (input ?? {}) as Record<string, unknown>;
  const userId = Number(data.userId);

  if (!Number.isInteger(userId) || userId <= 0) {
    throw new ValidationError("userId inválido");
  }
  if (!Array.isArray(data.items) || data.items.length === 0) {
    throw new ValidationError("o pedido precisa de ao menos um item");
  }
  return { userId, items: data.items.map(parseItem) };
}
