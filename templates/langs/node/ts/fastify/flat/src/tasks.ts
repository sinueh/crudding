import type { HttpRequest, HttpResponse, Route } from "./types.js";

export interface Task {
  id: number;
  title: string;
  done: boolean;
  createdAt: string;
}

// Armazena as tarefas em memória. Troque por um banco quando o projeto crescer.
export class TaskStore {
  #tasks = new Map<number, Task>();
  #nextId = 1;

  list(): Task[] {
    return [...this.#tasks.values()];
  }

  get(id: number): Task | undefined {
    return this.#tasks.get(id);
  }

  create(title: string): Task {
    const task: Task = { id: this.#nextId++, title, done: false, createdAt: new Date().toISOString() };
    this.#tasks.set(task.id, task);
    return task;
  }

  complete(id: number): Task | undefined {
    const task = this.#tasks.get(id);
    if (task) {
      task.done = true;
    }
    return task;
  }

  remove(id: number): boolean {
    return this.#tasks.delete(id);
  }
}

// Valida o corpo de criação e devolve o título normalizado, ou null se inválido.
export function parseTitle(body: unknown): string | null {
  if (typeof body !== "object" || body === null) {
    return null;
  }
  const { title } = body as { title?: unknown };
  if (typeof title !== "string" || title.trim() === "") {
    return null;
  }
  return title.trim();
}

function found(task: Task | undefined): HttpResponse {
  return task ? { status: 200, body: task } : { status: 404, body: { error: "tarefa não encontrada" } };
}

function taskId(request: HttpRequest): number {
  return Number(request.params.id);
}

export function taskRoutes(store: TaskStore): Route[] {
  return [
    { method: "GET", path: "/tasks", handler: () => ({ status: 200, body: store.list() }) },
    { method: "GET", path: "/tasks/:id", handler: (request) => found(store.get(taskId(request))) },
    {
      method: "POST",
      path: "/tasks",
      handler: (request) => {
        const title = parseTitle(request.body);
        if (title === null) {
          return { status: 400, body: { error: "title é obrigatório" } };
        }
        return { status: 201, body: store.create(title) };
      },
    },
    { method: "PATCH", path: "/tasks/:id/done", handler: (request) => found(store.complete(taskId(request))) },
    {
      method: "DELETE",
      path: "/tasks/:id",
      handler: (request) =>
        store.remove(taskId(request)) ? { status: 204 } : { status: 404, body: { error: "tarefa não encontrada" } },
    },
  ];
}
