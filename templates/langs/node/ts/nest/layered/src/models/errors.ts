// Erros de negócio compartilhados pelas camadas. Os controllers os traduzem
// para status HTTP (veja controllers/respond).
export class ValidationError extends Error {
  override name = "ValidationError";
}

export class NotFoundError extends Error {
  override name = "NotFoundError";
}

export class ConflictError extends Error {
  override name = "ConflictError";
}
