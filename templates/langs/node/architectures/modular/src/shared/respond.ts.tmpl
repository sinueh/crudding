import { ConflictError, NotFoundError, ValidationError } from "./errors.js";
import type { HttpResponse } from "./http/types.js";

// Executa a ação do controller e traduz erros de negócio em status HTTP.
export async function respond(action: () => unknown, status = 200): Promise<HttpResponse> {
  try {
    return { status, body: await action() };
  } catch (error) {
    if (error instanceof ValidationError) {
      return { status: 400, body: { error: error.message } };
    }
    if (error instanceof NotFoundError) {
      return { status: 404, body: { error: error.message } };
    }
    if (error instanceof ConflictError) {
      return { status: 409, body: { error: error.message } };
    }
    throw error;
  }
}

export function parseId(value: string | undefined): number {
  const id = Number(value);
  if (!Number.isInteger(id) || id <= 0) {
    throw new ValidationError(`id inválido: ${value}`);
  }
  return id;
}
