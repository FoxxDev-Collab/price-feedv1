/**
 * Theme Management
 * Handles dark/light mode toggling with localStorage persistence
 */

const theme = {
  // Storage key
  STORAGE_KEY: 'pricefeed-theme',

  // Initialize theme on page load
  init() {
    const savedTheme = localStorage.getItem(this.STORAGE_KEY);

    if (savedTheme) {
      this.setTheme(savedTheme);
    } else {
      // Check system preference
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      this.setTheme(prefersDark ? 'dark' : 'light');
    }

    // Listen for system theme changes
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
      if (!localStorage.getItem(this.STORAGE_KEY)) {
        this.setTheme(e.matches ? 'dark' : 'light');
      }
    });
  },

  // Set theme
  setTheme(themeName) {
    if (themeName === 'dark') {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
    this.updateToggleButton();
  },

  // Toggle between light and dark
  toggle() {
    const isDark = document.documentElement.classList.contains('dark');
    const newTheme = isDark ? 'light' : 'dark';

    localStorage.setItem(this.STORAGE_KEY, newTheme);
    this.setTheme(newTheme);
  },

  // Get current theme
  current() {
    return document.documentElement.classList.contains('dark') ? 'dark' : 'light';
  },

  // Update toggle button icon
  updateToggleButton() {
    const buttons = document.querySelectorAll('.theme-toggle-btn');
    const isDark = document.documentElement.classList.contains('dark');

    buttons.forEach(btn => {
      btn.innerHTML = isDark ? this.getSunIcon() : this.getMoonIcon();
      btn.title = isDark ? 'Switch to light mode' : 'Switch to dark mode';
    });
  },

  // Moon icon (for light mode - clicking will switch to dark)
  getMoonIcon() {
    return `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
    </svg>`;
  },

  // Sun icon (for dark mode - clicking will switch to light)
  getSunIcon() {
    return `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <circle cx="12" cy="12" r="5"></circle>
      <line x1="12" y1="1" x2="12" y2="3"></line>
      <line x1="12" y1="21" x2="12" y2="23"></line>
      <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
      <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
      <line x1="1" y1="12" x2="3" y2="12"></line>
      <line x1="21" y1="12" x2="23" y2="12"></line>
      <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
      <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
    </svg>`;
  },

  // Get toggle button HTML
  getToggleButtonHtml(className = '') {
    const isDark = document.documentElement.classList.contains('dark');
    return `<button class="theme-toggle-btn ${className}" onclick="theme.toggle()" title="${isDark ? 'Switch to light mode' : 'Switch to dark mode'}">
      ${isDark ? this.getSunIcon() : this.getMoonIcon()}
    </button>`;
  }
};

// Initialize theme immediately (before DOM loads to prevent flash)
(function() {
  const savedTheme = localStorage.getItem('pricefeed-theme');
  if (savedTheme === 'dark') {
    document.documentElement.classList.add('dark');
  } else if (!savedTheme) {
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    if (prefersDark) {
      document.documentElement.classList.add('dark');
    }
  }
})();

// Full init when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  theme.updateToggleButton();
});
