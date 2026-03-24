/**
 * Zod v4 includes a runtime probe `new Function("")` to detect whether dynamic code
 * evaluation is available. MCP Apps hosts use a strict CSP without `unsafe-eval`, so
 * that probe fires a violation (blockedURI "eval") even though we never rely on JIT.
 *
 * Replace the probe with `return!1` so Zod treats eval as unavailable and uses safe paths.
 * Pattern is stable across Vite/esbuild minification for our single-file build.
 *
 * @see https://apps.extensions.modelcontextprotocol.io/api/documents/csp-and-cors.html
 */
import { readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const out = path.join(root, "embed", "mcp-app-viewer.html");

let s = readFileSync(out, "utf8");
const probe = /try\{const t=Function;return new t\(\"\"\),!0\}catch\{return!1\}/g;
const next = s.replace(probe, "return!1");
if (next === s) {
  console.error("strip-csp-eval: Zod Function probe pattern not found; update script if Vite output changed");
  process.exit(1);
}
writeFileSync(out, next);
console.log("strip-csp-eval: patched Zod eval probe in", out);
