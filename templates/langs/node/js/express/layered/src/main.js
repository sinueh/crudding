import { OrderController } from "./controllers/order-controller.js";
import { UserController } from "./controllers/user-controller.js";
import { startServer } from "./http/server.js";
import { createRoutes } from "./routes/index.js";
import { OrderService } from "./services/order-service.js";
import { UserService } from "./services/user-service.js";

const userService = new UserService();
const orderService = new OrderService(userService);

userService.create({ name: "Ada Lovelace", email: "ada@example.com" });

const routes = createRoutes({
  users: new UserController(userService),
  orders: new OrderController(orderService),
});

await startServer(routes);
