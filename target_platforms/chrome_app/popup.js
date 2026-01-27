// Check server status
async function checkServerStatus() {
  const statusEl = document.getElementById('server-status');
  try {
    const response = await fetch('http://localhost:8090/api/health', { 
      method: 'GET',
      mode: 'cors'
    });
    if (response.ok) {
      statusEl.textContent = '✓ Running';
      statusEl.style.color = '#4ade80';
    } else {
      statusEl.textContent = '⚠ Error';
      statusEl.style.color = '#fbbf24';
    }
  } catch (error) {
    statusEl.textContent = '✗ Offline';
    statusEl.style.color = '#f87171';
  }
}

// Open admin UI
document.getElementById('open-admin').addEventListener('click', () => {
  chrome.tabs.create({ url: 'http://localhost:8090/_/' });
});

// Open data browser
document.getElementById('open-data').addEventListener('click', () => {
  chrome.tabs.create({ url: 'http://localhost:8090/' });
});

// Open docs
document.getElementById('docs-link').addEventListener('click', (e) => {
  e.preventDefault();
  chrome.tabs.create({ url: 'https://github.com/darianmavgo/flight3' });
});

// Check status on load
checkServerStatus();
