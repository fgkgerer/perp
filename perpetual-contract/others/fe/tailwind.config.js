/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        page: "#050809",
        surface: "#080c0e",
        elevated: "#101618",
        panel: "#0a0f11",
        panelBorder: "#1d272b",
        foreground: "#f1f4f4",
        muted: "#a9b3b5",
        subtle: "#747f82",
        faint: "#505b5e",
        accent: "#16d8d4",
        buy: "#2ecb8b",
        sell: "#f05b78"
      }
    }
  },
  plugins: []
};
