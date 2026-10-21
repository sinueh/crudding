// API pública do módulo catalog: outros módulos só importam deste arquivo.
import { InMemoryCollection } from "../../shared/database.js";
import { CatalogController } from "./catalog-controller.js";
import { catalogRoutes } from "./catalog-routes.js";
import { CatalogService } from "./catalog-service.js";

export function createCatalogModule() {
  const service = new CatalogService(new InMemoryCollection());
  return { service, routes: catalogRoutes(new CatalogController(service)) };
}
