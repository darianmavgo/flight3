// Background service worker for Flight3 Chrome extension

chrome.runtime.onInstalled.addListener(() => {
    console.log('Flight3 extension installed');
});

// Handle messages from popup or content scripts
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
    if (request.action === 'checkServer') {
        fetch('http://localhost:8090/api/health')
            .then(response => sendResponse({ status: response.ok }))
            .catch(() => sendResponse({ status: false }));
        return true; // Will respond asynchronously
    }
});
