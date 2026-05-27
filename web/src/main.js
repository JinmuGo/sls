import './style.css';
import demoGif from '../../demo.gif';

// 1. GitHub API Integration for Stars
async function fetchGitHubStars() {
  const starCountEl = document.getElementById('github-star-count');
  if (!starCountEl) return;
  
  try {
    const response = await fetch('https://api.github.com/repos/jinmugo/sls');
    if (response.ok) {
      const data = await response.json();
      const stars = data.stargazers_count;
      starCountEl.textContent = `★ ${stars.toLocaleString()}`;
    } else {
      starCountEl.textContent = '★ 142';
    }
  } catch (error) {
    starCountEl.textContent = '★ 142';
  }
}

// 2. Clipboard Copy Logic
const INSTALL_COMMANDS = {
  brew: 'brew install jinmugo/tap/sls',
  curl: 'curl -fsSL https://package.jinmu.me/install.sh | sudo sh -s sls',
  go: 'go install github.com/jinmugo/sls@latest'
};

function initInstaller() {
  const copyBtn = document.getElementById('btn-copy-install');
  const codeEl = document.getElementById('cmd-brew');
  const tabs = document.querySelectorAll('.switcher-tab');
  
  if (!copyBtn || !codeEl) return;
  
  // Wire up tabs
  tabs.forEach(tab => {
    tab.addEventListener('click', () => {
      tabs.forEach(t => t.classList.remove('active'));
      tab.classList.add('active');
      
      const cmdType = tab.getAttribute('data-cmd');
      codeEl.textContent = INSTALL_COMMANDS[cmdType];
    });
  });
  
  copyBtn.addEventListener('click', async () => {
    const commandText = codeEl.textContent.trim();
    
    try {
      await navigator.clipboard.writeText(commandText);
      
      // Visual feedback
      copyBtn.classList.add('success');
      const textSpan = copyBtn.querySelector('.copy-text');
      const originalText = textSpan.textContent;
      textSpan.textContent = 'Copied!';
      
      // Replace copy icon with checkmark
      const originalSvg = copyBtn.querySelector('svg').innerHTML;
      copyBtn.querySelector('svg').innerHTML = `
        <polyline points="20 6 9 17 4 12"></polyline>
      `;
      
      setTimeout(() => {
        copyBtn.classList.remove('success');
        textSpan.textContent = originalText;
        copyBtn.querySelector('svg').innerHTML = originalSvg;
      }, 2000);
    } catch (err) {
      console.error('Failed to copy to clipboard', err);
    }
  });
}

// 3. App Initialization
function initApp() {
  fetchGitHubStars();
  initInstaller();
  
  const demoImageEl = document.getElementById('demo-image');
  if (demoImageEl) {
    demoImageEl.src = demoGif;
  }
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initApp);
} else {
  initApp();
}
