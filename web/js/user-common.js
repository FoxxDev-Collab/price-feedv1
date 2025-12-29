/**
 * User Common JavaScript
 * Shared functionality for all user pages
 */

// User namespace
const user = {
  // Current user info
  currentUser: null,

  // Initialize user panel
  async init() {
    await auth.init();

    if (!auth.requireAuth()) {
      return false;
    }

    this.currentUser = auth.user;
    this.updateUserInfo();
    this.highlightCurrentNav();
    this.setupMobileNav();

    return true;
  },

  // Update user info in sidebar
  updateUserInfo() {
    const u = this.currentUser;
    if (!u) return;

    const nameEl = document.getElementById('user-name');
    const avatarEl = document.getElementById('user-avatar');
    const regionEl = document.getElementById('user-region');

    if (nameEl) {
      nameEl.textContent = u.username || u.email.split('@')[0];
    }
    if (avatarEl) {
      avatarEl.textContent = (u.username || u.email)[0].toUpperCase();
    }
    if (regionEl) {
      regionEl.textContent = u.region_name || 'No region set';
    }
  },

  // Highlight current navigation item
  highlightCurrentNav() {
    const path = window.location.pathname;
    const navItems = document.querySelectorAll('.user-nav-item');

    navItems.forEach(item => {
      item.classList.remove('active');
      const href = item.getAttribute('href');

      // Exact match for dashboard, starts-with for other pages
      if (path === '/user/' && href === '/user/') {
        item.classList.add('active');
      } else if (href !== '/user/' && path.startsWith(href)) {
        item.classList.add('active');
      }
    });
  },

  // Setup mobile navigation toggle
  setupMobileNav() {
    const sidebar = document.getElementById('user-sidebar');
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
    toast.className = `user-toast user-toast-${type}`;

    // Create elements safely to prevent XSS
    const span = document.createElement('span');
    span.textContent = message; // Use textContent to prevent XSS

    const button = document.createElement('button');
    button.textContent = '\u00D7'; // × character
    button.onclick = function() { this.parentElement.remove(); };

    toast.appendChild(span);
    toast.appendChild(button);

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

  // Format currency
  formatCurrency(amount) {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
    }).format(amount);
  },

  // Format date relative
  formatDate(dateStr) {
    if (!dateStr) return '-';
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
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  },

  // Format date for display (short)
  formatDateShort(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric'
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
  },

  // Get user ID from current user
  getUserId() {
    return this.currentUser?.id;
  },

  // Check if user can edit a resource
  canEdit(createdBy) {
    return createdBy === this.getUserId();
  }
};

// Modal management for user pages
const userModal = {
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
      const errorEl = el.querySelector('.user-form-error');
      if (errorEl) errorEl.textContent = '';
      // Reset form if exists
      const form = el.querySelector('form');
      if (form) form.reset();
    }
  },

  // Close on backdrop click
  setupBackdropClose() {
    document.querySelectorAll('.user-modal-backdrop').forEach(backdrop => {
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

// Tab management
const userTabs = {
  init(containerId) {
    const container = document.getElementById(containerId);
    if (!container) return;

    const tabs = container.querySelectorAll('.user-tab');
    const contents = container.querySelectorAll('.user-tab-content');

    tabs.forEach(tab => {
      tab.addEventListener('click', () => {
        const targetId = tab.dataset.tab;

        // Update tabs
        tabs.forEach(t => t.classList.remove('active'));
        tab.classList.add('active');

        // Update content
        contents.forEach(c => {
          c.classList.remove('active');
          if (c.id === targetId) {
            c.classList.add('active');
          }
        });

        // Trigger custom event
        container.dispatchEvent(new CustomEvent('tabchange', {
          detail: { tabId: targetId }
        }));
      });
    });
  },

  setActive(containerId, tabId) {
    const container = document.getElementById(containerId);
    if (!container) return;

    const tab = container.querySelector(`.user-tab[data-tab="${tabId}"]`);
    if (tab) tab.click();
  }
};

// Expandable rows
const userExpandable = {
  toggle(element) {
    const row = element.closest('.user-expandable-row');
    const header = row.querySelector('.user-expandable-header');
    const content = row.querySelector('.user-expandable-content');

    header.classList.toggle('expanded');
    content.classList.toggle('expanded');
  },

  expand(rowId) {
    const row = document.getElementById(rowId);
    if (row) {
      row.querySelector('.user-expandable-header').classList.add('expanded');
      row.querySelector('.user-expandable-content').classList.add('expanded');
    }
  },

  collapse(rowId) {
    const row = document.getElementById(rowId);
    if (row) {
      row.querySelector('.user-expandable-header').classList.remove('expanded');
      row.querySelector('.user-expandable-content').classList.remove('expanded');
    }
  }
};

// Toggle sidebar for mobile
function toggleSidebar() {
  const sidebar = document.getElementById('user-sidebar');
  if (sidebar) {
    sidebar.classList.toggle('open');
  }
}

// Add toast styles dynamically
const userToastStyles = document.createElement('style');
userToastStyles.textContent = `
  .user-toast {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.75rem 1rem;
    border-radius: 0.5rem;
    background: var(--card, white);
    color: var(--foreground, #1f2937);
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    min-width: 280px;
    animation: slideIn 0.3s ease;
    border: 1px solid var(--border, #e5e7eb);
  }

  .dark .user-toast {
    box-shadow: 0 4px 12px rgba(0,0,0,0.4);
  }

  .user-toast-success {
    border-left: 4px solid #22c55e;
  }

  .user-toast-error {
    border-left: 4px solid #ef4444;
  }

  .user-toast-warning {
    border-left: 4px solid #f59e0b;
  }

  .user-toast-info {
    border-left: 4px solid #3b82f6;
  }

  .user-toast button {
    background: none;
    border: none;
    font-size: 1.25rem;
    cursor: pointer;
    color: var(--muted-foreground, #9ca3af);
    padding: 0;
    line-height: 1;
  }

  .user-toast button:hover {
    color: var(--foreground, #374151);
  }

  .user-toast.fade-out {
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
document.head.appendChild(userToastStyles);
