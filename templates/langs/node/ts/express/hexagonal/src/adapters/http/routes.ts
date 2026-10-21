import type { ProductController } from "./product-controller.js";
import type { Route } from "./types.js";

export function productRoutes(controller: ProductController): Route[] {
  return [
    { method: "GET", path: "/health", handler: () => ({ status: 200, body: { status: "ok" } }) },
    { method: "GET", path: "/products", handler: controller.list },
    { method: "GET", path: "/products/:id", handler: controller.get },
    { method: "POST", path: "/products", handler: controller.create },
    { method: "POST", path: "/products/:id/discount", handler: controller.applyDiscount },
  ];
}
