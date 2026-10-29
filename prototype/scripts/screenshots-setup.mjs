// Setup & deploy screenshots (dark unless noted).
export default async function (page, file, out) {
  const go = async (q, hash, wait = 600, keep = false) => { if (!keep) await page.goto("about:blank"); await page.goto(file + q + hash); await page.waitForTimeout(wait); };
  const btn = (name) => page.getByRole("button", { name });
  const snap = (n) => page.screenshot({ path: out(n) });
  await go("?nosplash", "#/setup/prod-eu-west"); await snap("14-setup-overview");
  await go("?nosplash", "#/connect/prod-eu-west"); await snap("15-connect-details");
  await btn(/Test connection/).click(); await page.waitForTimeout(5200); await snap("16-connect-test");
  await btn(/^Continue/).click(); await page.waitForTimeout(400); await snap("17-connect-capabilities");
  await btn(/^Continue/).click(); await page.waitForTimeout(300);
  await btn(/verify now/).click(); await page.waitForTimeout(1400); await snap("18-connect-role");
  await btn(/^Continue/).click(); await page.waitForTimeout(300);
  await btn(/Add to Fleet/).click(); await page.waitForTimeout(700); await snap("19-fleet-new-connection");
  await go("?nosplash", "#/advisor/prod-eu-west"); await snap("20-config-advisor");
  await btn(/Apply \d+ to prod/).click(); await page.waitForTimeout(400); await snap("21-advisor-prod-guard");
  await page.keyboard.press("Escape");
  await go("?nosplash", "#/deploy/prod-eu-west"); await snap("22-deploy-target");
  await btn(/^Continue/).click(); await page.waitForTimeout(400); await snap("23-deploy-topology");
  await btn(/^Continue/).click(); await page.waitForTimeout(400); await snap("24-deploy-sizing");
  await btn(/^Continue/).click(); await page.waitForTimeout(2600); await snap("25-deploy-review");
  await btn(/^Deploy orders-demo/).click(); await page.waitForTimeout(3500); await snap("26-deploy-running");
  await btn(/Fast-forward/).click(); await page.waitForTimeout(2500); await page.mouse.wheel(0, -2000); await snap("27-deploy-complete");
  await btn(/View in Fleet/).click(); await page.waitForTimeout(4200); await snap("28-fleet-new-cluster");
  await page.keyboard.press("g"); await page.keyboard.press("l"); await page.waitForTimeout(500); await snap("29-deployments-history");
  await go("?nosplash&empty=1", "#/fleet/prod-eu-west"); await snap("30-fleet-empty-state");
  await go("?nosplash", "#/fleet/prod-eu-west", 400);
  await page.keyboard.press("Control+k"); await page.waitForTimeout(250); await page.keyboard.type("deploy"); await page.waitForTimeout(250); await snap("31-palette-setup");
  await page.keyboard.press("Escape");
  await go("?splash=hold", "#/fleet/prod-eu-west", 1200); await snap("32-splash");
  await go("?nosplash", "#/fleet/prod-eu-west", 400); await page.getByTitle("About Tuskline").first().click(); await page.waitForTimeout(400); await snap("33-about");
  await page.keyboard.press("Escape");
  await go("?nosplash&theme=light", "#/deploy/prod-eu-west"); await btn(/^Continue/).click(); await page.waitForTimeout(500); await snap("34-light-deploy-topology");
  await go("?nosplash&theme=light", "#/setup/prod-eu-west"); await snap("35-light-setup-overview");
}
