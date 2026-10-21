import assert from "node:assert/strict";
import { test } from "node:test";
import { TaskStore, parseTitle, taskRoutes } from "./tasks.js";

test("parseTitle normaliza e rejeita títulos inválidos", () => {
  assert.equal(parseTitle({ title: "  estudar node  " }), "estudar node");
  assert.equal(parseTitle({ title: "   " }), null);
  assert.equal(parseTitle({}), null);
  assert.equal(parseTitle(null), null);
});

test("TaskStore cria, conclui e remove tarefas", () => {
  const store = new TaskStore();
  const task = store.create("escrever testes");

  assert.equal(task.id, 1);
  assert.equal(store.complete(task.id)?.done, true);
  assert.equal(store.remove(task.id), true);
  assert.deepEqual(store.list(), []);
});

test("POST /tasks responde 400 sem título", async () => {
  const create = taskRoutes(new TaskStore()).find((route) => route.method === "POST");
  const response = await create!.handler({ params: {}, query: {}, body: {} });

  assert.equal(response.status, 400);
});
