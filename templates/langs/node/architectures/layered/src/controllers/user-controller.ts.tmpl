import { parseNewUser } from "../models/user.js";
import type { UserService } from "../services/user-service.js";
import type { HttpRequest, HttpResponse } from "../http/types.js";
import { parseId, respond } from "./respond.js";

// Recebe a requisição, valida a entrada e delega para o UserService.
export class UserController {
  #service: UserService;

  constructor(service: UserService) {
    this.#service = service;
  }

  list = (): Promise<HttpResponse> => respond(() => this.#service.list());

  getById = (request: HttpRequest): Promise<HttpResponse> =>
    respond(() => this.#service.getById(parseId(request.params.id)));

  create = (request: HttpRequest): Promise<HttpResponse> =>
    respond(() => this.#service.create(parseNewUser(request.body)), 201);
}
