import { startServer } from "./server.js";
import { TaskStore, taskRoutes } from "./tasks.js";

const store = new TaskStore();
store.create("Explorar o projeto gerado pelo crudding");

await startServer(taskRoutes(store));
