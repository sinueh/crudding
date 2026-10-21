import { OrderController } from "../adapters/controllers/order-controller.js";
import { InMemoryOrderRepository } from "../adapters/repositories/in-memory-order-repository.js";
import { CreateOrder } from "../use-cases/create-order.js";
import { GetOrder } from "../use-cases/get-order.js";
import { ListOrders } from "../use-cases/list-orders.js";
import { PayOrder } from "../use-cases/pay-order.js";
import { orderRoutes } from "./http/routes.js";
import { startServer } from "./http/server.js";

// Composition root: a única camada que conhece todas as outras.
const orders = new InMemoryOrderRepository();
const controller = new OrderController({
  createOrder: new CreateOrder(orders),
  getOrder: new GetOrder(orders),
  listOrders: new ListOrders(orders),
  payOrder: new PayOrder(orders),
});

await startServer(orderRoutes(controller));
