import { ConflictError, NotFoundError } from "../models/errors.js";

// Regras de negócio de usuários. Os dados ficam em memória nesta camada.
export class UserService {
  #users = new Map();
  #nextId = 1;

  list() {
    return [...this.#users.values()];
  }

  getById(id) {
    const user = this.#users.get(id);
    if (!user) {
      throw new NotFoundError(`usuário ${id} não encontrado`);
    }
    return user;
  }

  create(input) {
    if (this.list().some((user) => user.email === input.email)) {
      throw new ConflictError(`email ${input.email} já cadastrado`);
    }
    const user = { id: this.#nextId++, ...input };
    this.#users.set(user.id, user);
    return user;
  }
}
