import type { Config } from "tailwindcss";

// Crucible brand theme — anti-vibe-coding aesthetic.
//
// Senior-engineer-coded UI: tight type, monospace surfaces, ink palette, no
// rounded-corners-blue-gradient. The visual register signals "build receipts,
// not vibes." See docs/05-decisions/ADR-001-brand-voice.md.
const config: Config = {
  darkMode: ["class"],
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    container: {
      center: true,
      padding: "1.5rem",
      screens: { "2xl": "1400px" },
    },
    extend: {
      fontFamily: {
        sans: ["ui-sans-serif", "system-ui", "-apple-system", "Inter", "sans-serif"],
        mono: ["JetBrains Mono", "Berkeley Mono", "ui-monospace", "Menlo", "monospace"],
      },
      colors: {
        // Ink palette — high-contrast, low-saturation, paper-and-ink.
        ink: {
          50: "#f7f7f6",
          100: "#ececea",
          200: "#d8d8d3",
          300: "#b8b8b1",
          400: "#8f8f87",
          500: "#6c6c64",
          600: "#52524b",
          700: "#3d3d37",
          800: "#272723",
          900: "#161614",
          950: "#0a0a09",
        },
        // Semantic accents — used sparingly. A red-pen-on-page register.
        accent: {
          ok: "#1f7a3a", // verified green; muted, not lime
          warn: "#b76b00", // amber; muted
          alert: "#a3231f", // pen-red; not blood
          info: "#2f548a", // navy; not sky
        },
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        muted: { DEFAULT: "hsl(var(--muted))", foreground: "hsl(var(--muted-foreground))" },
        card: { DEFAULT: "hsl(var(--card))", foreground: "hsl(var(--card-foreground))" },
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        popover: { DEFAULT: "hsl(var(--popover))", foreground: "hsl(var(--popover-foreground))" },
        primary: { DEFAULT: "hsl(var(--primary))", foreground: "hsl(var(--primary-foreground))" },
        secondary: { DEFAULT: "hsl(var(--secondary))", foreground: "hsl(var(--secondary-foreground))" },
        destructive: { DEFAULT: "hsl(var(--destructive))", foreground: "hsl(var(--destructive-foreground))" },
      },
      borderRadius: {
        // Anti-vibe: 2px corners, not pills. Documents, not capsules.
        lg: "2px",
        md: "2px",
        sm: "1px",
      },
      boxShadow: {
        // Hard edges, not glow.
        ink: "0 1px 0 0 rgba(0,0,0,0.08), 0 0 0 1px rgba(0,0,0,0.04)",
        "ink-lg": "0 2px 0 0 rgba(0,0,0,0.12), 0 0 0 1px rgba(0,0,0,0.06)",
      },
      keyframes: {
        "fade-in": { from: { opacity: "0" }, to: { opacity: "1" } },
        "slide-in-bottom": {
          from: { transform: "translateY(4px)", opacity: "0" },
          to: { transform: "translateY(0)", opacity: "1" },
        },
      },
      animation: {
        "fade-in": "fade-in 120ms ease-out",
        "slide-in-bottom": "slide-in-bottom 160ms ease-out",
      },
    },
  },
  plugins: [require("tailwindcss-animate")],
};

export default config;
