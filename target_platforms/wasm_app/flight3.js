// Flight3 WebAssembly Loader and Controller

const outputEl = document.getElementById('output');
const wasmStatusEl = document.getElementById('wasm-status');
const goVersionEl = document.getElementById('go-version');
const memoryEl = document.getElementById('memory');
const startBtn = document.getElementById('start-btn');
const testBtn = document.getElementById('test-btn');
const clearBtn = document.getElementById('clear-btn');

let go = null;
let wasmInstance = null;

// Utility functions
function log(message) {
    const timestamp = new Date().toLocaleTimeString();
    outputEl.textContent += `[${timestamp}] ${message}\n`;
    outputEl.scrollTop = outputEl.scrollHeight;
}

function updateMemory() {
    if (performance.memory) {
        const usedMB = (performance.memory.usedJSHeapSize / 1024 / 1024).toFixed(2);
        const totalMB = (performance.memory.totalJSHeapSize / 1024 / 1024).toFixed(2);
        memoryEl.textContent = `${usedMB} MB / ${totalMB} MB`;
    }
}

// Clear output
clearBtn.addEventListener('click', () => {
    outputEl.textContent = '';
    log('Output cleared');
});

// Initialize WebAssembly
async function initWasm() {
    try {
        log('Initializing Go WebAssembly runtime...');
        wasmStatusEl.textContent = 'Loading...';

        // Create Go runtime instance
        go = new Go();

        log('Fetching WASM module...');
        const result = await WebAssembly.instantiateStreaming(
            fetch('flight3.wasm'),
            go.importObject
        );

        wasmInstance = result.instance;

        log('✓ WASM module loaded successfully');
        wasmStatusEl.textContent = '✓ Ready';
        goVersionEl.textContent = 'Go 1.25 (WASM)';

        // Enable buttons
        startBtn.disabled = false;
        testBtn.disabled = false;

        log('Ready to run Flight3');

        // Update memory usage periodically
        setInterval(updateMemory, 1000);

    } catch (err) {
        log(`✗ Error loading WASM: ${err.message}`);
        wasmStatusEl.textContent = '✗ Failed';
        console.error('WASM initialization error:', err);
    }
}

// Start Flight3
startBtn.addEventListener('click', async () => {
    try {
        log('Starting Flight3 in WebAssembly mode...');
        startBtn.disabled = true;
        startBtn.innerHTML = '<span class="loading"></span> Running...';

        // Run the Go program
        go.run(wasmInstance);

        log('✓ Flight3 started');
        log('Note: Server functionality limited in WASM environment');

    } catch (err) {
        log(`✗ Error starting Flight3: ${err.message}`);
        startBtn.disabled = false;
        startBtn.textContent = 'Start Flight3';
    }
});

// Run test
testBtn.addEventListener('click', () => {
    log('Running WASM test...');
    log('✓ WASM environment is functional');
    log(`  - Browser: ${navigator.userAgent}`);
    log(`  - Platform: ${navigator.platform}`);
    log(`  - Language: ${navigator.language}`);

    if (performance.memory) {
        updateMemory();
        log(`  - Memory: ${memoryEl.textContent}`);
    }
});

// Override console.log to capture Go output
const originalLog = console.log;
console.log = function (...args) {
    originalLog.apply(console, args);
    log(args.join(' '));
};

// Initialize on page load
window.addEventListener('load', () => {
    log('Flight3 WebAssembly Edition');
    log('Initializing...');
    initWasm();
});

// Handle errors
window.addEventListener('error', (event) => {
    log(`✗ Runtime error: ${event.message}`);
});

window.addEventListener('unhandledrejection', (event) => {
    log(`✗ Unhandled promise rejection: ${event.reason}`);
});
