import { orderRoutes } from "./order-routes.js";
import { userRoutes } from "./user-routes.js";

// Router central: junta as rotas de todos os recursos da aplicação.
export function createRoutes(controllers) {
  return [
    { method: "GET", path: "/health", handler: () => ({ status: 200, body: { status: "ok" } }) },
    ...userRoutes(controllers.users),
    ...orderRoutes(controllers.orders),
  ];
}
