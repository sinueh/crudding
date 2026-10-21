import { ValidationError } from "./errors.js";

export interface User {
  id: number;
  name: string;
  email: string;
}

export interface NewUser {
  name: string;
  email: string;
}

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

// Valida o payload recebido e devolve um NewUser normalizado.
export function parseNewUser(input: unknown): NewUser {
  const data = (input ?? {}) as Record<string, unknown>;
  const name = typeof data.name === "string" ? data.name.trim() : "";
  const email = typeof data.email === "string" ? data.email.trim().toLowerCase() : "";

  if (name === "") {
    throw new ValidationError("name é obrigatório");
  }
  if (!EMAIL_PATTERN.test(email)) {
    throw new ValidationError("email inválido");
  }
  return { name, email };
}
