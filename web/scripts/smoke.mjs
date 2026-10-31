// Playwright checks for the shipped shell. Deploy, HA, backups and migrations
// stay in prototype/scripts/smoke.mjs until those milestones land. The
// connection check is the shipped one-step form. The advisor check is the
// prototype's production confirmation.
import { spawn } from 'node:child_process';
import { existsSync } from 'node:fs';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { chromium } from 'playwright';

const root = path.resolve(import.meta.dirname, '../..');
const bin = path.join(root, 'bin/tuskline');
const port = process.env.TUSKLINE_SMOKE_PORT ?? '18080';
const base = `http://127.0.0.1:${port}/?demo=1`;
const failures = [];

function ok(name, value) {
	console.log(value ? 'PASS' : 'FAIL', name);
	if (!value) failures.push(name);
}

function chromePath() {
	const candidates = [
		process.env.CHROME_PATH,
		'/usr/bin/google-chrome',
		'/usr/bin/chromium',
		'/usr/bin/chromium-browser'
	].filter(Boolean);
	return candidates.find((candidate) => existsSync(candidate));
}

async function waitForHealth(child) {
	const deadline = Date.now() + 20_000;
	let stderr = '';
	child.stderr?.on('data', (chunk) => {
		stderr += chunk;
	});
	while (Date.now() < deadline) {
		if (child.exitCode !== null) break;
		try {
			const response = await fetch(`http://127.0.0.1:${port}/api/v1/health`);
			if (response.ok) return;
		} catch {
			/* still starting */
		}
		await new Promise((resolve) => setTimeout(resolve, 200));
	}
	throw new Error(`tuskline did not start on :${port}\n${stderr}`);
}

const dir = await mkdtemp(path.join(tmpdir(), 'tl-smoke-'));
const child = spawn(bin, ['serve', '-addr', `127.0.0.1:${port}`], {
	env: {
		...process.env,
		TUSKLINE_ADDR: `127.0.0.1:${port}`,
		TUSKLINE_INVENTORY: path.join(dir, 'inventory.yaml'),
		TUSKLINE_STORE: path.join(dir, 'meta.db')
	},
	stdio: ['ignore', 'ignore', 'pipe']
});

let browser;
try {
	await waitForHealth(child);
	browser = await chromium.launch({
		headless: true,
		executablePath: chromePath(),
		args: ['--no-sandbox', '--disable-dev-shm-usage']
	});
	const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
	const pageErrors = [];
	page.on('pageerror', (error) => pageErrors.push(String(error)));

	await page.goto(base);
	await page.getByRole('heading', { name: 'Fleet' }).waitFor();
	ok('fleet shows prod-eu-west', await page.getByText('prod-eu-west').first().isVisible());

	await page.keyboard.press('g');
	await page.keyboard.press('c');
	ok(
		'g c -> cluster health',
		await page.getByRole('heading', { name: 'prod-eu-west' }).isVisible()
	);

	await page.keyboard.press('g');
	await page.keyboard.press('q');
	ok('g q -> queries', await page.getByRole('heading', { name: 'Query insights' }).isVisible());

	await page.keyboard.press('t');
	ok('t -> light', await page.evaluate(() => !document.documentElement.classList.contains('dark')));
	await page.keyboard.press('t');
	ok('t -> dark', await page.evaluate(() => document.documentElement.classList.contains('dark')));

	await page.keyboard.press('e');
	await page.getByRole('button', { name: 'Staging', exact: true }).click();
	ok(
		'env switch -> staging',
		await page.getByRole('button', { name: 'staging-eu-west', exact: true }).isVisible()
	);

	await page.keyboard.press('g');
	await page.keyboard.press('f');
	ok('g f -> fleet', await page.getByRole('heading', { name: 'Fleet' }).isVisible());

	await page.keyboard.press('Control+k');
	await page.getByRole('dialog', { name: 'Command palette' }).waitFor();
	await page.keyboard.type('deploy');
	ok('palette lists deploy', await page.getByText('Deploy a new cluster…').isVisible());
	await page.keyboard.press('Escape');

	await page.keyboard.press('g');
	await page.keyboard.press('d');
	ok(
		'g d -> deploy placeholder',
		(await page.getByRole('heading', { name: 'Deploy cluster' }).isVisible()) &&
			(await page.getByText('later milestone').isVisible())
	);

	await page.keyboard.press('n');
	ok(
		'n -> add connection',
		(await page.getByRole('heading', { name: 'Add connection' }).isVisible()) &&
			(await page.getByRole('button', { name: 'verify-full' }).isVisible()) &&
			(await page.getByRole('button', { name: 'Create connection' }).isVisible())
	);
	await page.getByRole('button', { name: 'Create connection' }).click();
	await page.getByRole('textbox', { name: 'Filter objects' }).waitFor();
	ok(
		'create opens the database',
		await page.getByText('orders', { exact: true }).first().isVisible()
	);
	await page.evaluate(() => {
		const active = document.activeElement;
		if (active instanceof HTMLElement) active.blur();
	});

	await page.keyboard.press('g');
	await page.keyboard.press('h');
	ok('g h -> HA placeholder', await page.getByRole('heading', { name: 'HA topology' }).isVisible());

	await page.keyboard.press('g');
	await page.keyboard.press('s');
	await page.getByRole('textbox', { name: 'Filter objects' }).waitFor();
	await page.getByRole('button', { name: /^Run/ }).click();
	ok('sql run shows rows', await page.getByText(/12 rows · \d+ ms/).isVisible());
	await page.getByRole('button', { name: /^Analyse/ }).click();
	ok('sql analyse summarises the plan', await page.getByText(/returned 100 rows/).isVisible());

	await page.keyboard.press('g');
	await page.keyboard.press('f');
	await page
		.getByRole('button', { name: /prod-eu-west/ })
		.first()
		.click();
	await page.keyboard.press('g');
	await page.keyboard.press('v');
	ok(
		'g v -> config advisor',
		await page.getByRole('heading', { name: 'Config advisor' }).isVisible()
	);
	await page.getByRole('button', { name: /Apply \d+ to prod/ }).click();
	const confirm = page.getByRole('button', { name: 'Apply to production', exact: true });
	ok('advisor: prod apply disabled before confirmation', await confirm.isDisabled());
	await page.getByPlaceholder(/CHG-2041/).fill('CHG-1 test');
	await page.locator('.fixed input.mono').last().fill('prod-eu-west');
	ok('advisor: enabled after ticket + typed name', await confirm.isEnabled());
	await confirm.click();
	ok(
		'advisor: apply does not run the change',
		await page.getByText('Nothing was applied').isVisible()
	);

	ok('no page errors', pageErrors.length === 0);
	if (pageErrors.length) console.log(pageErrors.slice(0, 5).join('\n'));
} finally {
	await browser?.close();
	child.kill('SIGTERM');
	await rm(dir, { recursive: true, force: true });
}

if (failures.length) {
	console.error(`\n${failures.length} smoke check(s) failed`);
	process.exit(1);
}
