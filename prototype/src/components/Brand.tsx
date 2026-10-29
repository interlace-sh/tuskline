// Tusk Arc brand marks (concept 1). Geometry mirrors /workspace/tuskline-brand/svg.
import { useApp } from "../store";

export const BRAND = { ink: "#0E1116", paper: "#FAFAF9", accent: "#3E63DD", accentDark: "#7B96FF" };
export const WORDMARK_PATH = "M14.8 -22.7V-18H0.4V-22.7ZM4.3 -28.9H10.2V-6.8Q10.2 -5.6 10.7 -5.1Q11.1 -4.7 12.4 -4.7Q13 -4.7 13.7 -4.7Q14.4 -4.7 14.8 -4.7V0Q14.3 0 13.3 0Q12.3 0 11.4 0Q7.7 0 6 -1.4Q4.3 -2.9 4.3 -5.9ZM25.1 0.5Q22.7 0.5 20.9 -0.6Q19 -1.6 18 -3.6Q17 -5.6 17 -8.5V-22.7H22.9V-9.5Q22.9 -7.1 24.1 -5.9Q25.2 -4.6 27.3 -4.6Q28.6 -4.6 29.7 -5.2Q30.8 -5.8 31.4 -6.9Q32 -8.1 32 -10V-22.7H37.9V0H32.1V-5.8H33.2Q32.1 -2.8 30.1 -1.2Q28.1 0.5 25.1 0.5ZM50.6 0.5Q47.7 0.5 45.5 -0.3Q43.2 -1.2 41.9 -2.9Q40.5 -4.6 40.3 -7.1H46.1Q46.3 -5.5 47.4 -4.6Q48.5 -3.8 50.5 -3.8Q52.3 -3.8 53.4 -4.5Q54.5 -5.2 54.5 -6.3Q54.5 -7.3 53.7 -7.9Q52.9 -8.5 51.5 -8.7L47.5 -9.5Q44.3 -10.2 42.6 -11.9Q40.9 -13.5 40.9 -16.1Q40.9 -18.2 42 -19.8Q43.2 -21.4 45.3 -22.3Q47.4 -23.2 50.2 -23.2Q53.1 -23.2 55.2 -22.2Q57.4 -21.3 58.6 -19.6Q59.8 -17.9 59.9 -15.6H54.4Q54.4 -17 53.3 -17.9Q52.2 -18.9 50.4 -18.9Q48.7 -18.9 47.7 -18.2Q46.7 -17.4 46.7 -16.3Q46.7 -15.4 47.5 -14.8Q48.2 -14.2 49.6 -13.9L53.8 -13Q57.2 -12.3 58.8 -10.8Q60.4 -9.3 60.4 -6.8Q60.4 -4.6 59.2 -2.9Q57.9 -1.3 55.7 -0.4Q53.5 0.5 50.6 0.5ZM68.2 -6.8V-13.7H68.8L77.2 -22.7H84.2L73.7 -11.4H72.5ZM62.9 0V-32H68.7V0ZM77.7 0 70.2 -10.1 74.1 -14.2 84.7 0ZM91.5 -32V0H85.6V-32ZM95.1 0V-22.7H100.9V0ZM98 -25.8Q96.5 -25.8 95.5 -26.7Q94.6 -27.7 94.6 -29Q94.6 -30.4 95.5 -31.3Q96.5 -32.3 98 -32.3Q99.5 -32.3 100.4 -31.3Q101.4 -30.4 101.4 -29Q101.4 -27.6 100.4 -26.7Q99.5 -25.8 98 -25.8ZM110.4 -12.7V0H104.5V-22.7H110.3V-16.8H109.2Q110.3 -19.9 112.3 -21.5Q114.3 -23.2 117.3 -23.2Q119.7 -23.2 121.5 -22.1Q123.3 -21.1 124.3 -19.1Q125.3 -17.1 125.3 -14.2V0H119.5V-13.2Q119.5 -15.6 118.3 -16.8Q117.2 -18 115.1 -18Q113.7 -18 112.7 -17.5Q111.6 -16.9 111 -15.7Q110.4 -14.6 110.4 -12.7ZM139.1 0.5Q135.6 0.5 133.1 -1Q130.6 -2.5 129.2 -5.2Q127.9 -7.9 127.9 -11.3Q127.9 -14.8 129.3 -17.5Q130.7 -20.2 133.1 -21.7Q135.6 -23.2 138.8 -23.2Q141.3 -23.2 143.3 -22.3Q145.4 -21.5 146.8 -19.9Q148.3 -18.3 149.1 -16.2Q149.9 -14 149.9 -11.4V-9.8H130.1V-13.7H147L144.3 -12.7Q144.3 -14.6 143.7 -15.9Q143 -17.2 141.8 -17.9Q140.6 -18.6 139 -18.6Q137.3 -18.6 136.1 -17.9Q134.9 -17.2 134.2 -15.9Q133.6 -14.6 133.6 -12.7V-10.2Q133.6 -8.3 134.2 -6.9Q134.9 -5.5 136.2 -4.8Q137.4 -4 139.2 -4Q140.5 -4 141.5 -4.4Q142.5 -4.8 143.2 -5.5Q143.9 -6.1 144.2 -7.1H149.6Q149.2 -4.8 147.7 -3.1Q146.2 -1.4 144 -0.5Q141.8 0.5 139.1 0.5Z";
const TUSK = "M13 45.5 C13 27.5 26.5 13.5 51 12 C34 19 23 30.5 23 45.5 A5 5 0 0 1 13 45.5 Z";
const FAV_TUSK = "M11 47 C11 27 25.5 12 53 10.5 C34 18.5 23 30.5 23 47 A6 6 0 0 1 11 47 Z";

