// CSS-variable colours that still honour Tailwind opacity modifiers (e.g. border-line/60) via color-mix.
const v = (name) => ({ opacityValue }) =>
  opacityValue === undefined || opacityValue === "1" ? `var(--${name})` : `color-mix(in srgb, var(--${name}) calc(${opacityValue} * 100%), transparent)`;
/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        bg: v("bg"), elev: v("elev"), subtle: v("subtle"), hover: v("hover"),
        line: v("line"), line2: v("line2"),
        fg: v("fg"), fg2: v("fg2"), fg3: v("fg3"),
        accent: v("accent"), prod: v("prod"), staging: v("staging"), dev: v("dev"),
        ok: v("ok"), warn: v("warn"), crit: v("crit"),
      },
      fontFamily: {
        sans: ['"Inter Variable"', "Inter", "system-ui", "sans-serif"],
        mono: ['"JetBrains Mono"', "ui-monospace", "monospace"],
      },
      fontSize: { "2xs": ["11px", "16px"] },
      keyframes: {
        fadeIn: { from: { opacity: "0" }, to: { opacity: "1" } },
        pop: { from: { opacity: "0", transform: "translateY(6px) scale(.98)" }, to: { opacity: "1", transform: "none" } },
        slideIn: { from: { opacity: "0", transform: "translateX(12px)" }, to: { opacity: "1", transform: "none" } },
      },
      animation: { fadeIn: "fadeIn .15s ease-out", pop: "pop .18s cubic-bezier(.2,.8,.2,1)", slideIn: "slideIn .2s ease-out" },
    },
  },
  plugins: [],
};
