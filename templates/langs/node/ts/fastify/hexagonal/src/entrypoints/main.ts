import { ProductController } from "../adapters/http/product-controller.js";
import { productRoutes } from "../adapters/http/routes.js";
import { startServer } from "../adapters/http/server.js";
import { InMemoryProductRepository } from "../adapters/persistence/in-memory-product-repository.js";
import { ProductManager } from "../domain/product-manager.js";

// Composition root: conecta adaptadores às portas.
const repository = new InMemoryProductRepository(); // driven adapter
const service = new ProductManager(repository); // núcleo da aplicação
const controller = new ProductController(service); // driving adapter

await service.create({ name: "Headset 7.1 Surround", price: 320, stock: 15 });

await startServer(productRoutes(controller));
