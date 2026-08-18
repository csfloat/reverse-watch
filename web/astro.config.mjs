// @ts-check
import { defineConfig } from 'astro/config';

// Static build: Astro emits a plain dist/ (index.html + hashed _astro/
// bundles + verbatim public/ assets) that the Go server serves via
// http.FileServer. No SSR adapter — the dashboard is fully client-rendered
// against the public /api/v1/* endpoints.
export default defineConfig({
  output: 'static',
});
