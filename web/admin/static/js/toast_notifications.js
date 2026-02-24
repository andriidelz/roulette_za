// ============================================
// TOAST NOTIFICATION SYSTEM
// ============================================

window.showToast = function (message, type = "info", duration = 3000) {
  let toastContainer = document.getElementById("toast-container");
  if (!toastContainer) {
    toastContainer = document.createElement("div");
    toastContainer.id = "toast-container";
    toastContainer.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            z-index: 9999;
            min-width: 300px;
        `;
    document.body.appendChild(toastContainer);
  }

  const toast = document.createElement("div");
  toast.className = `alert alert-${type} alert-dismissible fade show`;
  toast.style.cssText = `
        margin-bottom: 10px;
        box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        animation: slideIn 0.3s ease-out;
    `;
  toast.setAttribute("role", "alert");

  const icons = {
    success: "bi-check-circle-fill",
    error: "bi-x-circle-fill",
    warning: "bi-exclamation-triangle-fill",
    info: "bi-info-circle-fill",
  };

  toast.innerHTML = `
        <i class="bi ${icons[type]} me-2"></i>
        <span>${message}</span>
        <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
    `;

  toastContainer.appendChild(toast);

  setTimeout(() => {
    toast.classList.remove("show");
    setTimeout(() => toast.remove(), 300);
  }, duration);
};

if (!document.getElementById("toast-animation-style")) {
  const style = document.createElement("style");
  style.id = "toast-animation-style";
  style.textContent = `
        @keyframes slideIn {
            from {
                transform: translateX(100%);
                opacity: 0;
            }
            to {
                transform: translateX(0);
                opacity: 1;
            }
        }
    `;
  document.head.appendChild(style);
}

// ============================================
// AJAX ERROR HANDLER
// ============================================
$(document).ajaxError(function (event, jqxhr, settings, thrownError) {
  if (jqxhr.status === 403) {
    const response = jqxhr.responseJSON;
    if (response && response.error) {
      showToast(response.error, "error", 5000);
    } else {
      showToast(
        "Ви не маєте необхідних прав для виконання цієї операції",
        "error",
        5000,
      );
    }
  }
});
