// Coleção em memória usada pelos módulos como "tabela". Substitua por um
// cliente de banco real mantendo a mesma interface.
export class InMemoryCollection<T extends { id: number }> {
  #items = new Map<number, T>();
  #nextId = 1;

  insert(data: Omit<T, "id">): T {
    const item = { id: this.#nextId++, ...data } as T;
    this.#items.set(item.id, item);
    return item;
  }

  findAll(): T[] {
    return [...this.#items.values()];
  }

  findById(id: number): T | undefined {
    return this.#items.get(id);
  }

  update(item: T): T {
    this.#items.set(item.id, item);
    return item;
  }
}
