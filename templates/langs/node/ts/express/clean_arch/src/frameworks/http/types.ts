// Contrato HTTP independente de framework. Controllers recebem um HttpRequest
// e devolvem um HttpResponse; o adaptador em server.ts traduz isso para o
// framework escolhido.
export type HttpMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

export interface HttpRequest {
  params: Record<string, string>;
  query: Record<string, unknown>;
  body: unknown;
}

export interface HttpResponse {
  status: number;
  body?: unknown;
}

export type HttpHandler = (request: HttpRequest) => HttpResponse | Promise<HttpResponse>;

export interface Route {
  method: HttpMethod;
  path: string;
  handler: HttpHandler;
}
