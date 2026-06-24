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

    // Unlock screen (lock first)
    await page.click('form[action="/lock"] button');
    await page.waitForSelector('input[name="passphrase"]');
    await shot(page, 'unlock.png');

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

    // Rates: one per payer
    await page.goto(url + 'rates');
    await page.selectOption('select[name="payer_id"]', { label: 'HealthBridge' });
    await page.fill('input[name="service"]', '90837');
    await page.fill('input[name="amount"]', '$150');
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
    for (const s of sessions) {
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

    // Dashboard with data
    await page.goto(url);
    await page.waitForSelector('.cards');
    await shot(page, 'dashboard.png');

    // Import page
    await page.goto(url + 'import');
    await page.waitForSelector('h1');
    await shot(page, 'import.png');

  } finally {
    proc.kill();
  }
});
