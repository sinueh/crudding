import type { UserController } from "../controllers/user-controller.js";
import type { Route } from "../http/types.js";

export function userRoutes(controller: UserController): Route[] {
  return [
    { method: "GET", path: "/users", handler: controller.list },
    { method: "GET", path: "/users/:id", handler: controller.getById },
    { method: "POST", path: "/users", handler: controller.create },
  ];
}
