// Coleção em memória usada pelos módulos como "tabela". Substitua por um
// cliente de banco real mantendo a mesma interface.
export class InMemoryCollection {
  #items = new Map();
  #nextId = 1;

  insert(data) {
    const item = { id: this.#nextId++, ...data };
    this.#items.set(item.id, item);
    return item;
  }

  findAll() {
    return [...this.#items.values()];
  }

  findById(id) {
    return this.#items.get(id);
  }

  update(item) {
    this.#items.set(item.id, item);
    return item;
  }
}