export type Variant = "light" | "dark";
/** App-icon tile. light = accent tile, white glyph; dark = ink tile, paper glyph, blue node. */
export function TuskMark({ size = 24, variant, className = "" }: { size?: number; variant?: Variant; className?: string }) {
  const { theme } = useApp();
  const v = variant ?? theme;
  const fg = v === "light" ? "#FFFFFF" : BRAND.paper;
  const ac = v === "light" ? "#FFFFFF" : BRAND.accentDark;
  return (
    <svg width={size} height={size} viewBox="0 0 64 64" className={`shrink-0 ${className}`} role="img" aria-label="Tuskline">
      <rect width="64" height="64" rx="14" fill={v === "light" ? BRAND.accent : BRAND.ink} />
      {v === "dark" && <rect x="0.75" y="0.75" width="62.5" height="62.5" rx="13.25" fill="none" stroke="#2A303A" strokeWidth="1.5" />}
      <path d={TUSK} fill={fg} />
      <path d="M30 45.5 H40.5" stroke={fg} strokeWidth="5" strokeLinecap="round" fill="none" />
      <circle cx="48.5" cy="45.5" r="4.25" fill={ac} />
    </svg>
  );
}
/** Bare glyph (no tile), for inline use on coloured surfaces. */
export function TuskGlyph({ size = 16, color = "currentColor", dot = "var(--accent)" }: { size?: number; color?: string; dot?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 64 64" className="shrink-0" aria-hidden>
      <path d={FAV_TUSK} fill={color} /><circle cx="45" cy="47" r="6" fill={dot} />
    </svg>
  );
}
export function Wordmark({ height = 16, color = "var(--fg)" }: { height?: number; color?: string }) {
  return (
    <svg height={height} viewBox="0 -33 152 34" style={{ width: (height * 152) / 34 }} className="shrink-0" role="img" aria-label="tuskline">
      <path d={WORDMARK_PATH} fill={color} />
    </svg>
  );
}
export function Lockup({ size = 24, variant }: { size?: number; variant?: Variant }) {
  return <span className="inline-flex items-center gap-2.5"><TuskMark size={size} variant={variant} /><Wordmark height={size * 0.62} /></span>;
}
export function faviconSvg(theme: Variant) {
  const tile = theme === "light" ? BRAND.accent : BRAND.ink;
  const dot = theme === "light" ? "#FFFFFF" : BRAND.accentDark;
  const fg = theme === "light" ? "#FFFFFF" : BRAND.paper;
  return `<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 64 64'><rect width='64' height='64' rx='14' fill='${tile}'/><path d='${FAV_TUSK}' fill='${fg}'/><circle cx='45' cy='47' r='6' fill='${dot}'/></svg>`;
}
