# MCP App viewer (bundled HTML)

This package builds the **single-file HTML** used by generated MCP servers for [MCP Apps](https://modelcontextprotocol.io/extensions/apps/overview)–capable hosts (e.g. [MCP Jam](https://www.mcpjam.com/), Claude).

The output is committed at `../embed/mcp-app-viewer.html` so `go generate` / `go build` can embed it without running Node in CI.

## Rebuild after editing `src/`

```bash
cd backend/internal/generator/mcpappviewer
npm install
npm run build
```

Then commit `backend/internal/generator/embed/mcp-app-viewer.html`.

## CSP in MCP Jam / Claude

Embedded hosts enforce CSP on MCP App HTML. The **generated server** declares `_meta.ui.csp.resourceDomains` on `resources/read` so external images (e.g. NASA APOD) can load. Defaults include common CDNs and `https://*.nasa.gov`.

To allow more origins at runtime (comma-separated origins):

```bash
export MCP_APP_RESOURCE_DOMAINS="https://cdn.mycompany.com,https://images.example.org"
```

See [CSP & CORS (MCP Apps)](https://apps.extensions.modelcontextprotocol.io/api/documents/csp-and-cors.html).

### `script-src: eval` (MCP Jam CSP badge)

The viewer bundles **Zod** (via MCP SDK) which, in v4, runs a tiny `new Function("")` probe on load. Strict CSP flags that as `eval`. The build runs `scripts/strip-csp-eval.mjs` after Vite to replace that probe so hosts do not report a false-positive CSP violation. If you change bundling and the build fails at that step, update the script’s regex to match the new minified output.
