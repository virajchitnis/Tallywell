import { test } from '@playwright/test';
import { spawn, ChildProcess } from 'node:child_process';
import { mkdtempSync, mkdirSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

const SCREENSHOTS_DIR = process.env.SCREENSHOTS_DIR
  ? resolve(process.env.SCREENSHOTS_DIR)
  : resolve(__dirname, '..', 'screenshots');

function startApp(): Promise<{ proc: ChildProcess; url: string }> {
  const bin = process.env.TALLYWELL_BIN;
  if (!bin) throw new Error('set TALLYWELL_BIN to the built binary path');
  const home = mkdtempSync(join(tmpdir(), 'tallywell-shots-'));
  return new Promise((res, rej) => {
    const proc = spawn(bin, [], { env: { ...process.env, HOME: home, XDG_CONFIG_HOME: join(home, '.config'), TALLYWELL_NO_TRAY: '1' } });
    let out = '';
    const onData = (d: Buffer) => {
      out += d.toString();
      const m = out.match(/http:\/\/127\.0\.0\.1:\d+\//);
      if (m) { proc.stdout?.off('data', onData); res({ proc, url: m[0] }); }
    };
    proc.stdout?.on('data', onData);
    proc.on('error', rej);
    setTimeout(() => rej(new Error('app did not start within 10 s')), 10_000);
  });
}

async function shot(page: import('@playwright/test').Page, name: string) {
  await page.screenshot({ path: join(SCREENSHOTS_DIR, name) });
}

test.setTimeout(120_000);

test('generate website screenshots', async ({ page }) => {
  mkdirSync(SCREENSHOTS_DIR, { recursive: true });
  await page.setViewportSize({ width: 1280, height: 800 });

  const { proc, url } = await startApp();
  try {
    // Setup screen
    await page.goto(url);
    await page.waitForSelector('h1');
    await shot(page, 'setup.png');

    await page.fill('input[name="passphrase"]', 'demo-passphrase-2026');
    await page.fill('input[name="confirm"]', 'demo-passphrase-2026');
    await page.click('button[type="submit"]');
    await page.waitForSelector('.cards');

    // Unlock screen (lock first, then show clean unlock)
    await page.click('form[action="/lock"] button');
    await page.waitForSelector('input[name="passphrase"]');
    await shot(page, 'unlock.png');

    // Wrong passphrase — shows the error flash
    await page.fill('input[name="passphrase"]', 'wrong-passphrase');
    await page.click('button[type="submit"]');
    await page.waitForSelector('.flash');
    await shot(page, 'wrong-passphrase.png');

    // Correct passphrase — unlock for real
    await page.fill('input[name="passphrase"]', 'demo-passphrase-2026');
    await page.click('button[type="submit"]');
    await page.waitForSelector('.cards');

    // Settings: practice + two payers
    await page.goto(url + 'settings');
    await page.fill('form[action="/settings/practice"] input[name="name"]', 'Sunrise Counseling');
    await page.selectOption('form[action="/settings/practice"] select[name="kind"]', 'own');
    await page.click('form[action="/settings/practice"] button');
    await page.waitForSelector('form[action="/settings/payer"]');

    await page.fill('form[action="/settings/payer"] input[name="name"]', 'HealthBridge');
    await page.selectOption('form[action="/settings/payer"] select[name="kind"]', 'insurance_platform');
    await page.click('form[action="/settings/payer"] button');
    await page.waitForSelector('form[action="/settings/payer"]');

    await page.fill('form[action="/settings/payer"] input[name="name"]', 'Private Pay');
    await page.selectOption('form[action="/settings/payer"] select[name="kind"]', 'private');
    await page.click('form[action="/settings/payer"] button');
    await page.waitForSelector('form[action="/settings/payer"]');
    await shot(page, 'settings.png');

    // Auto-unlock section (disabled state — always visible regardless of keychain support)
    await page.locator('.card:has(h2:text("Auto-unlock"))').scrollIntoViewIfNeeded();
    await shot(page, 'settings-auto-unlock.png');

    // Attempt to enable auto-unlock and capture enabled state if the OS keychain
    // is available (succeeds on macOS; may silently fail on CI Linux without D-Bus).
    await page.click('form[action="/settings/keychain"] button');
    await page.waitForSelector('h1');
    if (await page.locator('form[action="/settings/keychain"] input[value="remove"]').count() > 0) {
      await page.locator('.card:has(h2:text("Auto-unlock"))').scrollIntoViewIfNeeded();
      await shot(page, 'settings-auto-unlock-enabled.png');
      // Disable again so the app returns to a known state.
      await page.click('form[action="/settings/keychain"] button');
      await page.waitForSelector('h1');
    }
    await page.goto(url + 'settings');
    await page.waitForSelector('h1');

    // Danger zone — reset page (shows the warning and confirmation form)
    await page.goto(url + 'reset');
    await page.waitForSelector('h1');
    await shot(page, 'reset.png');
    await page.goto(url + 'settings');
    await page.waitForSelector('h1');

    // Rates: fill form for first payer, screenshot before submitting
    await page.goto(url + 'rates');
    await page.selectOption('select[name="payer_id"]', { label: 'HealthBridge' });
    await page.fill('input[name="service"]', '90837');
    await page.fill('input[name="amount"]', '$150');
    await shot(page, 'rates-form.png');
    await page.click('form[action="/rates"] button');
    await page.waitForSelector('table');

    await page.selectOption('select[name="payer_id"]', { label: 'Private Pay' });
    await page.fill('input[name="service"]', '90837');
    await page.fill('input[name="amount"]', '$175');
    await page.click('form[action="/rates"] button');
    await page.waitForSelector('table');
    await shot(page, 'rates.png');

    // Sessions: realistic mix of paid and outstanding
    const sessions = [
      { date: '2026-05-05', client: 'AB', payer: 'HealthBridge', paid: true },
      { date: '2026-05-12', client: 'CD', payer: 'Private Pay',  paid: true },
      { date: '2026-05-19', client: 'EF', payer: 'HealthBridge', paid: true },
      { date: '2026-05-26', client: 'AB', payer: 'HealthBridge', paid: true },
      { date: '2026-06-02', client: 'CD', payer: 'Private Pay',  paid: true },
      { date: '2026-06-09', client: 'EF', payer: 'HealthBridge', paid: false },
      { date: '2026-06-16', client: 'AB', payer: 'HealthBridge', paid: false },
      { date: '2026-06-23', client: 'GH', payer: 'Private Pay',  paid: false },
    ];

    await page.goto(url + 'sessions');

    // Fill the first session form and screenshot before submitting
    const first = sessions[0];
    await page.fill('input[name="date"]', first.date);
    await page.fill('input[name="client_id"]', first.client);
    await page.selectOption('select[name="payer_id"]', { label: first.payer });
    await page.fill('input[name="service"]', '90837');
    await page.selectOption('select[name="status"]', 'completed');
    await page.uncheck('input[name="paid"]');
    await shot(page, 'session-form.png');
    // Set actual paid state then submit
    if (first.paid) await page.check('input[name="paid"]');
    await page.click('form[action="/sessions"] button');
    await page.waitForSelector('table');

    // Add remaining sessions
    for (const s of sessions.slice(1)) {
      await page.fill('input[name="date"]', s.date);
      await page.fill('input[name="client_id"]', s.client);
      await page.selectOption('select[name="payer_id"]', { label: s.payer });
      await page.fill('input[name="service"]', '90837');
      await page.selectOption('select[name="status"]', 'completed');
      if (s.paid) {
        await page.check('input[name="paid"]');
      } else {
        await page.uncheck('input[name="paid"]');
      }
      await page.click('form[action="/sessions"] button');
      await page.waitForSelector('table');
    }
    await shot(page, 'sessions.png');

    // Close-up of the sessions table (highlighting paid vs outstanding rows)
    await page.locator('.card').last().screenshot({ path: join(SCREENSHOTS_DIR, 'sessions-table.png') });

    // Dashboard with data
    await page.goto(url);
    await page.waitForSelector('.cards');
    await shot(page, 'dashboard.png');

    // Income breakdown: clip to the .grid2 section
    await page.locator('.grid2').scrollIntoViewIfNeeded();
    const grid2Box = await page.locator('.grid2').boundingBox();
    if (grid2Box) {
      await page.screenshot({
        path: join(SCREENSHOTS_DIR, 'dashboard-breakdown.png'),
        clip: {
          x: 0,
          y: Math.max(0, grid2Box.y - 16),
          width: 1280,
          height: grid2Box.height + 32,
        },
      });
    }

    // Navigation bar (element screenshot — shows Home, Sessions, Rates, Import,
    // Settings, Export, Lock, and Quit)
    await page.locator('.topbar').screenshot({ path: join(SCREENSHOTS_DIR, 'nav-bar.png') });

    // Import page
    await page.goto(url + 'import');
    await page.waitForSelector('h1');
    await shot(page, 'import.png');

  } finally {
    proc.kill();
  }
});
