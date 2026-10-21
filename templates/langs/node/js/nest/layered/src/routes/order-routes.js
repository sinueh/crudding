export function orderRoutes(controller) {
  return [
    { method: "GET", path: "/orders", handler: controller.list },
    { method: "GET", path: "/orders/:id", handler: controller.getById },
    { method: "POST", path: "/orders", handler: controller.create },
    { method: "POST", path: "/orders/:id/pay", handler: controller.pay },
    { method: "POST", path: "/orders/:id/cancel", handler: controller.cancel },
  ];
}
