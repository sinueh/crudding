// API pública do módulo orders.
import { InMemoryCollection } from "../../shared/database.js";
import type { Route } from "../../shared/http/types.js";
import type { Order } from "./order.js";
import { OrderController } from "./order-controller.js";
import { orderRoutes } from "./order-routes.js";
import { type Catalog, OrderService } from "./order-service.js";

export interface OrdersModule {
  service: OrderService;
  routes: Route[];
}

export function createOrdersModule(deps: { catalog: Catalog }): OrdersModule {
  const service = new OrderService(new InMemoryCollection<Order>(), deps.catalog);
  return { service, routes: orderRoutes(new OrderController(service)) };
}
