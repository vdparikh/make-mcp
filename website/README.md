# Make MCP website

Static site for [Make MCP](https://github.com/vdparikh/make-mcp): Mintlify-style docs layout (sidebar, sticky top bar, “On this page” TOC) with the same Lexend font and blue palette as the web app (`App.css`).

## Deploy to GitHub Pages

1. In the repo: **Settings → Pages**
2. Under **Build and deployment**, set **Source** to **GitHub Actions**
3. Push to `main` (or run the workflow manually). The workflow deploys the `website/` folder to GitHub Pages.

The site will be available at **https://vdparikh.github.io/make-mcp/** (or your custom domain if configured).

## Local preview

Open `index.html` in a browser, or serve the folder:

```bash
cd website && python3 -m http.server 8080
# or: npx serve website
```

Then open http://localhost:8080
