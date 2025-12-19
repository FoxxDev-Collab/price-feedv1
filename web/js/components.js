/**
 * PRICE FEED - Component Library
 *
 * This file serves as documentation and additional component utilities.
 * Main component definitions are in script.js
 *
 * Usage Pattern (shadcn-inspired):
 *
 * 1. Import/use components via the global PriceFeed object:
 *    const { renderComponent } = window.PriceFeed;
 *
 * 2. Render a component:
 *    const buttonHtml = renderComponent('Button', { text: 'Click me', variant: 'primary' });
 *
 * 3. Mount to DOM:
 *    mount('#container', buttonHtml);
 */

// ============================================
// COMPONENT CATALOG
// Lists all available components with their props
// ============================================

const ComponentCatalog = {
  Button: {
    description: 'A versatile button component with multiple variants',
    props: {
      text: { type: 'string', required: true, description: 'Button text' },
      variant: { type: 'string', default: 'primary', options: ['primary', 'secondary', 'outline', 'ghost'] },
      size: { type: 'string', default: 'md', options: ['sm', 'md', 'lg'] },
      href: { type: 'string', description: 'If provided, renders as anchor' },
      shadow: { type: 'boolean', default: false, description: 'Add shadow effect' },
      className: { type: 'string', description: 'Additional CSS classes' },
      id: { type: 'string', description: 'Element ID' },
      type: { type: 'string', default: 'button', options: ['button', 'submit', 'reset'] },
    },
    examples: [
      { props: { text: 'Get Started', variant: 'primary', size: 'lg' } },
      { props: { text: 'Learn More', variant: 'outline' } },
      { props: { text: 'Sign In', variant: 'ghost' } },
    ]
  },

  Badge: {
    description: 'Small label for status, categories, or counts',
    props: {
      text: { type: 'string', required: true },
      variant: { type: 'string', default: 'primary', options: ['primary', 'secondary', 'accent', 'outline'] },
      size: { type: 'string', default: 'sm', options: ['sm', 'lg'] },
    },
    examples: [
      { props: { text: 'New', variant: 'primary' } },
      { props: { text: 'Hot Deal', variant: 'secondary' } },
      { props: { text: 'Verified', variant: 'accent' } },
    ]
  },

  Avatar: {
    description: 'User avatar with initials or image',
    props: {
      name: { type: 'string', required: true, description: 'Used to generate initials' },
      src: { type: 'string', description: 'Image URL (optional)' },
      size: { type: 'string', default: 'md', options: ['sm', 'md', 'lg', 'xl'] },
      variant: { type: 'string', default: 'primary', options: ['primary', 'secondary', 'accent'] },
    },
    examples: [
      { props: { name: 'John Doe', size: 'lg' } },
      { props: { name: 'Sarah M.', variant: 'secondary' } },
    ]
  },

  IconBox: {
    description: 'Container for icons with background',
    props: {
      icon: { type: 'string', required: true, description: 'SVG icon HTML' },
      size: { type: 'string', default: 'lg', options: ['sm', 'md', 'lg'] },
      variant: { type: 'string', default: 'primary', options: ['primary', 'secondary', 'accent', 'white'] },
    }
  },

  FeedItem: {
    description: 'Price feed activity item',
    props: {
      userName: { type: 'string', required: true },
      time: { type: 'string', required: true, description: 'Relative time string' },
      text: { type: 'string', required: true, description: 'Activity description' },
      price: { type: 'string', required: true },
      priceLabel: { type: 'string', description: 'Product description' },
      variant: { type: 'string', default: 'primary', options: ['primary', 'secondary', 'accent'] },
      badge: { type: 'string', description: 'Optional verification badge' },
      oldPrice: { type: 'string', description: 'Strikethrough old price' },
    }
  },

  Stat: {
    description: 'Statistics display component',
    props: {
      value: { type: 'string', required: true },
      label: { type: 'string', required: true },
      variant: { type: 'string', default: 'primary', options: ['primary', 'secondary', 'accent'] },
    }
  },

  FeatureCard: {
    description: 'Feature showcase card with icon',
    props: {
      icon: { type: 'string', required: true, description: 'SVG icon HTML' },
      iconVariant: { type: 'string', default: 'primary' },
      title: { type: 'string', required: true },
      description: { type: 'string', required: true },
    }
  },

  SectionHeader: {
    description: 'Section title and subtitle',
    props: {
      title: { type: 'string', required: true },
      subtitle: { type: 'string' },
    }
  },

  Modal: {
    description: 'Modal dialog component',
    props: {
      id: { type: 'string', required: true },
      title: { type: 'string', required: true },
      body: { type: 'string', required: true, description: 'HTML content' },
      showFooter: { type: 'boolean', default: true },
      confirmText: { type: 'string', default: 'Confirm' },
      cancelText: { type: 'string', default: 'Cancel' },
    }
  },

  FormInput: {
    description: 'Form input with label and error state',
    props: {
      id: { type: 'string', required: true },
      name: { type: 'string' },
      label: { type: 'string' },
      type: { type: 'string', default: 'text' },
      placeholder: { type: 'string' },
      required: { type: 'boolean', default: false },
      error: { type: 'string', description: 'Error message' },
      value: { type: 'string' },
    }
  },

  Spinner: {
    description: 'Loading spinner',
    props: {
      size: { type: 'string', default: 'md', options: ['sm', 'md', 'lg'] },
    }
  },

  EmptyState: {
    description: 'Empty/no-data state display',
    props: {
      icon: { type: 'string', description: 'SVG icon HTML' },
      title: { type: 'string', required: true },
      description: { type: 'string' },
      actionText: { type: 'string' },
      actionHref: { type: 'string' },
    }
  },

  Navbar: {
    description: 'Main navigation bar',
    props: {
      brandName: { type: 'string', default: 'Price Feed' },
      links: { type: 'array', description: 'Array of {href, text}' },
      showAuth: { type: 'boolean', default: true },
    }
  },

  Footer: {
    description: 'Page footer with columns',
    props: {
      brandName: { type: 'string', default: 'Price Feed' },
      tagline: { type: 'string' },
      columns: { type: 'array', description: 'Array of {title, links: [{href, text}]}' },
      copyright: { type: 'string' },
    }
  },
};

