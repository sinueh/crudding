import type { HttpRequest, HttpResponse } from "../../shared/http/types.js";
import { parseId, respond } from "../../shared/respond.js";
import type { CatalogService } from "./catalog-service.js";
import { parseNewProduct } from "./product.js";

export class CatalogController {
  #service: CatalogService;

  constructor(service: CatalogService) {
    this.#service = service;
  }

  list = (): Promise<HttpResponse> => respond(() => this.#service.list());

  getById = (request: HttpRequest): Promise<HttpResponse> =>
    respond(() => this.#service.getById(parseId(request.params.id)));

  create = (request: HttpRequest): Promise<HttpResponse> =>
    respond(() => this.#service.create(parseNewProduct(request.body)), 201);
}
