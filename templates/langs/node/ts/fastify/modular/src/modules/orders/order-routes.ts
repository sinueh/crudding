import type { Route } from "../../shared/http/types.js";
import type { OrderController } from "./order-controller.js";

export function orderRoutes(controller: OrderController): Route[] {
  return [
    { method: "GET", path: "/orders", handler: controller.list },
    { method: "GET", path: "/orders/:id", handler: controller.getById },
    { method: "POST", path: "/orders", handler: controller.place },
  ];
}