// ============================================
// ADDITIONAL COMPONENT HELPERS
// ============================================

/**
 * Create a list of components
 * @param {string} componentName - Name of component to render
 * @param {Array} items - Array of props objects
 * @param {string} separator - HTML between items
 * @returns {string} Combined HTML
 */
function renderList(componentName, items, separator = '') {
  const { renderComponent } = window.PriceFeed;
  return items.map(props => renderComponent(componentName, props)).join(separator);
}

/**
 * Conditional rendering helper
 * @param {boolean} condition - Condition to check
 * @param {string} trueHtml - HTML if true
 * @param {string} falseHtml - HTML if false (optional)
 * @returns {string} Selected HTML
 */
function renderIf(condition, trueHtml, falseHtml = '') {
  return condition ? trueHtml : falseHtml;
}

/**
 * Create a grid of components
 * @param {string} componentName - Name of component
 * @param {Array} items - Array of props
 * @param {Object} gridOptions - Grid configuration
 * @returns {string} Grid HTML
 */
function renderGrid(componentName, items, gridOptions = {}) {
  const { cols = 3, gap = 8, className = '' } = gridOptions;
  const colClass = `grid-cols-1 md:grid-cols-${cols}`;
  const gapClass = `gap-${gap}`;

  const itemsHtml = renderList(componentName, items);

  return `<div class="grid ${colClass} ${gapClass} ${className}">${itemsHtml}</div>`;
}

// ============================================
// PAGE TEMPLATE HELPERS
// ============================================

/**
 * Create a full page with navbar and footer
 * @param {Object} options - Page configuration
 * @returns {string} Full page HTML
 */
function createPage(options = {}) {
  const {
    title = 'Price Feed',
    bodyContent = '',
    navLinks = [
      { href: '#features', text: 'Features' },
      { href: '#how-it-works', text: 'How It Works' },
      { href: '#community', text: 'Community' },
    ],
    footerColumns = [
      {
        title: 'Product',
        links: [
          { href: '#', text: 'Features' },
          { href: '#', text: 'Pricing' },
        ]
      },
      {
        title: 'Company',
        links: [
          { href: '#', text: 'About' },
          { href: '#', text: 'Contact' },
        ]
      },
      {
        title: 'Legal',
        links: [
          { href: '#', text: 'Privacy' },
          { href: '#', text: 'Terms' },
        ]
      },
    ],
  } = options;

  const { renderComponent } = window.PriceFeed;

  const navbar = renderComponent('Navbar', { links: navLinks });
  const footer = renderComponent('Footer', {
    columns: footerColumns,
    copyright: '2025 Foxx Cyber LLC. All rights reserved.',
  });

  return `
    ${navbar}
    <main>
      ${bodyContent}
    </main>
    ${footer}
  `;
}

// ============================================
// INTERACTIVE COMPONENT BEHAVIORS
// ============================================

/**
 * Initialize tag input with autocomplete
 * @param {string} containerId - Container element ID
 * @param {Function} fetchSuggestions - Async function to fetch suggestions
 */
