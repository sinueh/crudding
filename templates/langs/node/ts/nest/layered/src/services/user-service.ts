import { ConflictError, NotFoundError } from "../models/errors.js";
import type { NewUser, User } from "../models/user.js";

// Regras de negócio de usuários. Os dados ficam em memória nesta camada.
export class UserService {
  #users = new Map<number, User>();
  #nextId = 1;

  list(): User[] {
    return [...this.#users.values()];
  }

  getById(id: number): User {
    const user = this.#users.get(id);
    if (!user) {
      throw new NotFoundError(`usuário ${id} não encontrado`);
    }
    return user;
  }

  create(input: NewUser): User {
    if (this.list().some((user) => user.email === input.email)) {
      throw new ConflictError(`email ${input.email} já cadastrado`);
    }
    const user: User = { id: this.#nextId++, ...input };
    this.#users.set(user.id, user);
    return user;
  }
}
