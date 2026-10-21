export function userRoutes(controller) {
  return [
    { method: "GET", path: "/users", handler: controller.list },
    { method: "GET", path: "/users/:id", handler: controller.getById },
    { method: "POST", path: "/users", handler: controller.create },
  ];
}
