let clearTimer = null;

export function showMessage(text, type = "info") {
    const msg = document.getElementById('msg');
    if (!msg) return;

    if (clearTimer) {
        clearTimeout(clearTimer);
        clearTimer = null;
    }

    msg.textContent = text;
    msg.classList.remove('msg-error', 'msg-success', 'msg-info');
    msg.classList.add('msg-' + type);

    if (type !== 'error') {
        clearTimer = setTimeout(() => {
            msg.textContent = '';
            msg.classList.remove('msg-error', 'msg-success', 'msg-info');
        }, 5000);
    }
}

export function setButtonLoading(button, isLoading, loadingText = 'Loading...') {
    if (!button) return;
    if (isLoading) {
        if (button.dataset.originalText === undefined) {
            button.dataset.originalText = button.textContent;
        }
        button.textContent = loadingText;
        button.disabled = true;
    } else {
        if (button.dataset.originalText !== undefined) {
            button.textContent = button.dataset.originalText;
            delete button.dataset.originalText;
        }
        button.disabled = false;
    }
}
