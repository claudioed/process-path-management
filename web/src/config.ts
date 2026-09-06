/** Local-dev base URL for process-path-management's own REST API. Mirrors
 *  e2e-tests/env.sh's port-offset convention (each service's own :8080
 *  default, offset by index -- process-path-management is not yet wired
 *  into e2e-tests/env.sh, but 8087 continues that same sequence:
 *  FACILITY=8081, INVENTORY=8082, WES=8083, FULFILLMENT=8084,
 *  WORKFORCE=8085, ORDER=8086, PROCESS_PATH=8087). */
export const PROCESS_PATH_API_BASE = "http://localhost:8087";
