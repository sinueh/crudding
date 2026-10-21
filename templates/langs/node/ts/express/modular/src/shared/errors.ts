// Erros reutilizados por todos os módulos.
export class ValidationError extends Error {
  override name = "ValidationError";
}

export class NotFoundError extends Error {
  override name = "NotFoundError";
}

export class ConflictError extends Error {
  override name = "ConflictError";
}
