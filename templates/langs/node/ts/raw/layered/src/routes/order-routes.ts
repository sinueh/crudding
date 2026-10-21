import type { OrderController } from "../controllers/order-controller.js";
import type { Route } from "../http/types.js";

export function orderRoutes(controller: OrderController): Route[] {
  return [
    { method: "GET", path: "/orders", handler: controller.list },
    { method: "GET", path: "/orders/:id", handler: controller.getById },
    { method: "POST", path: "/orders", handler: controller.create },
    { method: "POST", path: "/orders/:id/pay", handler: controller.pay },
    { method: "POST", path: "/orders/:id/cancel", handler: controller.cancel },
  ];
}
