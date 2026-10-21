// API pública do módulo catalog: outros módulos só importam deste arquivo.
import { InMemoryCollection } from "../../shared/database.js";
import type { Route } from "../../shared/http/types.js";
import { CatalogController } from "./catalog-controller.js";
import { catalogRoutes } from "./catalog-routes.js";
import { CatalogService } from "./catalog-service.js";
import type { Product } from "./product.js";

export type { CatalogService } from "./catalog-service.js";
export type { Product } from "./product.js";

export interface CatalogModule {
  service: CatalogService;
  routes: Route[];
}

export function createCatalogModule(): CatalogModule {
  const service = new CatalogService(new InMemoryCollection<Product>());
  return { service, routes: catalogRoutes(new CatalogController(service)) };
}