function initTagInput(containerId, fetchSuggestions) {
  const container = document.getElementById(containerId);
  if (!container) return;

  const input = container.querySelector('.tag-input');
  const chipsContainer = container.querySelector('.tag-chips');
  const suggestionsContainer = container.querySelector('.tag-suggestions');
  const hiddenInput = container.querySelector('input[type="hidden"]');

  let selectedTags = [];
  let debounceTimer;

  // Handle input
  input.addEventListener('input', (e) => {
    clearTimeout(debounceTimer);
    const query = e.target.value.trim();

    if (query.length < 2) {
      suggestionsContainer.classList.add('hidden');
      return;
    }

    debounceTimer = setTimeout(async () => {
      const suggestions = await fetchSuggestions(query);
      renderSuggestions(suggestions);
    }, 300);
  });

  // Handle suggestion click
  suggestionsContainer.addEventListener('click', (e) => {
    const item = e.target.closest('.tag-suggestion-item');
    if (item) {
      addTag(item.dataset.value);
    }
  });

  // Handle Enter key
  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      const value = input.value.trim();
      if (value) {
        addTag(value);
      }
    }
  });

  function addTag(tag) {
    if (!selectedTags.includes(tag)) {
      selectedTags.push(tag);
      renderChips();
      updateHiddenInput();
    }
    input.value = '';
    suggestionsContainer.classList.add('hidden');
  }

  function removeTag(tag) {
    selectedTags = selectedTags.filter(t => t !== tag);
    renderChips();
    updateHiddenInput();
  }

  function renderChips() {
    const { escapeHtml } = window.PriceFeed;
    chipsContainer.innerHTML = selectedTags.map(tag => `
      <span class="tag-chip">
        ${escapeHtml(tag)}
        <button type="button" class="tag-chip-remove" data-tag="${escapeHtml(tag)}">&times;</button>
      </span>
    `).join('');

    // Bind remove handlers
    chipsContainer.querySelectorAll('.tag-chip-remove').forEach(btn => {
      btn.addEventListener('click', () => removeTag(btn.dataset.tag));
    });
  }

  function renderSuggestions(suggestions) {
    if (!suggestions.length) {
      suggestionsContainer.classList.add('hidden');
      return;
    }

    const { escapeHtml } = window.PriceFeed;
    suggestionsContainer.innerHTML = suggestions.map(s => `
      <div class="tag-suggestion-item" data-value="${escapeHtml(s.name)}">
        ${escapeHtml(s.name)}
        ${s.count ? `<span class="tag-suggestion-count">(${s.count})</span>` : ''}
      </div>
    `).join('');
    suggestionsContainer.classList.remove('hidden');
  }

  function updateHiddenInput() {
    if (hiddenInput) {
      hiddenInput.value = JSON.stringify(selectedTags);
    }
  }

  // Return API for external control
  return {
    getTags: () => [...selectedTags],
    setTags: (tags) => {
      selectedTags = [...tags];
      renderChips();
      updateHiddenInput();
    },
    addTag,
    removeTag,
    clear: () => {
      selectedTags = [];
      renderChips();
      updateHiddenInput();
    }
  };
}

/**
 * Initialize infinite scroll for a list
 * @param {string} containerId - Container element ID
 * @param {Function} fetchMore - Async function to fetch more items
 * @param {Function} renderItem - Function to render each item
 */
function initInfiniteScroll(containerId, fetchMore, renderItem) {
  const container = document.getElementById(containerId);
  if (!container) return;

  let loading = false;
  let page = 1;
  let hasMore = true;

  const sentinel = document.createElement('div');
  sentinel.className = 'infinite-scroll-sentinel';
  container.appendChild(sentinel);

  const observer = new IntersectionObserver(async (entries) => {
    if (entries[0].isIntersecting && !loading && hasMore) {
      loading = true;

      // Show loading indicator
      const loader = document.createElement('div');
      loader.className = 'infinite-scroll-loader';
      loader.innerHTML = window.PriceFeed.renderComponent('Spinner', {});
      container.insertBefore(loader, sentinel);

      try {
        const items = await fetchMore(page);
        page++;

        if (items.length === 0) {
          hasMore = false;
        } else {
          const fragment = document.createDocumentFragment();
          items.forEach(item => {
            const div = document.createElement('div');
            div.innerHTML = renderItem(item);
            fragment.appendChild(div.firstChild);
          });
          container.insertBefore(fragment, sentinel);
        }
      } catch (error) {
        console.error('Failed to fetch more items:', error);
      }

      // Remove loader
      loader.remove();
      loading = false;
    }
  }, { rootMargin: '100px' });

  observer.observe(sentinel);

  return {
    reset: () => {
      page = 1;
      hasMore = true;
      // Clear existing items (except sentinel)
      Array.from(container.children).forEach(child => {
        if (child !== sentinel) child.remove();
      });
    },
    destroy: () => {
      observer.disconnect();
      sentinel.remove();
    }
  };
}

// ============================================
// EXPORTS
// ============================================

window.PriceFeed = {
  ...window.PriceFeed,

  // Component catalog
  ComponentCatalog,

  // Helpers
  renderList,
  renderIf,
  renderGrid,
  createPage,

  // Interactive behaviors
  initTagInput,
  initInfiniteScroll,
};
