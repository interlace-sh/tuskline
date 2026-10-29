import { chromium } from "playwright-core";
import path from "node:path";
const file = "file://" + path.resolve("tuskline-prototype.html");
const shots = [
  ["01-fleet", "#/fleet/prod-eu-west"],
  ["02-cluster-health", "#/cluster/prod-eu-west"],
  ["03-query-insights", "#/queries/prod-eu-west"],
  ["04-sql-editor-prod-readonly", "#/sql/prod-eu-west"],
  ["05-ha-topology", "#/ha/prod-eu-west"],
  ["06-backups-pitr", "#/backups/prod-eu-west"],
  ["07-migrations", "#/migrations/prod-eu-west"],
];
const browser = await chromium.launch({ executablePath: "/usr/bin/google-chrome", args: ["--no-sandbox"] });
const page = await browser.newPage({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 1 });
const errors = [];
page.on("pageerror", (e) => errors.push(String(e)));
page.on("console", (m) => { if (m.type() === "error") errors.push(m.text()); });
const out = (n) => `screenshots/${n}.png`;
for (const [name, hash] of shots) {
  await page.goto(file + "?nosplash" + hash);
  await page.waitForTimeout(700);
  await page.screenshot({ path: out(name) });
}
// interactions
await page.goto(file + "?nosplash#/fleet/prod-eu-west"); await page.waitForTimeout(400);
await page.keyboard.press("Control+k"); await page.waitForTimeout(250); await page.keyboard.type("switch"); await page.waitForTimeout(250);
await page.screenshot({ path: out("08-command-palette") });
await page.keyboard.press("Escape");
await page.goto(file + "?nosplash#/ha/prod-eu-west"); await page.waitForTimeout(400);
await page.getByRole("button", { name: /Switchover…/ }).click(); await page.waitForTimeout(1800);
await page.screenshot({ path: out("09-switchover-modal") });
await page.goto(file + "?nosplash#/sql/prod-eu-west"); await page.waitForTimeout(400);
await page.getByText("fix-status.sql").click(); await page.waitForTimeout(150);
await page.getByRole("button", { name: /^Run/ }).click(); await page.waitForTimeout(700);
await page.screenshot({ path: out("10-sql-write-blocked") });
await page.goto(file + "?nosplash&theme=light#/cluster/prod-eu-west"); await page.waitForTimeout(700);
await page.screenshot({ path: out("11-light-cluster-health") });
await page.goto(file + "?nosplash&theme=light#/fleet/prod-eu-west"); await page.waitForTimeout(700);
await page.screenshot({ path: out("12-light-fleet") });
await import("./screenshots-setup.mjs").then((m) => m.default(page, file, out));
await browser.close();
console.log("errors:", errors.length ? errors : "none");
