import { createCatalogModule } from "./modules/catalog/index.js";
import { createOrdersModule } from "./modules/orders/index.js";
import { startServer } from "./shared/http/server.js";

const catalog = createCatalogModule();
const orders = createOrdersModule({ catalog: catalog.service });

catalog.service.create({ sku: "KB-01", name: "Teclado Mecânico", price: 450, stock: 10 });
catalog.service.create({ sku: "MS-01", name: "Mouse Ergonômico", price: 189.9, stock: 25 });

await startServer([
  { method: "GET", path: "/health", handler: () => ({ status: 200, body: { status: "ok" } }) },
  ...catalog.routes,
  ...orders.routes,
]);
