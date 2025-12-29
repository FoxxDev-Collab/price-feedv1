/**
 * User Sidebar Component
 * Renders the sidebar navigation for all user pages
 */

function renderUserSidebar() {
  return `
    <div class="user-sidebar-header">
      <a href="/user/" class="user-logo">
        <div class="user-logo-icon">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
          </svg>
        </div>
        <span class="user-logo-text">PriceFeed</span>
      </a>
    </div>

    <nav class="user-nav">
      <div class="user-nav-section">
        <div class="user-nav-label">Overview</div>
        <a href="/user/" class="user-nav-item" data-page="dashboard">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="7" height="7"></rect>
            <rect x="14" y="3" width="7" height="7"></rect>
            <rect x="14" y="14" width="7" height="7"></rect>
            <rect x="3" y="14" width="7" height="7"></rect>
          </svg>
          Dashboard
        </a>
      </div>

      <div class="user-nav-section">
        <div class="user-nav-label">My Data</div>
        <a href="/user/stores/" class="user-nav-item" data-page="stores">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path>
            <polyline points="9 22 9 12 15 12 15 22"></polyline>
          </svg>
          Stores
        </a>
        <a href="/user/items/" class="user-nav-item" data-page="items">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M6 2L3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4z"></path>
            <line x1="3" y1="6" x2="21" y2="6"></line>
            <path d="M16 10a4 4 0 0 1-8 0"></path>
          </svg>
          Items
        </a>
        <a href="/user/prices/" class="user-nav-item" data-page="prices">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="1" x2="12" y2="23"></line>
            <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"></path>
          </svg>
          Prices
        </a>
        <a href="/user/receipts/" class="user-nav-item" data-page="receipts">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
            <polyline points="14 2 14 8 20 8"></polyline>
            <line x1="16" y1="13" x2="8" y2="13"></line>
            <line x1="16" y1="17" x2="8" y2="17"></line>
            <polyline points="10 9 9 9 8 9"></polyline>
          </svg>
          Receipts
        </a>
        <a href="/user/inventory/" class="user-nav-item" data-page="inventory">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path>
            <polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline>
            <line x1="12" y1="22.08" x2="12" y2="12"></line>
          </svg>
          Inventory
        </a>
      </div>

      <div class="user-nav-section">
        <div class="user-nav-label">Shopping</div>
        <a href="/user/lists/" class="user-nav-item" data-page="lists">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M9 11l3 3L22 4"></path>
            <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"></path>
          </svg>
          Shopping Lists
        </a>
        <a href="/user/compare/" class="user-nav-item" data-page="compare">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="20" x2="18" y2="10"></line>
            <line x1="12" y1="20" x2="12" y2="4"></line>
            <line x1="6" y1="20" x2="6" y2="14"></line>
          </svg>
          Price Comparison
        </a>
      </div>

      <div class="user-nav-section">
        <div class="user-nav-label">Account</div>
        <a href="/user/profile/" class="user-nav-item" data-page="profile">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
            <circle cx="12" cy="7" r="4"></circle>
          </svg>
          My Profile
        </a>
        <a href="/user/settings/" class="user-nav-item" data-page="settings">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3"></circle>
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
          </svg>
          Settings
        </a>
      </div>
    </nav>

    <div class="user-sidebar-footer">
      <div class="user-info">
        <div class="user-avatar" id="user-avatar">U</div>
        <div class="user-details">
          <div class="user-name" id="user-name">User</div>
          <div class="user-region" id="user-region">No region</div>
        </div>
        <button class="user-logout-btn" onclick="auth.logout()" title="Logout">
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
            <polyline points="16 17 21 12 16 7"></polyline>
            <line x1="21" y1="12" x2="9" y2="12"></line>
          </svg>
        </button>
      </div>
    </div>
  `;
}

function renderUserHeader(breadcrumbs) {
  const breadcrumbHtml = breadcrumbs.map((b, i) => {
    if (i === breadcrumbs.length - 1) {
      return `<span class="user-breadcrumb-current">${b.label}</span>`;
    }
    return `<a href="${b.href}">${b.label}</a><span class="user-breadcrumb-separator">/</span>`;
  }).join('');

  return `
    <div class="user-header-left">
      <button class="user-header-btn" id="sidebar-toggle" onclick="toggleSidebar()">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="3" y1="12" x2="21" y2="12"></line>
          <line x1="3" y1="6" x2="21" y2="6"></line>
          <line x1="3" y1="18" x2="21" y2="18"></line>
        </svg>
      </button>
      <nav class="user-breadcrumb">
        ${breadcrumbHtml}
      </nav>
    </div>
    <div class="user-header-right">
      <button class="user-header-btn theme-toggle-btn" onclick="theme.toggle()" title="Toggle theme">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
        </svg>
      </button>
      <a href="/admin/" class="user-header-btn admin-link" title="Admin Panel" id="admin-panel-link" style="display: none;">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 4.5a2.5 2.5 0 0 0-4.96-.46 2.5 2.5 0 0 0-1.98 3 2.5 2.5 0 0 0-1.32 4.24 3 3 0 0 0 .34 5.58 2.5 2.5 0 0 0 2.96 3.08 2.5 2.5 0 0 0 4.91.05L12 20V4.5Z"></path>
          <path d="M16 8V5c0-1.1.9-2 2-2"></path>
          <path d="M12 13h4"></path>
          <path d="M12 18h6a2 2 0 0 1 2 2v1"></path>
          <path d="M12 8h8"></path>
          <path d="M20.5 8a.5.5 0 1 1-1 0 .5.5 0 0 1 1 0Z"></path>
          <path d="M16.5 13a.5.5 0 1 1-1 0 .5.5 0 0 1 1 0Z"></path>
          <path d="M20.5 21a.5.5 0 1 1-1 0 .5.5 0 0 1 1 0Z"></path>
          <path d="M18.5 3a.5.5 0 1 1-1 0 .5.5 0 0 1 1 0Z"></path>
        </svg>
      </a>
      <a href="/" class="user-header-btn" title="Home">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path>
          <polyline points="9 22 9 12 15 12 15 22"></polyline>
        </svg>
      </a>
    </div>
  `;
}

// Initialize sidebar on any user page
function initUserSidebar(currentPage) {
  const sidebar = document.getElementById('user-sidebar');
  if (sidebar) {
    sidebar.innerHTML = renderUserSidebar();

    // Highlight current page
    const navItems = sidebar.querySelectorAll('.user-nav-item');
    navItems.forEach(item => {
      if (item.dataset.page === currentPage) {
        item.classList.add('active');
      }
    });

    // Update user info now that sidebar elements exist
    if (typeof user !== 'undefined' && user.updateUserInfo) {
      user.updateUserInfo();
    }

    // Show admin link if user is admin
    if (typeof auth !== 'undefined' && auth.user?.role === 'admin') {
      const adminLink = document.getElementById('admin-panel-link');
      if (adminLink) {
        adminLink.style.display = 'block';
      }
    }
  }
}

// Initialize header
function initUserHeader(breadcrumbs) {
  const header = document.getElementById('user-header');
  if (header) {
    header.innerHTML = renderUserHeader(breadcrumbs);

    // Show admin link if user is admin (after header is rendered)
    if (typeof auth !== 'undefined' && auth.user?.role === 'admin') {
      const adminLink = document.getElementById('admin-panel-link');
      if (adminLink) {
        adminLink.style.display = 'block';
      }
    }
  }
}
