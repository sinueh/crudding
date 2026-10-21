export function productRoutes(controller) {
  return [
    { method: "GET", path: "/health", handler: () => ({ status: 200, body: { status: "ok" } }) },
    { method: "GET", path: "/products", handler: controller.list },
    { method: "GET", path: "/products/:id", handler: controller.get },
    { method: "POST", path: "/products", handler: controller.create },
    { method: "POST", path: "/products/:id/discount", handler: controller.applyDiscount },
  ];
}
