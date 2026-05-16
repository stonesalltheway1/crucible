// Next.js config for the Crucible web console.
//
// The deterministic-bundling flags are mandatory for hermetic Nix builds —
// see docs/02-engineering/local-dev.md §"Web console build hermeticity".

const config = {
  reactStrictMode: true,
  poweredByHeader: false,
  experimental: {
    typedRoutes: true,
    serverActions: { bodySizeLimit: "1mb" },
    deterministicBundling: true,
  },
  // Crucible never embeds customer code/task content in third-party tools.
  // Plausible page-view analytics are first-party-proxied; no GA, no Segment.
  productionBrowserSourceMaps: false,
  compiler: {
    removeConsole: { exclude: ["error", "warn"] },
  },
  async headers() {
    return [
      {
        source: "/(.*)",
        headers: [
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "X-Frame-Options", value: "DENY" },
          { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
          { key: "Strict-Transport-Security", value: "max-age=31536000; includeSubDomains; preload" },
          {
            key: "Content-Security-Policy",
            value: [
              "default-src 'self'",
              "script-src 'self' 'unsafe-inline' https://clerk.crucible.dev https://*.workos.com",
              "style-src 'self' 'unsafe-inline'",
              "img-src 'self' data: https:",
              "connect-src 'self' https://api.crucible.dev https://*.clerk.crucible.dev https://*.workos.com wss://api.crucible.dev",
              "font-src 'self' data:",
              "frame-ancestors 'none'",
              "base-uri 'self'",
              "form-action 'self'",
            ].join("; "),
          },
        ],
      },
    ];
  },
};

export default config;
