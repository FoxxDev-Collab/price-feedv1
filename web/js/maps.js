/**
 * Price Feed Google Maps API Module
 * Provides Google Maps integration for address autocomplete, interactive maps,
 * and nearby store discovery.
 *
 * Dependencies: api.js must be loaded before this module
 */

const mapsApi = {
  // State
  apiKey: null,
  mapInstance: null,
  autocomplete: null,
  isLoaded: false,
  loadPromise: null,

  // Default map center: Colorado Springs, CO
  defaultCenter: { lat: 38.8339, lng: -104.8214 },
  defaultZoom: 13,

  // Custom map styling for a subtle, modern look
  mapStyles: [
    {
      featureType: 'poi',
      elementType: 'labels',
      stylers: [{ visibility: 'off' }]
    },
    {
      featureType: 'poi.business',
      stylers: [{ visibility: 'off' }]
    },
    {
      featureType: 'transit',
      elementType: 'labels.icon',
      stylers: [{ visibility: 'off' }]
    },
    {
      featureType: 'road',
      elementType: 'labels.icon',
      stylers: [{ visibility: 'off' }]
    },
    {
      featureType: 'water',
      elementType: 'geometry',
      stylers: [{ color: '#e9e9e9' }, { lightness: 17 }]
    },
    {
      featureType: 'landscape',
      elementType: 'geometry',
      stylers: [{ color: '#f5f5f5' }, { lightness: 20 }]
    },
    {
      featureType: 'road.highway',
      elementType: 'geometry.fill',
      stylers: [{ color: '#ffffff' }, { lightness: 17 }]
    },
    {
      featureType: 'road.highway',
      elementType: 'geometry.stroke',
      stylers: [{ color: '#ffffff' }, { lightness: 29 }, { weight: 0.2 }]
    },
    {
      featureType: 'road.arterial',
      elementType: 'geometry',
      stylers: [{ color: '#ffffff' }, { lightness: 18 }]
    },
    {
      featureType: 'road.local',
      elementType: 'geometry',
      stylers: [{ color: '#ffffff' }, { lightness: 16 }]
    },
    {
      featureType: 'administrative',
      elementType: 'geometry.stroke',
      stylers: [{ color: '#c9c9c9' }, { lightness: 14 }, { weight: 1.4 }]
    }
  ],

  /**
   * Initialize the Maps API
   * Fetches API key from backend and loads Google Maps script
   * @returns {Promise<boolean>} Resolves to true when maps is ready
   */
  async init() {
    // Return existing promise if already initializing
    if (this.loadPromise) {
      return this.loadPromise;
    }

    // Return immediately if already loaded
    if (this.isLoaded) {
      return true;
    }

    this.loadPromise = this._doInit();
    return this.loadPromise;
  },

  /**
   * Internal initialization logic
   * @private
   */
  async _doInit() {
    try {
      // 1. Fetch API key from backend
      const response = await api.get('/maps/config');
      // Handle both wrapped ({data: {frontend_key}}) and unwrapped ({frontend_key}) responses
      const config = response?.data || response;
      if (!config || !config.frontend_key) {
        throw new Error('Failed to retrieve Maps API configuration - API key not configured');
      }
      this.apiKey = config.frontend_key;

      // 2. Load Google Maps script
      await this.loadScript();

      this.isLoaded = true;
      return true;
    } catch (error) {
      console.error('Maps API initialization failed:', error);
      this.loadPromise = null;
      throw error;
    }
  },

  /**
   * Load the Google Maps JavaScript API dynamically
   * @returns {Promise<void>} Resolves when the script is loaded
   */
  loadScript() {
    return new Promise((resolve, reject) => {
      // Check if already loaded
      if (window.google && window.google.maps) {
        resolve();
        return;
      }

      // Create callback function
      const callbackName = 'mapsApiReady';
      window[callbackName] = () => {
        delete window[callbackName];
        resolve();
      };

      // Create script element
      const script = document.createElement('script');
      script.src = `https://maps.googleapis.com/maps/api/js?key=${this.apiKey}&libraries=places&callback=${callbackName}`;
      script.async = true;
      script.defer = true;

      script.onerror = () => {
        delete window[callbackName];
        reject(new Error('Failed to load Google Maps script'));
      };

      document.head.appendChild(script);
    });
  },

  /**
   * Create a map instance in a container element
   * @param {string} elementId - The ID of the container element
   * @param {Object} options - Map options
   * @param {Object} options.center - Center coordinates { lat, lng }
   * @param {number} options.zoom - Zoom level
   * @param {Array} options.styles - Custom map styles
   * @param {boolean} options.disableDefaultUI - Disable default controls
   * @returns {google.maps.Map} The map instance
   */
  createMap(elementId, options = {}) {
    if (!this.isLoaded) {
      throw new Error('Maps API not initialized. Call init() first.');
    }

    const element = document.getElementById(elementId);
    if (!element) {
      throw new Error(`Element with ID "${elementId}" not found`);
    }

    const mapOptions = {
      center: options.center || this.defaultCenter,
      zoom: options.zoom !== undefined ? options.zoom : this.defaultZoom,
      styles: options.styles || this.mapStyles,
      disableDefaultUI: options.disableDefaultUI || false,
      zoomControl: true,
      mapTypeControl: false,
      streetViewControl: false,
      fullscreenControl: true,
      ...options
    };

    this.mapInstance = new google.maps.Map(element, mapOptions);
    return this.mapInstance;
  },

  /**
   * Setup Google Places Autocomplete on an input element
   * @param {string} inputId - The ID of the input element
   * @param {Object} options - Autocomplete options
   * @param {Array} options.types - Types of predictions (e.g., ['address'])
   * @param {Object} options.componentRestrictions - Country restrictions
   * @param {Array} options.fields - Fields to include in place details
   * @returns {google.maps.places.Autocomplete} The autocomplete instance
   */
  setupAutocomplete(inputId, options = {}) {
    if (!this.isLoaded) {
      throw new Error('Maps API not initialized. Call init() first.');
    }

    const input = document.getElementById(inputId);
    if (!input) {
      throw new Error(`Input element with ID "${inputId}" not found`);
    }

    const autocompleteOptions = {
      types: options.types || ['address'],
      componentRestrictions: options.componentRestrictions || { country: 'us' },
      fields: options.fields || [
        'address_components',
        'formatted_address',
        'geometry',
        'place_id',
        'name'
      ],
      ...options
    };

    this.autocomplete = new google.maps.places.Autocomplete(input, autocompleteOptions);
    return this.autocomplete;
  },

  /**
   * Parse address components from a Google Place result
   * @param {Array} components - The address_components array from a Place result
   * @returns {Object} Parsed address { street_address, city, state, zip_code, country }
   */
  parseAddressComponents(components) {
    if (!components || !Array.isArray(components)) {
      return {
        street_address: '',
        city: '',
        state: '',
        zip_code: '',
        country: ''
      };
    }

    const result = {
      street_number: '',
      route: '',
      city: '',
      state: '',
      zip_code: '',
      country: ''
    };

    for (const component of components) {
      const types = component.types;

      if (types.includes('street_number')) {
        result.street_number = component.long_name;
      } else if (types.includes('route')) {
        result.route = component.long_name;
      } else if (types.includes('locality')) {
        result.city = component.long_name;
      } else if (types.includes('administrative_area_level_1')) {
        result.state = component.short_name;
      } else if (types.includes('postal_code')) {
        result.zip_code = component.long_name;
      } else if (types.includes('country')) {
        result.country = component.short_name;
      }
    }

    // Combine street number and route for full street address
    const street_address = [result.street_number, result.route]
      .filter(Boolean)
      .join(' ');

    return {
      street_address,
      city: result.city,
      state: result.state,
      zip_code: result.zip_code,
      country: result.country
    };
  },

  /**
   * Get user's current location via browser geolocation
   * @returns {Promise<{lat: number, lng: number}>} User's coordinates
   */
  getCurrentLocation() {
    return new Promise((resolve, reject) => {
      if (!navigator.geolocation) {
        reject(new Error('Geolocation is not supported by this browser'));
        return;
      }

      navigator.geolocation.getCurrentPosition(
        (position) => {
          resolve({
            lat: position.coords.latitude,
            lng: position.coords.longitude
          });
        },
        (error) => {
          let message;
          switch (error.code) {
            case error.PERMISSION_DENIED:
              message = 'Location access was denied. Please enable location permissions.';
              break;
            case error.POSITION_UNAVAILABLE:
              message = 'Location information is unavailable.';
              break;
            case error.TIMEOUT:
              message = 'Location request timed out.';
              break;
            default:
              message = 'An unknown error occurred while getting location.';
          }
          reject(new Error(message));
        },
        {
          enableHighAccuracy: true,
          timeout: 10000,
          maximumAge: 300000 // 5 minutes cache
        }
      );
    });
  },

  // ==========================================
  // Backend API Calls (proxied through server)
  // ==========================================

  /**
   * Geocode an address to coordinates
   * @param {string} address - The address to geocode
   * @returns {Promise<Object>} Geocoding result with lat/lng
   */
  async geocode(address) {
    return api.post('/maps/geocode', { address });
  },

  /**
   * Reverse geocode coordinates to an address (uses backend - requires auth)
   * @param {number} lat - Latitude
   * @param {number} lng - Longitude
   * @returns {Promise<Object>} Address information
   */
  async reverseGeocode(lat, lng) {
    return api.post('/maps/reverse-geocode', { latitude: lat, longitude: lng });
  },

  /**
   * Reverse geocode coordinates to an address (client-side - no auth required)
   * Use this for registration/public pages
   * @param {number} lat - Latitude
   * @param {number} lng - Longitude
   * @returns {Promise<Object>} Address information
   */
  async reverseGeocodeClient(lat, lng) {
    if (!this.isLoaded) {
      throw new Error('Maps API not initialized. Call init() first.');
    }

    return new Promise((resolve, reject) => {
      const geocoder = new google.maps.Geocoder();
      const latlng = { lat, lng };

      geocoder.geocode({ location: latlng }, (results, status) => {
        if (status === 'OK' && results[0]) {
          const place = results[0];
          resolve({
            formatted_address: place.formatted_address,
            place_id: place.place_id,
            address_components: place.address_components,
            components: this.parseAddressComponents(place.address_components)
          });
        } else {
          reject(new Error('Reverse geocoding failed: ' + status));
        }
      });
    });
  },

  /**
   * Find nearby stores
   * @param {number} lat - Latitude
   * @param {number} lng - Longitude
   * @param {number} radius - Search radius in meters (default: 5000)
   * @returns {Promise<Object>} List of nearby stores
   */
  async findNearbyStores(lat, lng, radius = 5000) {
    return api.post('/maps/nearby-stores', { latitude: lat, longitude: lng, radius });
  },

  /**
   * Get place details by place ID
   * @param {string} placeId - Google Place ID
   * @returns {Promise<Object>} Place details
   */
  async getPlaceDetails(placeId) {
    return api.get(`/maps/place/${encodeURIComponent(placeId)}`);
  },

  // ==========================================
  // Map Helpers
  // ==========================================

  /**
   * Add a marker to a map
   * @param {google.maps.Map} map - The map instance
   * @param {Object} position - Marker position { lat, lng }
   * @param {Object} options - Marker options
   * @param {string} options.title - Marker title (tooltip)
   * @param {Object|string} options.icon - Custom icon
   * @param {string} options.animation - Animation type ('DROP' or 'BOUNCE')
   * @param {boolean} options.draggable - Whether marker is draggable
   * @returns {google.maps.Marker} The marker instance
   */
  addMarker(map, position, options = {}) {
    if (!this.isLoaded) {
      throw new Error('Maps API not initialized. Call init() first.');
    }

    const markerOptions = {
      position,
      map,
      title: options.title || '',
      draggable: options.draggable || false
    };

    // Handle animation
    if (options.animation) {
      if (options.animation === 'DROP') {
        markerOptions.animation = google.maps.Animation.DROP;
      } else if (options.animation === 'BOUNCE') {
        markerOptions.animation = google.maps.Animation.BOUNCE;
      }
    }

    // Handle custom icon
    if (options.icon) {
      markerOptions.icon = options.icon;
    }

    return new google.maps.Marker(markerOptions);
  },

  /**
   * Create a custom styled icon
   * @param {string} type - Icon type: 'user', 'store', 'store-selected'
   * @returns {Object} Google Maps icon object
   */
  createIcon(type) {
    if (!this.isLoaded) {
      throw new Error('Maps API not initialized. Call init() first.');
    }

    const icons = {
      user: {
        path: google.maps.SymbolPath.CIRCLE,
        fillColor: '#3b82f6', // Blue
        fillOpacity: 1,
        strokeColor: '#ffffff',
        strokeWeight: 3,
        scale: 10
      },
      store: {
        path: google.maps.SymbolPath.BACKWARD_CLOSED_ARROW,
        fillColor: '#16a34a', // Green (primary color)
        fillOpacity: 0.9,
        strokeColor: '#ffffff',
        strokeWeight: 2,
        scale: 6
      },
      'store-selected': {
        path: google.maps.SymbolPath.BACKWARD_CLOSED_ARROW,
        fillColor: '#f59e0b', // Amber (highlight)
        fillOpacity: 1,
        strokeColor: '#ffffff',
        strokeWeight: 2,
        scale: 8
      }
    };

    return icons[type] || icons.store;
  },

  /**
   * Fit map bounds to show all positions
   * @param {google.maps.Map} map - The map instance
   * @param {Array<{lat: number, lng: number}>} positions - Array of positions
   * @param {Object} options - Options
   * @param {number} options.padding - Padding in pixels (default: 50)
   * @param {number} options.maxZoom - Maximum zoom level (default: 16)
   */
  fitBounds(map, positions, options = {}) {
    if (!this.isLoaded) {
      throw new Error('Maps API not initialized. Call init() first.');
    }

    if (!positions || positions.length === 0) {
      return;
    }

    // Single position - just center on it
    if (positions.length === 1) {
      map.setCenter(positions[0]);
      map.setZoom(options.maxZoom || 16);
      return;
    }

    // Multiple positions - create bounds
    const bounds = new google.maps.LatLngBounds();
    for (const position of positions) {
      bounds.extend(position);
    }

    const padding = options.padding !== undefined ? options.padding : 50;
    map.fitBounds(bounds, padding);

    // Limit max zoom after fitting
    if (options.maxZoom) {
      google.maps.event.addListenerOnce(map, 'bounds_changed', () => {
        if (map.getZoom() > options.maxZoom) {
          map.setZoom(options.maxZoom);
        }
      });
    }
  },

  /**
   * Create an info window for a marker
   * @param {Object} options - Info window options
   * @param {string} options.content - HTML content
   * @param {number} options.maxWidth - Maximum width in pixels
   * @returns {google.maps.InfoWindow} The info window instance
   */
  createInfoWindow(options = {}) {
    if (!this.isLoaded) {
      throw new Error('Maps API not initialized. Call init() first.');
    }

    return new google.maps.InfoWindow({
      content: options.content || '',
      maxWidth: options.maxWidth || 300
    });
  },

  /**
   * Clear all markers from a map
   * @param {Array<google.maps.Marker>} markers - Array of marker instances
   */
  clearMarkers(markers) {
    if (!markers || !Array.isArray(markers)) {
      return;
    }

    for (const marker of markers) {
      if (marker && typeof marker.setMap === 'function') {
        marker.setMap(null);
      }
    }
  },

  /**
   * Calculate distance between two points (in meters)
   * @param {Object} point1 - First point { lat, lng }
   * @param {Object} point2 - Second point { lat, lng }
   * @returns {number} Distance in meters
   */
  calculateDistance(point1, point2) {
    if (!this.isLoaded) {
      throw new Error('Maps API not initialized. Call init() first.');
    }

    const from = new google.maps.LatLng(point1.lat, point1.lng);
    const to = new google.maps.LatLng(point2.lat, point2.lng);

    return google.maps.geometry.spherical.computeDistanceBetween(from, to);
  },

  /**
   * Format distance for display
   * @param {number} meters - Distance in meters
   * @returns {string} Formatted distance (e.g., "0.5 mi" or "2.3 mi")
   */
  formatDistance(meters) {
    const miles = meters / 1609.344;
    if (miles < 0.1) {
      return `${Math.round(meters)} ft`;
    }
    return `${miles.toFixed(1)} mi`;
  },

  /**
   * Check if Maps API is ready
   * @returns {boolean} True if initialized and loaded
   */
  isReady() {
    return this.isLoaded && window.google && window.google.maps;
  },

  /**
   * Get the current map instance
   * @returns {google.maps.Map|null} The current map instance or null
   */
  getMap() {
    return this.mapInstance;
  },

  /**
   * Get the current autocomplete instance
   * @returns {google.maps.places.Autocomplete|null} The autocomplete instance or null
   */
  getAutocomplete() {
    return this.autocomplete;
  }
};

// Make globally available
window.mapsApi = mapsApi;

// Export for use in modules
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { mapsApi };
}
