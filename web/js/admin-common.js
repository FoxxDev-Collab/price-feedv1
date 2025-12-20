/**
 * Admin Common JavaScript
 * Shared functionality for all admin pages
 */

// Admin namespace
const admin = {
  // Current user info
  user: null,

  // Initialize admin panel
  async init() {
    await auth.init();

    if (!auth.requireAdmin()) {
      return false;
    }

    this.user = auth.user;
    this.updateUserInfo();
    this.highlightCurrentNav();
    this.setupMobileNav();

    return true;
  },

  // Update user info in sidebar
  updateUserInfo() {
    const user = this.user;
    if (!user) return;

    const nameEl = document.getElementById('admin-name');
    const avatarEl = document.getElementById('admin-avatar');
    const roleEl = document.querySelector('.admin-user-role');

    if (nameEl) {
      nameEl.textContent = user.username || user.email.split('@')[0];
    }
    if (avatarEl) {
      avatarEl.textContent = (user.username || user.email)[0].toUpperCase();
    }
    if (roleEl) {
      roleEl.textContent = user.role.charAt(0).toUpperCase() + user.role.slice(1);
    }
  },

  // Highlight current navigation item
  highlightCurrentNav() {
    const path = window.location.pathname;
    const navItems = document.querySelectorAll('.admin-nav-item');

    navItems.forEach(item => {
      item.classList.remove('active');
      const href = item.getAttribute('href');

      // Exact match for dashboard, starts-with for other pages
      if (path === '/admin/' && href === '/admin/') {
        item.classList.add('active');
      } else if (href !== '/admin/' && path.startsWith(href)) {
        item.classList.add('active');
      }
    });
  },

  // Setup mobile navigation toggle
  setupMobileNav() {
    const sidebar = document.getElementById('admin-sidebar');
    const toggle = document.getElementById('sidebar-toggle');

    if (toggle && sidebar) {
      // Close sidebar when clicking outside on mobile
      document.addEventListener('click', (e) => {
        if (window.innerWidth <= 768) {
          if (!sidebar.contains(e.target) && !toggle.contains(e.target)) {
            sidebar.classList.remove('open');
          }
        }
      });
    }
  },

  // Show toast notification
  toast(message, type = 'info') {
    const container = document.getElementById('toast-container') || this.createToastContainer();

    const toast = document.createElement('div');
    toast.className = `admin-toast admin-toast-${type}`;
    toast.innerHTML = `
      <span>${message}</span>
      <button onclick="this.parentElement.remove()">&times;</button>
    `;

    container.appendChild(toast);

    setTimeout(() => {
      toast.classList.add('fade-out');
      setTimeout(() => toast.remove(), 300);
    }, 5000);
  },

  createToastContainer() {
    const container = document.createElement('div');
    container.id = 'toast-container';
    container.style.cssText = `
      position: fixed;
      top: 1rem;
      right: 1rem;
      z-index: 9999;
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
    `;
    document.body.appendChild(container);
    return container;
  },

  // Confirm dialog
  confirm(message) {
    return new Promise((resolve) => {
      const result = window.confirm(message);
      resolve(result);
    });
  },

  // Format number with K/M suffix
  formatNumber(num) {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
  },

  // Format date relative
  formatDate(dateStr) {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now - date;
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (days === 0) return 'Today';
    if (days === 1) return 'Yesterday';
    if (days < 7) return days + ' days ago';
    if (days < 30) return Math.floor(days / 7) + ' weeks ago';
    return date.toLocaleDateString();
  },

  // Format date full
  formatDateFull(dateStr) {
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  },

  // Escape HTML
  escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text || '';
    return div.innerHTML;
  },

  // Debounce function
  debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
      const later = () => {
        clearTimeout(timeout);
        func(...args);
      };
      clearTimeout(timeout);
      timeout = setTimeout(later, wait);
    };
  }
};

// Modal management
const modal = {
  open(id) {
    const el = document.getElementById(id);
    if (el) {
      el.classList.remove('hidden');
      el.style.display = 'flex';
      document.body.style.overflow = 'hidden';
    }
  },

  close(id) {
    const el = document.getElementById(id);
    if (el) {
      el.classList.add('hidden');
      el.style.display = 'none';
      document.body.style.overflow = '';
      // Clear any error messages
      const errorEl = el.querySelector('.admin-form-error');
      if (errorEl) errorEl.textContent = '';
    }
  },

  // Close on backdrop click
  setupBackdropClose() {
    document.querySelectorAll('.admin-modal-backdrop').forEach(backdrop => {
      backdrop.addEventListener('click', (e) => {
        if (e.target === backdrop) {
          backdrop.classList.add('hidden');
          backdrop.style.display = 'none';
          document.body.style.overflow = '';
        }
      });
    });
  }
};

// Toggle sidebar for mobile
function toggleSidebar() {
  const sidebar = document.getElementById('admin-sidebar');
  if (sidebar) {
    sidebar.classList.toggle('open');
  }
}

// Add toast styles dynamically
const toastStyles = document.createElement('style');
toastStyles.textContent = `
  .admin-toast {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.75rem 1rem;
    border-radius: 0.5rem;
    background: white;
    color: #1f2937;
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    min-width: 280px;
    animation: slideIn 0.3s ease;
  }

  .admin-toast-success {
    border-left: 4px solid #22c55e;
  }

  .admin-toast-error {
    border-left: 4px solid #ef4444;
  }

  .admin-toast-warning {
    border-left: 4px solid #f59e0b;
  }

  .admin-toast-info {
    border-left: 4px solid #3b82f6;
  }

  .admin-toast button {
    background: none;
    border: none;
    font-size: 1.25rem;
    cursor: pointer;
    color: #9ca3af;
    padding: 0;
    line-height: 1;
  }

  .admin-toast button:hover {
    color: #374151;
  }

  .admin-toast.fade-out {
    animation: slideOut 0.3s ease forwards;
  }

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

  @keyframes slideOut {
    from {
      transform: translateX(0);
      opacity: 1;
    }
    to {
      transform: translateX(100%);
      opacity: 0;
    }
  }
`;
document.head.appendChild(toastStyles);
