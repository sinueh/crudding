import { ValidationError } from "../../shared/errors.js";

export function parseNewProduct(input) {
  const data = input ?? {};
  const sku = typeof data.sku === "string" ? data.sku.trim().toUpperCase() : "";
  const name = typeof data.name === "string" ? data.name.trim() : "";
  const price = Number(data.price);
  const stock = Number(data.stock ?? 0);

  if (sku === "" || name === "") {
    throw new ValidationError("sku e name são obrigatórios");
  }
  if (!Number.isFinite(price) || price <= 0) {
    throw new ValidationError("price deve ser maior que zero");
  }
  if (!Number.isInteger(stock) || stock < 0) {
    throw new ValidationError("stock deve ser um inteiro não negativo");
  }
  return { sku, name, price, stock };
}
