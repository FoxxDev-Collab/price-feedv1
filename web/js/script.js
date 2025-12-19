/**
 * PRICE FEED - Main Script
 * Vanilla JS component system (shadcn/Next.js inspired)
 *
 * Architecture:
 * - Components are defined as factory functions that return HTML strings
 * - Components can be composed together
 * - State is managed through simple patterns (localStorage, DOM)
 * - No build step, no bundler, just vanilla JS
 */

// ============================================
// UTILITY FUNCTIONS
// ============================================

/**
 * Escape HTML to prevent XSS attacks
 * @param {string} text - Text to escape
 * @returns {string} Escaped HTML string
 */
function escapeHtml(text) {
  if (text == null) return '';
  const div = document.createElement('div');
  div.textContent = String(text);
  return div.innerHTML;
}

/**
 * Generate a unique ID
 * @param {string} prefix - Optional prefix
 * @returns {string} Unique ID
 */
function generateId(prefix = 'pf') {
  return `${prefix}-${Math.random().toString(36).substr(2, 9)}`;
}

/**
 * Debounce function calls
 * @param {Function} func - Function to debounce
 * @param {number} wait - Wait time in ms
 * @returns {Function} Debounced function
 */
function debounce(func, wait = 300) {
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

/**
 * Format currency
 * @param {number} amount - Amount to format
 * @returns {string} Formatted currency string
 */
function formatCurrency(amount) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(amount);
}

/**
 * Format relative time
 * @param {Date|string} date - Date to format
 * @returns {string} Relative time string
 */
function formatRelativeTime(date) {
  const now = new Date();
  const past = new Date(date);
  const diffMs = now - past;
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  return past.toLocaleDateString();
}

/**
 * Get initials from a name
 * @param {string} name - Full name
 * @returns {string} Initials (max 2 chars)
 */
