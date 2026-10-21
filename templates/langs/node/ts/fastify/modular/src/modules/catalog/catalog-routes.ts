import type { Route } from "../../shared/http/types.js";
import type { CatalogController } from "./catalog-controller.js";

export function catalogRoutes(controller: CatalogController): Route[] {
  return [
    { method: "GET", path: "/products", handler: controller.list },
    { method: "GET", path: "/products/:id", handler: controller.getById },
    { method: "POST", path: "/products", handler: controller.create },
  ];
}
