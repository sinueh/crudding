import { parseId, respond } from "../../shared/respond.js";
import { parseNewProduct } from "./product.js";

export class CatalogController {
  #service;

  constructor(service) {
    this.#service = service;
  }

  list = () => respond(() => this.#service.list());

  getById = (request) => respond(() => this.#service.getById(parseId(request.params.id)));

  create = (request) => respond(() => this.#service.create(parseNewProduct(request.body)), 201);
}
