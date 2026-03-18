// i18n helper
const msg = (key, ...substitutions) => chrome.i18n.getMessage(key, substitutions);

// Apply i18n to all elements with data-i18n attribute
function applyI18n() {
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.getAttribute('data-i18n');
        const prefix = el.getAttribute('data-i18n-prefix') || '';
        const translated = msg(key);
        if (translated) {
            el.textContent = prefix + translated;
        }
    });
}

// State
let currentContentType = 'album';
let selectedUrl = '';

// UI Elements
const tokenStatusEl = document.getElementById('tokenStatus');
const tokenStatusText = document.getElementById('tokenStatusText');
const refreshTokenBtn = document.getElementById('refreshTokenBtn');
const connectionStatus = document.getElementById('connectionStatus');

const deezerUrlEl = document.getElementById('deezerUrl');
const extractBtn = document.getElementById('extractBtn');
const contentIdEl = document.getElementById('contentId');
const downloadBtn = document.getElementById('downloadBtn');
const downloadStatus = document.getElementById('downloadStatus');
const progressContainer = document.getElementById('progressContainer');
const progressBar = document.getElementById('progressBar');

const serverUrlEl = document.getElementById('serverUrl');
const saveSettingsBtn = document.getElementById('saveSettingsBtn');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    applyI18n();
    loadSettings();
    updateTokenStatus();
    setupEventListeners();
    checkServerConnection();
    extractCurrentPageInfo();
});

function setupEventListeners() {
    refreshTokenBtn.addEventListener('click', refreshToken);
    extractBtn.addEventListener('click', extractUrlId);
    downloadBtn.addEventListener('click', startDownload);
    saveSettingsBtn.addEventListener('click', saveSettings);
    
    deezerUrlEl.addEventListener('paste', () => {
        setTimeout(extractUrlId, 10);
    });
}

function extractUrlId() {
    const url = deezerUrlEl.value || selectedUrl;
    
    // Pattern: /album/123456 ou /playlist/123456
    const match = url.match(/\/(album|playlist)\/(\d+)/);
    
    if (match) {
        const type = match[1];
        const id = match[2];
        
        currentContentType = type; // Auto-detect type from URL
        contentIdEl.value = id;
        showStatus(msg('idExtracted', type, id), 'success');
    } else {
        showStatus(msg('errorInvalidUrl'), 'error');
    }
}

function updateTokenStatus() {
    chrome.storage.local.get(['licenseToken'], (result) => {
        const token = result.licenseToken;
        
        if (token) {
            tokenStatusEl.classList.add('active');
            const displayToken = token.substring(0, 20) + '...';
            tokenStatusText.textContent = '✅ ' + msg('tokenFound', displayToken);
        } else {
            tokenStatusEl.classList.remove('active');
            tokenStatusText.textContent = '⚠️ ' + msg('tokenNoneFallback');
        }
    });
}

function refreshToken() {
    refreshTokenBtn.textContent = '⏳ ' + msg('btnRefreshing');
    refreshTokenBtn.disabled = true;
    
    chrome.runtime.sendMessage({ action: 'getTokenFromTab' }, (response) => {
        refreshTokenBtn.disabled = false;
        refreshTokenBtn.textContent = '🔄 ' + msg('btnRefresh');
        
        if (response && response.token) {
            chrome.storage.local.set({ licenseToken: response.token }, () => {
                updateTokenStatus();
                showStatus(msg('tokenRefreshed'), 'success');
            });
        } else {
            showStatus(msg('tokenCaptureError'), 'error');
        }
    });
}

function checkServerConnection() {
    chrome.storage.local.get(['serverUrl'], (result) => {
        const serverUrl = result.serverUrl || 'http://localhost:8080';
        
        fetch(`${serverUrl}/health`, { method: 'GET' })
            .then(res => res.ok ? Promise.resolve() : Promise.reject())
            .then(() => {
                connectionStatus.textContent = msg('connected');
                connectionStatus.classList.add('success');
            })
            .catch(() => {
                connectionStatus.textContent = msg('disconnected');
                connectionStatus.classList.remove('success');
            });
    });
}

function extractCurrentPageInfo() {
    chrome.tabs.query({ active: true, currentWindow: true }, (tabs) => {
        const tab = tabs[0];
        if (tab.url.includes('deezer.com')) {
            deezerUrlEl.value = tab.url;
            extractUrlId();
        }
    });
}

async function startDownload() {
    const contentId = contentIdEl.value.trim();
    const serverUrl = serverUrlEl.value.trim() || 'http://localhost:8080';
    
    if (!contentId) {
        showStatus(msg('errorMissingId'), 'error');
        return;
    }
    
    // Get token from storage (optional - server may have a token in config)
    chrome.storage.local.get(['licenseToken'], async (result) => {
        const token = result.licenseToken || ''; // Token optionnel
        
        downloadBtn.disabled = true;
        downloadBtn.textContent = '⏳ ' + msg('btnDownloading');
        progressContainer.style.display = 'block';
        progressBar.style.width = '0%';
        progressBar.textContent = '0%';
        
        try {
            const endpoint = currentContentType === 'album' ? 'download-album' : 'download-playlist';
            
            const body = {
                id: contentId
            };
            
            // Inclure le token seulement s'il existe localement
            // Le serveur utilisera celui de config.toml comme fallback
            if (token) {
                body.license_token = token;
            }
            
            const response = await fetch(`${serverUrl}/${endpoint}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(body)
            });
            
            if (!response.ok) {
                throw new Error(msg('errorServer', response.status.toString()));
            }
            
            const data = await response.json();
            
            if (data.success) {
                progressBar.style.width = '100%';
                progressBar.textContent = '100%';
                showStatus('✅ ' + msg('downloadComplete', data.message || ''), 'success');
            } else {
                showStatus(msg('errorDownload', data.error || msg('errorUnknown')), 'error');
            }
        } catch (error) {
            showStatus(msg('errorConnection', error.message), 'error');
        } finally {
            downloadBtn.disabled = false;
            downloadBtn.textContent = '⬇️ ' + msg('btnDownload');
            setTimeout(() => {
                progressContainer.style.display = 'none';
            }, 2000);
        }
    });
}

function showStatus(message, type) {
    downloadStatus.textContent = message;
    downloadStatus.className = `show ${type}`;
    
    if (type === 'success') {
        setTimeout(() => {
            downloadStatus.classList.remove('show');
        }, 3000);
    }
}

function loadSettings() {
    chrome.storage.local.get(['serverUrl'], (result) => {
        if (result.serverUrl) {
            serverUrlEl.value = result.serverUrl;
        }
    });
}

function saveSettings() {
    const serverUrl = serverUrlEl.value.trim();
    
    if (!serverUrl) {
        showStatus(msg('errorMissingServerUrl'), 'error');
        return;
    }
    
    chrome.storage.local.set({ serverUrl }, () => {
        showStatus(msg('settingsSaved'), 'success');
        checkServerConnection();
    });
}

const settingsToggle = document.getElementById('settingsToggle');
const mainView = document.getElementById('mainView');
const settingsView = document.getElementById('settingsView');

let isSettingsOpen = false;

settingsToggle.addEventListener('click', () => {
    isSettingsOpen = !isSettingsOpen;

    if (isSettingsOpen) {
        mainView.style.display = 'none';
        settingsView.style.display = 'block';
        settingsToggle.textContent = '⬅️';
    } else {
        mainView.style.display = 'block';
        settingsView.style.display = 'none';
        settingsToggle.textContent = '⚙️';
    }
});
