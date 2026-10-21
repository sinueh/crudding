export function orderRoutes(controller) {
  return [
    { method: "GET", path: "/orders", handler: controller.list },
    { method: "GET", path: "/orders/:id", handler: controller.getById },
    { method: "POST", path: "/orders", handler: controller.place },
  ];
}
