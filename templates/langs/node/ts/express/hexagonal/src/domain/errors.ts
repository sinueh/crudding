// Erros do domínio: não conhecem HTTP nem banco. Os adaptadores decidem como
// apresentá-los.
export class DomainError extends Error {
  override name = "DomainError";
}

export class InvalidProductError extends DomainError {
  override name = "InvalidProductError";
}

export class ProductNotFoundError extends DomainError {
  override name = "ProductNotFoundError";

  constructor(id: string) {
    super(`produto ${id} não encontrado`);
  }
}
