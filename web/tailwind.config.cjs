/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: {
          950: "#0b0f14",
          900: "#111723",
          800: "#1a2230",
          700: "#273244",
          200: "#d3dae6",
          100: "#e9edf4"
        },
        ember: {
          500: "#ff7a45",
          600: "#f95d2a"
        },
        signal: {
          500: "#2dd4bf",
          600: "#0ea5a4"
        },
        haze: {
          300: "#f6f1e6",
          400: "#efe6d6"
        }
      },
      fontFamily: {
        display: ["'Space Grotesk'", "sans-serif"],
        body: ["'IBM Plex Sans'", "sans-serif"]
      },
      boxShadow: {
        soft: "0 12px 40px rgba(15, 23, 42, 0.18)",
        glow: "0 0 0 1px rgba(45, 212, 191, 0.25), 0 20px 40px rgba(2, 132, 199, 0.25)"
      }
    }
  },
  plugins: []
};
