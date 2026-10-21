export function orderRoutes(controller) {
  return [
    { method: "GET", path: "/health", handler: () => ({ status: 200, body: { status: "ok" } }) },
    { method: "GET", path: "/orders", handler: controller.list },
    { method: "GET", path: "/orders/:id", handler: controller.get },
    { method: "POST", path: "/orders", handler: controller.create },
    { method: "POST", path: "/orders/:id/pay", handler: controller.pay },
  ];
}
