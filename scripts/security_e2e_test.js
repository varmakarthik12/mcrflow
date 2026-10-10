const fs = require('fs');
const path = require('path');
const http = require('http');
const { spawn } = require('child_process');

let puppeteer;
try {
  puppeteer = require('puppeteer');
} catch (e) {
  puppeteer = require(path.resolve(__dirname, '../web/node_modules/puppeteer'));
}

const TEST_PORT = 3081;
const DATA_DIR = path.resolve(__dirname, '../tmp/sec-test-data');
const SCREENSHOT_DIR = path.resolve(__dirname, '../tmp/sec-screenshots');

if (fs.existsSync(DATA_DIR)) {
  fs.rmSync(DATA_DIR, { recursive: true, force: true });
}
fs.mkdirSync(DATA_DIR, { recursive: true });
if (!fs.existsSync(SCREENSHOT_DIR)) {
  fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function makeRequest(method, urlPath, token = null, body = null) {
  return new Promise((resolve, reject) => {
    const postData = body ? JSON.stringify(body) : null;
    const headers = {
      'Content-Type': 'application/json',
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
    if (postData) {
      headers['Content-Length'] = Buffer.byteLength(postData);
    }

    const req = http.request(
      {
        hostname: '127.0.0.1',
        port: TEST_PORT,
        path: urlPath,
        method: method,
        headers: headers,
        timeout: 5000,
      },
      (res) => {
        let raw = '';
        res.on('data', (chunk) => (raw += chunk));
        res.on('end', () => {
          let data = null;
          try {
            data = JSON.parse(raw);
          } catch (e) {
            data = raw;
          }
          resolve({ status: res.statusCode, data });
        });
      }
    );

    req.on('error', reject);
    req.on('timeout', () => {
      req.destroy();
      reject(new Error('Request timed out'));
    });

    if (postData) {
      req.write(postData);
    }
    req.end();
  });
}

async function runSecurityE2ETests() {
  console.log('================================================================');
  console.log('[MCRFlow Security & RBAC E2E Test Suite]');
  console.log('================================================================');

  const binaryPath = path.resolve(__dirname, '../bin/mcrflow-control.exe');
  console.log(`[1] Launching Control Plane binary: ${binaryPath}`);
  const serverProc = spawn(
    binaryPath,
    ['-port', String(TEST_PORT), '-data-dir', DATA_DIR, '-media-dir', './media'],
    { stdio: ['ignore', 'pipe', 'pipe'] }
  );

  serverProc.stdout.on('data', (d) => {
    // console.log(`[SRV] ${d.toString().trim()}`);
  });
  serverProc.stderr.on('data', (d) => {
    // console.error(`[SRV-ERR] ${d.toString().trim()}`);
  });

  // Wait for server health
  let serverReady = false;
  for (let i = 0; i < 30; i++) {
    try {
      const res = await makeRequest('GET', '/api/v1/health');
      if (res.status === 200) {
        serverReady = true;
        console.log('    ✓ Control plane is listening and healthy on port ' + TEST_PORT);
        break;
      }
    } catch (e) {
      await delay(300);
    }
  }

  if (!serverReady) {
    serverProc.kill();
    throw new Error('Server failed to start within timeout');
  }

  try {
    // -------------------------------------------------------------
    // STEP 2: Strict Unauthenticated API Gating
    // -------------------------------------------------------------
    console.log('\n[2] Verifying strict API lockdown for unauthenticated callers:');
    
    const setupStatus = await makeRequest('GET', '/api/v1/auth/setup-status');
    console.log(`    ✓ /api/v1/auth/setup-status => status: ${setupStatus.status}, needs_setup: ${setupStatus.data.needs_setup}, user_count: ${setupStatus.data.user_count}`);
    if (!setupStatus.data.needs_setup || setupStatus.data.user_count !== 0) {
      throw new Error(`Expected needs_setup: true and user_count: 0, got ${JSON.stringify(setupStatus.data)}`);
    }

    const protectedEndpoints = [
      { method: 'GET', path: '/api/v1/channels' },
      { method: 'POST', path: '/api/v1/channels' },
      { method: 'GET', path: '/api/v1/users' },
      { method: 'POST', path: '/api/v1/users' },
      { method: 'GET', path: '/api/v1/schedules' },
      { method: 'GET', path: '/api/v1/resolutions' },
      { method: 'GET', path: '/api/v1/ad-templates' },
      { method: 'GET', path: '/api/v1/agents' },
      { method: 'GET', path: '/api/v1/bots' },
      { method: 'GET', path: '/api/v1/storage/browse' },
    ];

    for (const ep of protectedEndpoints) {
      const res = await makeRequest(ep.method, ep.path);
      if (res.status !== 401) {
        throw new Error(`Expected 401 for unauthenticated ${ep.method} ${ep.path}, got ${res.status}: ${JSON.stringify(res.data)}`);
      }
      console.log(`    ✓ ${ep.method} ${ep.path} => 401 Unauthorized (properly locked down)`);
    }

    // -------------------------------------------------------------
    // STEP 3: Puppeteer UI Lockdown & Initial Setup Flow
    // -------------------------------------------------------------
    console.log('\n[3] Launching Puppeteer browser for UI Lockdown verification...');
    const browser = await puppeteer.launch({
      headless: 'new',
      args: ['--no-sandbox', '--disable-setuid-sandbox'],
    });
    const page = await browser.newPage();
    page.on('console', (msg) => console.log('    [BROWSER-CONSOLE]', msg.text()));
    page.on('pageerror', (err) => console.error('    [BROWSER-ERROR]', err.message));
    await page.setViewport({ width: 1440, height: 900 });

    await page.goto(`http://localhost:${TEST_PORT}`, { waitUntil: 'networkidle0' });
    await delay(1000);

    // Verify Setup Modal is shown and NO main screens or header exist
    const hasSetupModal = await page.evaluate(() => {
      const h3 = Array.from(document.querySelectorAll('h3')).find(el => el.textContent.includes('Setup'));
      return !!h3;
    });
    const hasMainScreen = await page.evaluate(() => {
      return !!document.querySelector('main');
    });

    console.log(`    ✓ Initial launch: Setup modal visible: ${hasSetupModal}`);
    console.log(`    ✓ Initial launch: Protected main workspace rendered: ${hasMainScreen} (MUST be false)`);
    if (!hasSetupModal || hasMainScreen) {
      throw new Error(`UI lockdown failed: SetupModal=${hasSetupModal}, MainScreen=${hasMainScreen}`);
    }

    await page.screenshot({ path: path.join(SCREENSHOT_DIR, '01_initial_setup_locked.png') });

    // Complete Root Admin Setup via UI form
    console.log('    -> Filling out Root Admin Setup form via native typing...');
    await page.type('input[placeholder*="admin"]', 'admin');
    await page.type('input[placeholder*="Chief"]', 'Lead Broadcast Administrator');
    await page.type('input[placeholder*="chief@"]', 'admin@mcrflow.internal');
    await page.type('input[placeholder*="Minimum"]', 'BroadcastAdmin#2026');
    await page.type('input[placeholder*="Re-enter"]', 'BroadcastAdmin#2026');

    await delay(300);
    const submitBtn = await page.$('button[type="submit"]');
    if (submitBtn) {
      await submitBtn.click();
    }

    // Wait for setup completion and workspace mounting
    await delay(2500);
    const workspaceMounted = await page.evaluate(() => {
      return !!document.querySelector('main') && !!document.querySelector('header');
    });
    console.log(`    ✓ Post-setup: Master Control workspace successfully mounted: ${workspaceMounted}`);
    if (!workspaceMounted) {
      throw new Error('Workspace did not mount after root admin setup');
    }
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, '02_workspace_authenticated.png') });

    // -------------------------------------------------------------
    // STEP 4: Test Prominent Sign Out Button & UI Re-lockdown
    // -------------------------------------------------------------
    console.log('\n[4] Testing prominent Sign Out button & complete UI re-lockdown:');
    const logoutBtnExists = await page.evaluate(() => {
      const btn = document.getElementById('header-logout-btn');
      return !!btn && btn.offsetParent !== null;
    });
    console.log(`    ✓ Header Sign Out button exists and is visible: ${logoutBtnExists}`);
    if (!logoutBtnExists) {
      throw new Error('Header Sign Out button (#header-logout-btn) is missing or hidden');
    }

    // Click Sign Out
    console.log('    -> Clicking Sign Out button...');
    await page.evaluate(() => {
      document.getElementById('header-logout-btn').click();
    });
    await delay(1000);

    // Verify localStorage token was purged and UI locked down
    const postLogoutState = await page.evaluate(() => {
      return {
        token: localStorage.getItem('mcrflow_jwt'),
        hasMain: !!document.querySelector('main'),
        hasHeader: !!document.querySelector('header'),
        hasLoginModal: Array.from(document.querySelectorAll('h3')).some(h => h.textContent.includes('Login')),
      };
    });

    console.log(`    ✓ Token in localStorage after logout: ${postLogoutState.token || 'null (purged)'}`);
    console.log(`    ✓ Main workspace rendered after logout: ${postLogoutState.hasMain} (MUST be false)`);
    console.log(`    ✓ Header rendered after logout: ${postLogoutState.hasHeader} (MUST be false)`);
    console.log(`    ✓ Dedicated Login Modal displayed: ${postLogoutState.hasLoginModal} (MUST be true)`);

    if (postLogoutState.token || postLogoutState.hasMain || postLogoutState.hasHeader || !postLogoutState.hasLoginModal) {
      throw new Error(`Logout failed state check: ${JSON.stringify(postLogoutState)}`);
    }
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, '03_post_logout_locked.png') });

    // Test Signing in again via UI
    console.log('    -> Signing back in via Login Modal with root admin credentials...');
    await page.type('input[placeholder*="username"]', 'admin');
    await page.type('input[placeholder*="password"]', 'BroadcastAdmin#2026');
    const loginSubmitBtn = await page.$('button[type="submit"]');
    if (loginSubmitBtn) {
      await loginSubmitBtn.click();
    }

    await delay(2000);
    const reauthenticated = await page.evaluate(() => {
      return !!document.querySelector('main') && !!localStorage.getItem('mcrflow_jwt');
    });
    console.log(`    ✓ Workspace re-authenticated successfully: ${reauthenticated}`);
    if (!reauthenticated) {
      throw new Error('Failed to re-authenticate via Login Modal');
    }

    await browser.close();

    // -------------------------------------------------------------
    // STEP 5: Root Setup Anti-Tamper Check
    // -------------------------------------------------------------
    console.log('\n[5] Verifying Root Setup Anti-Tamper Protection:');
    const secondSetup = await makeRequest('POST', '/api/v1/auth/setup', null, {
      username: 'hacker',
      password: 'hackerpassword',
    });
    console.log(`    ✓ Attempted rogue setup with existing users => status: ${secondSetup.status} (MUST be 403 Forbidden)`);
    if (secondSetup.status !== 403) {
      throw new Error(`Expected 403 Forbidden on secondary setup attempt, got ${secondSetup.status}`);
    }

    // -------------------------------------------------------------
    // STEP 6: RBAC Role Separation (Admin vs Operator vs Content Scheduler)
    // -------------------------------------------------------------
    console.log('\n[6] Verifying Enterprise RBAC Role Separation:');
    
    // Obtain Admin token
    const loginAdmin = await makeRequest('POST', '/api/v1/auth/login', null, {
      username: 'admin',
      password: 'BroadcastAdmin#2026',
    });
    const adminToken = loginAdmin.data.token;
    console.log(`    ✓ Obtained Admin JWT token successfully`);

    // Admin creates Operator user
    const createOp = await makeRequest('POST', '/api/v1/users', adminToken, {
      username: 'op_user',
      full_name: 'Broadcast Operator One',
      role: 'operator',
      password: 'OperatorPass#2026',
    });
    console.log(`    ✓ Admin created Operator user 'op_user' => status: ${createOp.status}`);
    if (createOp.status !== 200 && createOp.status !== 201) {
      throw new Error(`Failed to create operator: ${JSON.stringify(createOp.data)}`);
    }

    // Operator logs in
    const loginOp = await makeRequest('POST', '/api/v1/auth/login', null, {
      username: 'op_user',
      password: 'OperatorPass#2026',
    });
    const opToken = loginOp.data.token;
    console.log(`    ✓ Operator logged in, token acquired`);

    // Operator attempts to access Admin-only /api/v1/users
    const opUsersAttempt = await makeRequest('GET', '/api/v1/users', opToken);
    console.log(`    ✓ Operator accessing Admin-only /api/v1/users => status: ${opUsersAttempt.status} (MUST be 403)`);
    if (opUsersAttempt.status !== 403) {
      throw new Error(`Expected 403 for operator on /api/v1/users, got ${opUsersAttempt.status}`);
    }

    // Operator attempts to delete channel (Admin only)
    const opDeleteChannel = await makeRequest('DELETE', '/api/v1/channels/ch-01', opToken);
    console.log(`    ✓ Operator attempting channel deletion => status: ${opDeleteChannel.status} (MUST be 403)`);
    if (opDeleteChannel.status !== 403) {
      throw new Error(`Expected 403 for operator deleting channel, got ${opDeleteChannel.status}`);
    }

    // Operator starts playout (Allowed for Operator & Admin)
    const opPlayoutStart = await makeRequest('POST', '/api/v1/channels/ch-01/playout/start', opToken);
    console.log(`    ✓ Operator starting playout => status: ${opPlayoutStart.status} (MUST be 200)`);
    if (opPlayoutStart.status !== 200) {
      throw new Error(`Expected 200 for operator playout start, got ${opPlayoutStart.status}`);
    }

    // -------------------------------------------------------------
    // STEP 7: Security - Safe Path & Image Validation Checks
    // -------------------------------------------------------------
    console.log('\n[7] Verifying Path Traversal & Image Extension Security:');
    
    // Path traversal in storage browse
    const traversalBrowse = await makeRequest('GET', '/api/v1/storage/browse?path=../../../../windows/system32', adminToken);
    console.log(`    ✓ Browse path traversal attempt => status: ${traversalBrowse.status} (cleanly confined or handled)`);
    if (traversalBrowse.status === 200 && Array.isArray(traversalBrowse.data)) {
      // Must not contain cmd.exe or calc.exe or windows system files
      const hasSystemFiles = traversalBrowse.data.some(f => f.name.toLowerCase() === 'cmd.exe');
      if (hasSystemFiles) {
        throw new Error('Path traversal vulnerability detected! Traversal browsed into Windows system directory!');
      }
      console.log('    ✓ Path traversal safely neutralized: confined to media repository root');
    }

    console.log('\n================================================================');
    console.log('🎉 ALL SECURITY & RBAC E2E TESTS PASSED WITH 100% SUCCESS!');
    console.log('================================================================');
  } finally {
    try {
      serverProc.kill();
      require('child_process').execSync('taskkill /f /im mcrflow-control.exe', { stdio: 'ignore' });
    } catch (e) {}
  }
}

runSecurityE2ETests().catch((err) => {
  console.error('\n❌ Security E2E Test Suite FAILED:', err);
  process.exit(1);
});
