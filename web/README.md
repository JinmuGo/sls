# `sls` Minimalist Landing Page Website

A high-performance, ultra-minimalist, and responsive landing page for the `sls` (ssh ls) CLI tool, built with **Vite + Vanilla JS/HTML/CSS** for absolute speed, zero bloat, and clean styling control.

## Project Structure

```text
web/
├── index.html        # Main HTML layout, SEO meta tags, OpenGraph, & JSON-LD schema
├── package.json      # Dependencies and build scripts
├── package-lock.json # Locked dependency tree for deterministic CI builds
├── public/
│   ├── CNAME         # GitHub Pages custom domain mapping (sls.jinmu.me)
│   ├── favicon.svg   # Custom brand-aligned minimalist vector favicon
│   ├── robots.txt    # Search engine crawl rules referencing the sitemap
│   └── sitemap.xml   # Index sitemap mapping the landing page
├── src/
│   ├── main.js       # App entrypoint (GitHub stars fetcher & copy action)
│   └── style.css     # CSS Variables, dark theme tokens, layout design
└── README.md         # This documentation
```

## Features

1. **Static TUI Demo Frame**: Displays the real recorded `demo.gif` inside a mock terminal frame. It is responsive, highly optimized, and runs flawlessly on mobile, tablet, and desktop screens.
2. **One-Click Installation Copy**: Copies the Homebrew installation command to the clipboard with visual checkmark animations and instant user feedback.
3. **Dynamic GitHub Stars**: Non-blocking asynchronous query fetches the live star count directly from the `jinmugo/sls` repository with a sensible fallback.
4. **Comprehensive SEO, Social Graph & Location Metadata**:
   * Canonical URL tags and standard robot configuration.
   * Open Graph (Facebook/Discord/Slack) & Twitter Cards preview meta tags.
   * Custom minimalist preview card (`web/public/og-image.png`) generated for clean link previews.
   * Geotargeting tags (`geo.region`, `geo.position`, `ICBM`) anchoring the site locale.
   * Structured JSON-LD metadata declaring `SoftwareApplication` metrics to trigger Google rich snippets.
5. **Pre-composed Viral Share Loop**: Header includes a minimalist **Share** link that launches a pre-filled tweet detailing the tool and link.

## Development & Build

### Running Locally
To launch the hot-reloading development server:
```bash
cd web
npm install
npm run dev
```

### Production Build
To compile the static asset bundle:
```bash
npm run build
```
This resolves the `demo.gif` relative path, packages the assets, and outputs the production bundle to `web/dist/`.

---

## Deployment to `sls.jinmu.me`

The website is configured to deploy automatically to **GitHub Pages** using GitHub Actions:

### 1. Automated CI/CD Setup (`.github/workflows/deploy.yml`)
Pushing commits to the `main` branch that touch `web/`, the root `demo.gif`, or the workflow file itself triggers the deployment pipeline. The workflow:
* Installs dependencies via `npm ci`.
* Compiles the static assets via `npm run build`.
* Deploys the `web/dist` build output directory directly to GitHub Pages.

### 2. DNS Configuration (Cloudflare CNAME)
Ensure that you point a `CNAME` record in Cloudflare for the hostname `sls` to `<your-github-username>.github.io` (e.g. `jinmugo.github.io`). Turn on the proxy status (orange cloud) for DNS resolution and SSL handling.