function getInitials(name) {
  if (!name) return '??';
  return name
    .split(' ')
    .map(word => word[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);
}

// ============================================
// COMPONENT SYSTEM
// ============================================

/**
 * Component registry - stores component definitions
 */
const Components = {};

/**
 * Register a component
 * @param {string} name - Component name
 * @param {Function} renderFn - Render function that returns HTML string
 */
function registerComponent(name, renderFn) {
  Components[name] = renderFn;
}

/**
 * Render a component by name
 * @param {string} name - Component name
 * @param {Object} props - Component props
 * @returns {string} HTML string
 */
function renderComponent(name, props = {}) {
  if (!Components[name]) {
    console.error(`Component "${name}" not found`);
    return '';
  }
  return Components[name](props);
}

/**
 * Mount component HTML to a container
 * @param {string|Element} container - Container selector or element
 * @param {string} html - HTML to mount
 */
function mount(container, html) {
  const el = typeof container === 'string'
    ? document.querySelector(container)
    : container;
  if (el) {
    el.innerHTML = html;
  }
}

/**
 * Append component HTML to a container
 * @param {string|Element} container - Container selector or element
 * @param {string} html - HTML to append
 */
function append(container, html) {
  const el = typeof container === 'string'
    ? document.querySelector(container)
    : container;
  if (el) {
    el.insertAdjacentHTML('beforeend', html);
  }
}

// ============================================
// SVG ICONS (inline for performance)
// ============================================

const Icons = {
  tag: `<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"></path>
  </svg>`,

  tagSmall: `<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"></path>
  </svg>`,

  users: `<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
    <path d="M9 6a3 3 0 11-6 0 3 3 0 016 0zM17 6a3 3 0 11-6 0 3 3 0 016 0zM12.93 17c.046-.327.07-.66.07-1a6.97 6.97 0 00-1.5-4.33A5 5 0 0119 16v1h-6.07zM6 11a5 5 0 015 5v1H1v-1a5 5 0 015-5z"></path>
  </svg>`,

  lock: `<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
    <path fill-rule="evenodd" d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z" clip-rule="evenodd"></path>
  </svg>`,

  community: `<svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"></path>
  </svg>`,

  list: `<svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01"></path>
  </svg>`,

  chart: `<svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"></path>
  </svg>`,

  menu: `<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path>
  </svg>`,

  close: `<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
  </svg>`,

  check: `<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
    <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
  </svg>`,
};

// ============================================
// COMPONENT DEFINITIONS
// ============================================

/**
 * Button Component
 * @param {Object} props
 * @param {string} props.text - Button text
 * @param {string} props.variant - 'primary' | 'secondary' | 'outline' | 'ghost'
 * @param {string} props.size - 'sm' | 'md' | 'lg'
 * @param {string} props.href - If provided, renders as anchor
 * @param {string} props.className - Additional classes
 * @param {string} props.id - Button ID
 * @param {string} props.type - Button type
 * @param {boolean} props.shadow - Add shadow
 */
registerComponent('Button', ({
  text = '',
  variant = 'primary',
  size = 'md',
  href,
  className = '',
  id = '',
  type = 'button',
  shadow = false,
  icon = '',
}) => {
  const sizeClass = size === 'sm' ? 'btn-sm' : size === 'lg' ? 'btn-lg' : '';
  const shadowClass = shadow ? 'btn-shadow' : '';
  const classes = `btn btn-${variant} ${sizeClass} ${shadowClass} ${className}`.trim();
  const idAttr = id ? `id="${id}"` : '';

  const content = icon
    ? `<span class="btn-icon">${icon}</span>${escapeHtml(text)}`
    : escapeHtml(text);

  if (href) {
    return `<a href="${escapeHtml(href)}" class="${classes}" ${idAttr}>${content}</a>`;
  }
  return `<button type="${type}" class="${classes}" ${idAttr}>${content}</button>`;
});

/**
 * Badge Component
 * @param {Object} props
 * @param {string} props.text - Badge text
 * @param {string} props.variant - 'primary' | 'secondary' | 'accent' | 'outline'
 * @param {string} props.size - 'sm' | 'lg'
 */
registerComponent('Badge', ({
  text = '',
  variant = 'primary',
  size = 'sm',
  className = '',
}) => {
  const sizeClass = size === 'lg' ? 'badge-lg' : '';
  const classes = `badge badge-${variant} ${sizeClass} ${className}`.trim();
  return `<span class="${classes}">${escapeHtml(text)}</span>`;
});

/**
 * Avatar Component
 * @param {Object} props
 * @param {string} props.name - User name (for initials)
 * @param {string} props.src - Image source (optional)
 * @param {string} props.size - 'sm' | 'md' | 'lg' | 'xl'
 * @param {string} props.variant - 'primary' | 'secondary' | 'accent'
 */
registerComponent('Avatar', ({
  name = '',
  src = '',
  size = 'md',
  variant = 'primary',
  className = '',
}) => {
  const classes = `avatar avatar-${size} avatar-${variant} ${className}`.trim();

  if (src) {
    return `<img src="${escapeHtml(src)}" alt="${escapeHtml(name)}" class="${classes}">`;
  }

  return `<div class="${classes}">${escapeHtml(getInitials(name))}</div>`;
});

/**
 * Icon Box Component
 * @param {Object} props
 * @param {string} props.icon - Icon HTML
 * @param {string} props.size - 'sm' | 'md' | 'lg'
 * @param {string} props.variant - 'primary' | 'secondary' | 'accent' | 'white'
 */
registerComponent('IconBox', ({
  icon = '',
  size = 'lg',
  variant = 'primary',
  className = '',
}) => {
  const classes = `icon-box icon-box-${size} icon-box-${variant} ${className}`.trim();
  return `<div class="${classes}">${icon}</div>`;
});

/**
 * Feed Item Component
 * @param {Object} props
 * @param {string} props.userName - User name
 * @param {string} props.time - Timestamp
 * @param {string} props.text - Feed text
 * @param {string} props.price - Price value
 * @param {string} props.priceLabel - Price label/unit
 * @param {string} props.variant - 'primary' | 'secondary' | 'accent'
 * @param {string} props.badge - Optional badge text
 */
registerComponent('FeedItem', ({
  userName = '',
  time = '',
  text = '',
  price = '',
  priceLabel = '',
  variant = 'primary',
  badge = '',
  oldPrice = '',
}) => {
  const avatarVariant = variant === 'primary' ? 'primary'
    : variant === 'secondary' ? 'secondary'
    : 'accent';

  const badgeHtml = badge
    ? `<span class="badge badge-${variant}">${escapeHtml(badge)}</span>`
    : '';

  const oldPriceHtml = oldPrice
    ? `<span class="text-xs line-through text-gray-400">${escapeHtml(oldPrice)}</span>`
    : '';

  const priceLabelHtml = priceLabel && !badge
    ? `<span class="text-xs text-gray-500">${escapeHtml(priceLabel)}</span>`
    : '';

  return `
    <div class="feed-item feed-item-${variant}">
      ${renderComponent('Avatar', { name: userName, size: 'md', variant: avatarVariant })}
      <div class="feed-item-content">
        <div class="feed-item-header">
          <span class="feed-item-user">${escapeHtml(userName)}</span>
          <span class="feed-item-time">${escapeHtml(time)}</span>
        </div>
        <p class="feed-item-text">${escapeHtml(text)}</p>
        <div class="feed-item-price">
          <span class="feed-item-price-value ${variant}">${escapeHtml(price)}</span>
          ${priceLabelHtml}
          ${oldPriceHtml}
          ${badgeHtml}
        </div>
      </div>
    </div>
  `;
});

/**
 * Stat Component
 * @param {Object} props
 * @param {string} props.value - Stat value
 * @param {string} props.label - Stat label
 * @param {string} props.variant - 'primary' | 'secondary' | 'accent'
 */
registerComponent('Stat', ({
  value = '',
  label = '',
  variant = 'primary',
}) => {
  return `
    <div class="stat">
      <div class="stat-value ${variant}">${escapeHtml(value)}</div>
      <div class="stat-label">${escapeHtml(label)}</div>
    </div>
  `;
});

/**
 * Feature Card Component
 * @param {Object} props
 * @param {string} props.icon - Icon HTML
 * @param {string} props.iconVariant - 'primary' | 'secondary' | 'accent'
 * @param {string} props.title - Card title
 * @param {string} props.description - Card description
 */
registerComponent('FeatureCard', ({
  icon = '',
  iconVariant = 'primary',
  title = '',
  description = '',
}) => {
  return `
    <div class="feature-card">
      <div class="feature-card-icon">
        ${renderComponent('IconBox', { icon, variant: iconVariant, size: 'lg' })}
      </div>
      <h3 class="feature-card-title">${escapeHtml(title)}</h3>
      <p class="feature-card-description">${escapeHtml(description)}</p>
    </div>
  `;
});

/**
 * Section Header Component
 * @param {Object} props
 * @param {string} props.title - Section title
 * @param {string} props.subtitle - Section subtitle
 */
registerComponent('SectionHeader', ({
  title = '',
  subtitle = '',
}) => {
  return `
    <div class="section-header">
      <h2 class="section-title">${escapeHtml(title)}</h2>
      ${subtitle ? `<p class="section-subtitle">${escapeHtml(subtitle)}</p>` : ''}
    </div>
  `;
});

/**
 * Navbar Component
 * Renders the main navigation bar
 */
registerComponent('Navbar', ({
  brandName = 'Price Feed',
  links = [],
  showAuth = true,
}) => {
  const linksHtml = links.map(link =>
    `<a href="${escapeHtml(link.href)}" class="navbar-link">${escapeHtml(link.text)}</a>`
  ).join('');

  const authHtml = showAuth ? `
    <div class="navbar-actions">
      <button class="btn btn-ghost">Sign In</button>
      ${renderComponent('Button', { text: 'Get Started Free', variant: 'primary' })}
    </div>
  ` : '';

  return `
    <nav class="navbar">
      <div class="container navbar-container">
        <a href="/" class="navbar-brand">
          <div class="navbar-logo">${Icons.tagSmall}</div>
          <span class="navbar-title">${escapeHtml(brandName)}</span>
        </a>
        <div class="navbar-links">
          ${linksHtml}
        </div>
        ${authHtml}
        <button class="navbar-mobile-btn" aria-label="Toggle menu">
          ${Icons.menu}
        </button>
      </div>
    </nav>
  `;
});

/**
 * Footer Component
 */
registerComponent('Footer', ({
  brandName = 'Price Feed',
  tagline = 'Community-driven grocery price comparison for smart shoppers.',
  columns = [],
  copyright = '',
}) => {
  const columnsHtml = columns.map(col => `
    <div>
      <h4 class="footer-heading">${escapeHtml(col.title)}</h4>
      <ul class="footer-links">
        ${col.links.map(link =>
          `<li><a href="${escapeHtml(link.href)}" class="footer-link">${escapeHtml(link.text)}</a></li>`
        ).join('')}
      </ul>
    </div>
  `).join('');

  return `
    <footer class="footer">
      <div class="container">
        <div class="footer-grid">
          <div>
            <div class="footer-brand">
              <div class="footer-logo">${Icons.tagSmall}</div>
              <span class="footer-title">${escapeHtml(brandName)}</span>
            </div>
            <p class="footer-description">${escapeHtml(tagline)}</p>
          </div>
          ${columnsHtml}
        </div>
        <div class="footer-bottom">
          <p>${escapeHtml(copyright)}</p>
        </div>
      </div>
    </footer>
  `;
});

/**
 * Modal Component
 * @param {Object} props
 * @param {string} props.id - Modal ID
 * @param {string} props.title - Modal title
 * @param {string} props.body - Modal body content (HTML)
 * @param {boolean} props.showFooter - Show footer with buttons
 */
registerComponent('Modal', ({
  id = 'modal',
  title = '',
  body = '',
  showFooter = true,
  confirmText = 'Confirm',
  cancelText = 'Cancel',
}) => {
  const footerHtml = showFooter ? `
    <div class="modal-footer">
      ${renderComponent('Button', { text: cancelText, variant: 'outline', className: 'modal-cancel' })}
      ${renderComponent('Button', { text: confirmText, variant: 'primary', className: 'modal-confirm' })}
    </div>
  ` : '';

  return `
    <div class="modal-backdrop hidden" id="${escapeHtml(id)}-backdrop">
      <div class="modal" id="${escapeHtml(id)}">
        <div class="modal-header">
          <h3 class="modal-title">${escapeHtml(title)}</h3>
          <button class="modal-close" aria-label="Close modal">${Icons.close}</button>
        </div>
        <div class="modal-body">
          ${body}
        </div>
        ${footerHtml}
      </div>
    </div>
  `;
});

/**
 * Form Input Component
 * @param {Object} props
 */
registerComponent('FormInput', ({
  id = '',
  name = '',
  label = '',
  type = 'text',
  placeholder = '',
  required = false,
  error = '',
  value = '',
}) => {
  const errorHtml = error ? `<p class="form-error">${escapeHtml(error)}</p>` : '';
  const requiredAttr = required ? 'required' : '';

  return `
    <div class="form-group">
      ${label ? `<label for="${escapeHtml(id)}" class="form-label">${escapeHtml(label)}</label>` : ''}
      <input
        type="${type}"
        id="${escapeHtml(id)}"
        name="${escapeHtml(name || id)}"
        class="form-input"
        placeholder="${escapeHtml(placeholder)}"
        value="${escapeHtml(value)}"
        ${requiredAttr}
      >
      ${errorHtml}
    </div>
  `;
});

/**
 * Spinner Component
 */
registerComponent('Spinner', ({
  size = 'md',
}) => {
  const sizeClass = size === 'sm' ? 'spinner-sm' : size === 'lg' ? 'spinner-lg' : '';
  return `<div class="spinner ${sizeClass}"></div>`;
});

/**
 * Empty State Component
 */
registerComponent('EmptyState', ({
  icon = '',
  title = '',
  description = '',
  actionText = '',
  actionHref = '',
}) => {
  const actionHtml = actionText
    ? renderComponent('Button', { text: actionText, href: actionHref, variant: 'primary' })
    : '';

  return `
    <div class="empty-state">
      ${icon ? `<div class="empty-state-icon">${icon}</div>` : ''}
      <h3 class="empty-state-title">${escapeHtml(title)}</h3>
      <p class="empty-state-description">${escapeHtml(description)}</p>
      ${actionHtml}
    </div>
  `;
});

// ============================================
// MODAL HELPERS
// ============================================

/**
 * Open a modal by ID
 * @param {string} id - Modal ID
 */
function openModal(id) {
  const backdrop = document.getElementById(`${id}-backdrop`);
  if (backdrop) {
    backdrop.classList.remove('hidden');
    document.body.style.overflow = 'hidden';
  }
}

/**
 * Close a modal by ID
 * @param {string} id - Modal ID
 */
function closeModal(id) {
  const backdrop = document.getElementById(`${id}-backdrop`);
  if (backdrop) {
    backdrop.classList.add('hidden');
    document.body.style.overflow = '';
  }
}

/**
 * Initialize modal event listeners
 */
function initModals() {
  document.addEventListener('click', (e) => {
    // Close on backdrop click
    if (e.target.classList.contains('modal-backdrop')) {
      e.target.classList.add('hidden');
      document.body.style.overflow = '';
    }

    // Close on close button click
    if (e.target.closest('.modal-close')) {
      const modal = e.target.closest('.modal-backdrop');
      if (modal) {
        modal.classList.add('hidden');
        document.body.style.overflow = '';
      }
    }

    // Close on cancel button click
    if (e.target.closest('.modal-cancel')) {
      const modal = e.target.closest('.modal-backdrop');
      if (modal) {
        modal.classList.add('hidden');
        document.body.style.overflow = '';
      }
    }
  });

  // Close on Escape key
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      const openModals = document.querySelectorAll('.modal-backdrop:not(.hidden)');
      openModals.forEach(modal => {
        modal.classList.add('hidden');
      });
      document.body.style.overflow = '';
    }
  });
}

// ============================================
// MOBILE NAVIGATION
// ============================================

function initMobileNav() {
  const mobileBtn = document.querySelector('.navbar-mobile-btn');
  const navLinks = document.querySelector('.navbar-links');

  if (mobileBtn && navLinks) {
    mobileBtn.addEventListener('click', () => {
      navLinks.classList.toggle('active');
    });
  }
}

// ============================================
// INITIALIZATION
// ============================================

/**
 * Initialize all interactive components
 */
function initComponents() {
  initModals();
  initMobileNav();
}

// Auto-initialize when DOM is ready
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initComponents);
} else {
  initComponents();
}

// ============================================
// EXPORTS (for use in other scripts)
// ============================================

// Expose to global scope for vanilla JS usage
window.PriceFeed = {
  // Utilities
  escapeHtml,
  generateId,
  debounce,
  formatCurrency,
  formatRelativeTime,
  getInitials,

  // Component system
  Components,
  registerComponent,
  renderComponent,
  mount,
  append,

  // Icons
  Icons,

  // Modal helpers
  openModal,
  closeModal,
};
