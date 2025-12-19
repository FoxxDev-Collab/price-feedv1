/**
 * Price Feed Authentication Module
 * Handles login state, UI updates, and route protection
 */

const auth = {
  user: null,
  initialized: false,

  /**
   * Initialize auth state on page load
   */
  async init() {
    if (this.initialized) return this.user;

    const token = localStorage.getItem('token');
    if (!token) {
      this.initialized = true;
      this.updateUI();
      return null;
    }

    try {
      const response = await authApi.getCurrentUser();
      this.user = response?.data || response?.user || response;
      this.initialized = true;
      this.updateUI();
      return this.user;
    } catch (err) {
      console.error('Auth init failed:', err);
      this.logout(false); // Don't redirect
      this.initialized = true;
      return null;
    }
  },

  /**
   * Login user
   */
  async login(email, password) {
    try {
      const response = await authApi.login(email, password);
      this.user = response.user;
      this.updateUI();
      return { success: true, user: this.user };
    } catch (err) {
      return { success: false, error: err.message };
    }
  },

  /**
   * Register new user
   */
  async register(email, password, username = null, regionId = null) {
    try {
      const response = await authApi.register(email, password, username, regionId);
      this.user = response.user;
      this.updateUI();
      return { success: true, user: this.user };
    } catch (err) {
      return { success: false, error: err.message };
    }
  },

  /**
   * Logout user
   */
  logout(redirect = true) {
    localStorage.removeItem('token');
    this.user = null;
    this.updateUI();
    if (redirect) {
      window.location.href = '/';
    }
  },

  /**
   * Check if user is logged in
   */
  isLoggedIn() {
    return !!this.user || !!localStorage.getItem('token');
  },

  /**
   * Check if user is admin
   */
  isAdmin() {
    return this.user?.role === 'admin';
  },

  /**
   * Check if user is moderator or admin
   */
  isModerator() {
    return this.user?.role === 'admin' || this.user?.role === 'moderator';
  },

  /**
   * Require authentication - redirect to login if not authenticated
   */
  requireAuth() {
    if (!this.isLoggedIn()) {
      // Store intended destination
      sessionStorage.setItem('redirectAfterLogin', window.location.href);
      window.location.href = '/login/';
      return false;
    }
    return true;
  },

  /**
   * Require admin role - redirect if not admin
   */
  requireAdmin() {
    if (!this.requireAuth()) return false;

    if (!this.isAdmin()) {
      window.location.href = '/user/';
      return false;
    }
    return true;
  },

  /**
   * Update UI based on auth state
   */
  updateUI() {
    // Update navbar auth buttons
    const authButtons = document.getElementById('auth-buttons');
    const userMenu = document.getElementById('user-menu');
    const adminLink = document.getElementById('admin-link');

    if (this.user) {
      // Hide login/register buttons
      if (authButtons) authButtons.classList.add('hidden');

      // Show user menu
      if (userMenu) {
        userMenu.classList.remove('hidden');
        const usernameEl = userMenu.querySelector('.username');
        if (usernameEl) {
          usernameEl.textContent = this.user.username || this.user.email.split('@')[0];
        }
      }

      // Show/hide admin link
      if (adminLink) {
        if (this.isAdmin()) {
          adminLink.classList.remove('hidden');
        } else {
          adminLink.classList.add('hidden');
        }
      }
    } else {
      // Show login/register buttons
      if (authButtons) authButtons.classList.remove('hidden');

      // Hide user menu
      if (userMenu) userMenu.classList.add('hidden');

      // Hide admin link
      if (adminLink) adminLink.classList.add('hidden');
    }

    // Dispatch custom event for other components
    window.dispatchEvent(new CustomEvent('authStateChanged', {
      detail: { user: this.user, isLoggedIn: this.isLoggedIn() }
    }));
  },

  /**
   * Handle login form submission
   */
  async handleLoginForm(form) {
    const email = form.querySelector('[name="email"]').value;
    const password = form.querySelector('[name="password"]').value;
    const errorEl = form.querySelector('.error-message');
    const submitBtn = form.querySelector('button[type="submit"]');

    // Clear previous errors
    if (errorEl) errorEl.textContent = '';

    // Disable submit button
    if (submitBtn) {
      submitBtn.disabled = true;
      submitBtn.textContent = 'Logging in...';
    }

    const result = await this.login(email, password);

    if (result.success) {
      // Redirect admins to admin panel, others to user dashboard
      const defaultRedirect = this.isAdmin() ? '/admin/' : '/user/';
      const redirect = sessionStorage.getItem('redirectAfterLogin') || defaultRedirect;
      sessionStorage.removeItem('redirectAfterLogin');
      window.location.href = redirect;
    } else {
      if (errorEl) errorEl.textContent = result.error;
      if (submitBtn) {
        submitBtn.disabled = false;
        submitBtn.textContent = 'Login';
      }
    }
  },

  /**
   * Handle register form submission
   */
  async handleRegisterForm(form) {
    const email = form.querySelector('[name="email"]').value;
    const password = form.querySelector('[name="password"]').value;
    const confirmPassword = form.querySelector('[name="confirmPassword"]')?.value;
    const username = form.querySelector('[name="username"]')?.value || null;
    const regionId = form.querySelector('[name="region_id"]')?.value || null;
    const errorEl = form.querySelector('.error-message');
    const submitBtn = form.querySelector('button[type="submit"]');

    // Clear previous errors
    if (errorEl) errorEl.textContent = '';

    // Validate passwords match
    if (confirmPassword && password !== confirmPassword) {
      if (errorEl) errorEl.textContent = 'Passwords do not match';
      return;
    }

    // Validate password length
    if (password.length < 8) {
      if (errorEl) errorEl.textContent = 'Password must be at least 8 characters';
      return;
    }

    // Disable submit button
    if (submitBtn) {
      submitBtn.textContent = 'Creating account...';
    }

    const result = await this.register(email, password, username, regionId);

    if (result.success) {
      window.location.href = '/user/';
    } else {
      if (errorEl) errorEl.textContent = result.error;
      if (submitBtn) {
        submitBtn.disabled = false;
        submitBtn.textContent = 'Create Account';
      }
    }
  },
};

// Initialize auth on DOMContentLoaded
document.addEventListener('DOMContentLoaded', () => {
  auth.init();
});

// Handle logout button clicks
document.addEventListener('click', (e) => {
  if (e.target.matches('[data-logout], .logout-btn')) {
    e.preventDefault();
    auth.logout();
  }
});
