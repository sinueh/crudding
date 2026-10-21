import type { OrderController } from "../controllers/order-controller.js";
import type { UserController } from "../controllers/user-controller.js";
import type { Route } from "../http/types.js";
import { orderRoutes } from "./order-routes.js";
import { userRoutes } from "./user-routes.js";

export interface Controllers {
  users: UserController;
  orders: OrderController;
}

// Router central: junta as rotas de todos os recursos da aplicação.
export function createRoutes(controllers: Controllers): Route[] {
  return [
    { method: "GET", path: "/health", handler: () => ({ status: 200, body: { status: "ok" } }) },
    ...userRoutes(controllers.users),
    ...orderRoutes(controllers.orders),
  ];
}
