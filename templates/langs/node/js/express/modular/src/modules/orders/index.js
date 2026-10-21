// API pública do módulo orders.
import { InMemoryCollection } from "../../shared/database.js";
import { OrderController } from "./order-controller.js";
import { orderRoutes } from "./order-routes.js";
import { OrderService } from "./order-service.js";

export function createOrdersModule(deps) {
  const service = new OrderService(new InMemoryCollection(), deps.catalog);
  return { service, routes: orderRoutes(new OrderController(service)) };
}
