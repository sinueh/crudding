export function catalogRoutes(controller) {
  return [
    { method: "GET", path: "/products", handler: controller.list },
    { method: "GET", path: "/products/:id", handler: controller.getById },
    { method: "POST", path: "/products", handler: controller.create },
  ];
}
