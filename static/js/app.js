// Minimal HTMX & Alpine Helper (Zero Client State Boilerplate)
document.addEventListener('DOMContentLoaded', () => {
    // HTMX automatically reloads grid when notes change
    document.body.addEventListener('noteChanged', () => {
        const noteModal = document.getElementById('note-modal');
        if (noteModal) noteModal.style.display = 'none';
        showToast('Note saved!', 'success');
    });
});

function showToast(message, type = 'info') {
    const container = document.getElementById('toast-container');
    if (!container) return;
    const toast = document.createElement('div');
    toast.className = `alert ${type === 'success' ? 'alert-success' : (type === 'error' ? 'alert-error' : 'alert-info')} shadow-lg text-xs font-medium py-2.5 px-4 rounded-xl flex items-center gap-2`;
    let icon = type === 'success' ? 'ri-checkbox-circle-line' : (type === 'error' ? 'ri-error-warning-line' : 'ri-information-line');
    toast.innerHTML = `<i class="${icon} text-base"></i> <span>${message}</span>`;
    container.appendChild(toast);
    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transition = 'all 0.3s ease';
        setTimeout(() => toast.remove(), 300);
    }, 2800);
}
