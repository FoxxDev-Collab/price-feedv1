/**
 * Price Feed API Client
 * Vanilla JS wrapper for all API calls
 */

const API_BASE = '/api';

const api = {
  /**
   * Get auth token from localStorage
   */
  getToken() {
    return localStorage.getItem('token');
  },

  /**
   * Set auth token in localStorage
   */
  setToken(token) {
    localStorage.setItem('token', token);
  },

  /**
   * Remove auth token from localStorage
   */
  removeToken() {
    localStorage.removeItem('token');
  },

  /**
   * Make an authenticated request to the API
   */
  async request(endpoint, options = {}) {
    const token = this.getToken();
    const config = {
      headers: {
        'Content-Type': 'application/json',
        ...(token && { 'Authorization': `Bearer ${token}` }),
        ...options.headers,
      },
      ...options,
    };

    try {
      const response = await fetch(`${API_BASE}${endpoint}`, config);

      // Handle 401 - redirect to login
      if (response.status === 401) {
        this.removeToken();
        window.location.href = '/login/';
        return null;
      }

      // Parse JSON response
      const data = await response.json().catch(() => null);

      if (!response.ok) {
        const errorMessage = data?.error || 'Request failed';
        throw new Error(errorMessage);
      }

      return data;
    } catch (error) {
      console.error('API Error:', error);
      throw error;
    }
  },

  /**
   * GET request
   */
  get(endpoint) {
    return this.request(endpoint, { method: 'GET' });
  },

  /**
   * POST request
   */
  post(endpoint, data) {
    return this.request(endpoint, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  /**
   * PUT request
   */
  put(endpoint, data) {
    return this.request(endpoint, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  /**
   * DELETE request
   */
  delete(endpoint) {
    return this.request(endpoint, { method: 'DELETE' });
  },
};

/**
 * Authentication API
 */
const authApi = {
  /**
   * Register a new user
   */
  async register(email, password, username = null, regionId = null) {
    const data = { email, password };
    if (username) data.username = username;
    if (regionId) data.region_id = parseInt(regionId);

    const response = await api.post('/auth/register', data);
    if (response?.token) {
      api.setToken(response.token);
    }
    return response;
  },

  /**
   * Register a new user with full location data
   * @param {Object} formData - Registration data including location fields
   * @param {string} formData.email - User email
   * @param {string} formData.password - User password
   * @param {string} [formData.username] - Optional username
   * @param {number} [formData.region_id] - Optional region ID
   * @param {string} [formData.street_address] - Street address
   * @param {string} [formData.city] - City
   * @param {string} [formData.state] - State
   * @param {string} [formData.zip_code] - ZIP code
   * @param {number} [formData.latitude] - Latitude coordinate
   * @param {number} [formData.longitude] - Longitude coordinate
   * @param {string} [formData.google_place_id] - Google Place ID
   */
  async registerWithLocation(formData) {
    // Build registration data, filtering out null/undefined values
    const data = {
      email: formData.email,
      password: formData.password,
    };

    // Add optional fields if they have values
    if (formData.username) data.username = formData.username;
    if (formData.region_id) data.region_id = parseInt(formData.region_id);
    if (formData.street_address) data.street_address = formData.street_address;
    if (formData.city) data.city = formData.city;
    if (formData.state) data.state = formData.state;
    if (formData.zip_code) data.zip_code = formData.zip_code;
    if (formData.latitude) data.latitude = parseFloat(formData.latitude);
    if (formData.longitude) data.longitude = parseFloat(formData.longitude);
    if (formData.google_place_id) data.google_place_id = formData.google_place_id;

    const response = await api.post('/auth/register', data);
    if (response?.token) {
      api.setToken(response.token);
    }
    return response;
  },

  /**
   * Login user
   */
  async login(email, password) {
    const response = await api.post('/auth/login', { email, password });
    if (response?.token) {
      api.setToken(response.token);
    }
    return response;
  },

  /**
   * Logout user
   */
  async logout() {
    try {
      await api.post('/auth/logout', {});
    } finally {
      api.removeToken();
      window.location.href = '/';
    }
  },

  /**
   * Get current user
   */
  async getCurrentUser() {
    return api.get('/auth/me');
  },

  /**
   * Refresh token
   */
  async refreshToken() {
    const response = await api.post('/auth/refresh', {});
    if (response?.token) {
      api.setToken(response.token);
    }
    return response;
  },

  /**
   * Check if user is logged in
   */
  isLoggedIn() {
    return !!api.getToken();
  },
};

/**
 * User API
 */
const userApi = {
  /**
   * Get user by ID
   */
  getById(id) {
    return api.get(`/users/${id}`);
  },

  /**
   * Update user profile
   */
  update(id, data) {
    return api.put(`/users/${id}`, data);
  },

  /**
   * Get user stats
   */
  getStats(id) {
    return api.get(`/users/${id}/stats`);
  },
};

/**
 * Admin API
 */
const adminApi = {
  /**
   * Create a new user (admin)
   */
  createUser(data) {
    return api.post('/admin/users', data);
  },

  /**
   * List all users
   */
  listUsers(limit = 20, offset = 0) {
    return api.get(`/admin/users?limit=${limit}&offset=${offset}`);
  },

  /**
   * Get user by ID (admin view)
   */
  getUser(id) {
    return api.get(`/admin/users/${id}`);
  },

  /**
   * Update user (admin)
   */
  updateUser(id, data) {
    return api.put(`/admin/users/${id}`, data);
  },

  /**
   * Delete user
   */
  deleteUser(id) {
    return api.delete(`/admin/users/${id}`);
  },

  /**
   * Get system stats
   */
  getStats() {
    return api.get('/admin/stats');
  },
};

/**
 * Stores API
 */
const storesApi = {
  /**
   * List stores with pagination and filters
   */
  list(params = {}) {
    const query = new URLSearchParams();
    if (params.limit) query.set('limit', params.limit);
    if (params.offset) query.set('offset', params.offset);
    if (params.search) query.set('search', params.search);
    if (params.region_id) query.set('region_id', params.region_id);
    if (params.state) query.set('state', params.state);
    if (params.verified !== undefined) query.set('verified', params.verified);
    const queryStr = query.toString();
    return api.get(`/stores${queryStr ? '?' + queryStr : ''}`);
  },

  getById(id) {
    return api.get(`/stores/${id}`);
  },

  /**
   * Get store statistics
   */
  getStats() {
    return api.get('/stores/stats');
  },

  /**
   * Search stores
   */
  search(query, limit = 20) {
    return api.get(`/stores/search?q=${encodeURIComponent(query)}&limit=${limit}`);
  },

  /**
   * Create a new store (authenticated users)
   */
  create(data) {
    return api.post('/stores', data);
  },

  /**
   * Create a new store (admin only)
   */
  adminCreate(data) {
    return api.post('/admin/stores', data);
  },

  /**
   * Update a store
   */
  update(id, data) {
    return api.put(`/stores/${id}`, data);
  },

  /**
   * Delete a store
   */
  delete(id) {
    return api.delete(`/stores/${id}`);
  },

  /**
   * Verify a store (admin)
   */
  verify(id) {
    return api.post(`/admin/stores/${id}/verify`, {});
  },
};

/**
 * Items API
 */
const itemsApi = {
  /**
   * List items with pagination and filters
   */
  list(params = {}) {
    const query = new URLSearchParams();
    if (params.limit) query.set('limit', params.limit);
    if (params.offset) query.set('offset', params.offset);
    if (params.search) query.set('search', params.search);
    if (params.tag) query.set('tag', params.tag);
    const queryStr = query.toString();
    return api.get(`/items${queryStr ? '?' + queryStr : ''}`);
  },

  getById(id) {
    return api.get(`/items/${id}`);
  },

  /**
   * Get item statistics
   */
  getStats() {
    return api.get('/items/stats');
  },

  /**
   * Search items
   */
  search(query, limit = 20) {
    return api.get(`/items/search?q=${encodeURIComponent(query)}&limit=${limit}`);
  },

  /**
   * Create a new item
   */
  create(data) {
    return api.post('/items', data);
  },

  /**
   * Update an item
   */
  update(id, data) {
    return api.put(`/items/${id}`, data);
  },

  /**
   * Delete an item
   */
  delete(id) {
    return api.delete(`/items/${id}`);
  },
};

/**
 * Tags API
 */
const tagsApi = {
  /**
   * List all tags
   */
  list() {
    return api.get('/tags');
  },

  getAll() {
    return api.get('/tags');
  },
};

/**
 * Prices API
 */
const pricesApi = {
  /**
   * List prices with pagination and filters
   */
  list(params = {}) {
    const query = new URLSearchParams();
    if (params.limit) query.set('limit', params.limit);
    if (params.offset) query.set('offset', params.offset);
    if (params.search) query.set('search', params.search);
    if (params.store_id) query.set('store_id', params.store_id);
    if (params.item_id) query.set('item_id', params.item_id);
    if (params.region_id) query.set('region_id', params.region_id);
    if (params.verified !== undefined) query.set('verified', params.verified);
    if (params.date) query.set('date', params.date);
    const queryStr = query.toString();
    return api.get(`/prices${queryStr ? '?' + queryStr : ''}`);
  },

  getById(id) {
    return api.get(`/prices/${id}`);
  },

  /**
   * Get price statistics
   */
  getStats() {
    return api.get('/prices/stats');
  },

  /**
   * Get prices by store
   */
  getByStore(storeId) {
    return api.get(`/prices/by-store/${storeId}`);
  },

  /**
   * Get prices by item
   */
  getByItem(itemId) {
    return api.get(`/prices/by-item/${itemId}`);
  },

  /**
   * Create a new price (authenticated users)
   */
  create(data) {
    return api.post('/prices', data);
  },

  /**
   * Update a price (admin)
   */
  update(id, data) {
    return api.put(`/admin/prices/${id}`, data);
  },

  /**
   * Update user's own price
   */
  userUpdate(id, data) {
    return api.put(`/prices/${id}`, data);
  },

  /**
   * Delete a price (admin)
   */
  adminDelete(id) {
    return api.delete(`/admin/prices/${id}`);
  },

  /**
   * Delete user's own price
   */
  delete(id) {
    return api.delete(`/prices/${id}`);
  },

  /**
   * Verify a price (authenticated users)
   */
  verify(id, isAccurate) {
    return api.post(`/prices/${id}/verify`, { is_accurate: isAccurate });
  },
};

/**
 * Shopping Lists API
 */
const listsApi = {
  /**
   * Get all shopping lists for the current user
   * @param {Object} params - Optional params { status: 'active' | 'completed' }
   */
  getAll(params = {}) {
    const query = new URLSearchParams();
    if (params.status) query.set('status', params.status);
    const queryStr = query.toString();
    return api.get(`/lists${queryStr ? '?' + queryStr : ''}`);
  },

  /**
   * Get a single shopping list by ID with items
   */
  getById(id) {
    return api.get(`/lists/${id}`);
  },

  /**
   * Create a new shopping list
   */
  create(data) {
    return api.post('/lists', data);
  },

  /**
   * Update a shopping list
   */
  update(id, data) {
    return api.put(`/lists/${id}`, data);
  },

  /**
   * Delete a shopping list
   */
  delete(id) {
    return api.delete(`/lists/${id}`);
  },

  /**
   * Add an item to a shopping list
   */
  addItem(listId, itemId, quantity = 1) {
    return api.post(`/lists/${listId}/items`, { item_id: itemId, quantity });
  },

  /**
   * Update an item quantity in a shopping list
   */
  updateItem(listId, itemId, quantity) {
    return api.put(`/lists/${listId}/items/${itemId}`, { quantity });
  },

  /**
   * Remove an item from a shopping list
   */
  removeItem(listId, itemId) {
    return api.delete(`/lists/${listId}/items/${itemId}`);
  },

  /**
   * Build an optimized shopping plan for a list
   */
  buildPlan(listId, storeIds = null) {
    const data = {};
    if (storeIds && storeIds.length > 0) {
      data.store_ids = storeIds;
    }
    return api.post(`/lists/${listId}/build-plan`, data);
  },

  /**
   * Complete a shopping list with optional price confirmations
   * @param {number} listId - List ID
   * @param {Array} priceConfirmations - Optional array of { item_id, store_id, is_accurate, new_price }
   */
  complete(listId, priceConfirmations = null) {
    const data = {};
    if (priceConfirmations && priceConfirmations.length > 0) {
      data.price_confirmations = priceConfirmations;
    }
    return api.post(`/lists/${listId}/complete`, data);
  },

  /**
   * Reopen a completed shopping list
   */
  reopen(listId) {
    return api.post(`/lists/${listId}/reopen`, {});
  },

  /**
   * Duplicate a shopping list (create a copy)
   * @param {number} listId - Source list ID
   * @param {string} name - Name for the new list
   */
  duplicate(listId, name) {
    return api.post(`/lists/${listId}/duplicate`, { name });
  },
};

/**
 * Price Comparison API
 */
const compareApi = {
  /**
   * Get price comparison matrix
   * @param {number[]} storeIds - Array of store IDs to compare
   * @param {number[]} itemIds - Array of item IDs to compare (optional)
   */
  getComparison(storeIds, itemIds = null) {
    const query = new URLSearchParams();
    if (storeIds && storeIds.length > 0) {
      query.set('store_ids', storeIds.join(','));
    }
    if (itemIds && itemIds.length > 0) {
      query.set('item_ids', itemIds.join(','));
    }
    const queryStr = query.toString();
    return api.get(`/compare${queryStr ? '?' + queryStr : ''}`);
  },
};

/**
 * Feed API (placeholder for future implementation)
 */
const feedApi = {
  get(regionId = null, limit = 20, offset = 0) {
    let params = `?limit=${limit}&offset=${offset}`;
    if (regionId) params += `&region_id=${regionId}`;
    return api.get(`/feed${params}`);
  },

  getHotDeals(regionId = null) {
    const params = regionId ? `?region_id=${regionId}` : '';
    return api.get(`/feed/hot-deals${params}`);
  },

  getUserFeed(userId) {
    return api.get(`/feed/user/${userId}`);
  },
};

/**
 * Regions API
 */
const regionsApi = {
  /**
   * List regions with pagination and filters
   */
  list(params = {}) {
    const query = new URLSearchParams();
    if (params.limit) query.set('limit', params.limit);
    if (params.offset) query.set('offset', params.offset);
    if (params.search) query.set('search', params.search);
    if (params.state) query.set('state', params.state);
    const queryStr = query.toString();
    return api.get(`/regions${queryStr ? '?' + queryStr : ''}`);
  },

  /**
   * Get a single region by ID
   */
  getById(id) {
    return api.get(`/regions/${id}`);
  },

  /**
   * Get list of distinct states
   */
  getStates() {
    return api.get('/regions/states');
  },

  /**
   * Get region statistics
   */
  getStats() {
    return api.get('/regions/stats');
  },

  /**
   * Search regions
   */
  search(query, limit = 20) {
    return api.get(`/regions/search?q=${encodeURIComponent(query)}&limit=${limit}`);
  },

  /**
   * Create a new region (admin)
   */
  create(data) {
    return api.post('/admin/regions', data);
  },

  /**
   * Update a region (admin)
   */
  update(id, data) {
    return api.put(`/admin/regions/${id}`, data);
  },

  /**
   * Delete a region (admin)
   */
  delete(id) {
    return api.delete(`/admin/regions/${id}`);
  },
};

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    api,
    authApi,
    userApi,
    adminApi,
    storesApi,
    itemsApi,
    tagsApi,
    pricesApi,
    listsApi,
    compareApi,
    feedApi,
    regionsApi,
  };
}
