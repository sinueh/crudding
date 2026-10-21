import { parseNewUser } from "../models/user.js";
import { parseId, respond } from "./respond.js";

// Recebe a requisição, valida a entrada e delega para o UserService.
export class UserController {
  #service;

  constructor(service) {
    this.#service = service;
  }

  list = () => respond(() => this.#service.list());

  getById = (request) => respond(() => this.#service.getById(parseId(request.params.id)));

  create = (request) => respond(() => this.#service.create(parseNewUser(request.body)), 201);
}
