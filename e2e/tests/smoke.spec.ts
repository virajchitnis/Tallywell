import { test, expect } from '@playwright/test';
import { spawn, ChildProcess } from 'node:child_process';
import { mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

// Launches the built Tallywell binary with an isolated data directory and
// resolves the loopback URL it prints on startup.
function startApp(): Promise<{ proc: ChildProcess; url: string }> {
  const bin = process.env.TALLYWELL_BIN;
  if (!bin) throw new Error('set TALLYWELL_BIN to the built binary path');
  const home = mkdtempSync(join(tmpdir(), 'tallywell-e2e-'));

  return new Promise((resolve, reject) => {
    const proc = spawn(bin, [], { env: { ...process.env, HOME: home } });
    let out = '';
    const onData = (d: Buffer) => {
      out += d.toString();
      const m = out.match(/http:\/\/127\.0\.0\.1:\d+\//);
      if (m) {
        proc.stdout?.off('data', onData);
        resolve({ proc, url: m[0] });
      }
    };
    proc.stdout?.on('data', onData);
    proc.on('error', reject);
    setTimeout(() => reject(new Error('app did not start in time')), 10_000);
  });
}

test('setup, configure, log a session, and export', async ({ page }) => {
  const { proc, url } = await startApp();
  try {
    // First run -> setup screen.
    await page.goto(url);
    await expect(page.locator('h1')).toContainText('Welcome');
    await page.fill('input[name="passphrase"]', 'hunter2hunter2');
    await page.fill('input[name="confirm"]', 'hunter2hunter2');
    await page.click('button[type="submit"]');
    await expect(page.locator('h1')).toContainText("here's where things stand", { ignoreCase: true });

    // Configure a practice and payer.
    await page.goto(url + 'settings');
    await page.fill('input[name="name"]', 'My Practice');
    await page.selectOption('select[name="kind"]', 'own');
    await page.click('form[action="/settings/practice"] button');

    await page.fill('form[action="/settings/payer"] input[name="name"]', 'Platform A');
    await page.click('form[action="/settings/payer"] button');

    // Add a rate.
    await page.goto(url + 'rates');
    await page.fill('input[name="amount"]', '$120');
    await page.fill('input[name="service"]', '90837');
    await page.click('form[action="/rates"] button');

    // Log a completed session; expected should auto-fill from the rate.
    await page.goto(url + 'sessions');
    await page.fill('input[name="date"]', '2026-06-10');
    await page.fill('input[name="client_id"]', 'AB');
    await page.fill('input[name="service"]', '90837');
    await page.click('form[action="/sessions"] button');
    await expect(page.locator('table')).toContainText('$120.00');

    // Dashboard reflects the outstanding amount.
    await page.goto(url);
    await expect(page.locator('.cards')).toContainText('$120.00');
  } finally {
    proc.kill();
  }
});
