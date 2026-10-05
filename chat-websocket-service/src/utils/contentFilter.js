const logger = require('./logger');

// Contact-detail patterns. The backend checks gig text with the same
// patterns (backend/internal/contacts); both are tested against
// shared/contact-filter-cases.json so they agree. Order matters when
// masking: emails before links, so "john@gmail.com" isn't half a link.
//
// The lookbehinds only let a match start at the beginning of a run of
// address characters. Without them a long word is retried from every
// character, which is quadratic (ReDoS). Go's RE2 is linear anyway, and the
// lookbehinds don't change what matches.
const patterns = {
  emails: /(?<![a-z0-9._%+-])[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}/gi,

  // With a scheme, starting www., or a bare domain on a common TLD
  urls: /(?:https?|ftp):\/\/\S+|(?<!\S)www\.\S+|(?<![a-z0-9.-])[a-z0-9-]+(?:\.[a-z0-9-]+)*\.(?:com|net|org|ug|co|io|me|info|biz|app|dev|xyz|link|ly|africa)\b(?:\/\S*)?/gi,

  // 9 to 15 digits, optionally +, spaced by space - . ( ) — local (0772 123 456)
  // and international (+256 772 123456) formats, but not prices like 120,000
  phones: /\+?\(?\d(?:[\s\-.()]*\d){8,14}/g,

  // @handle, not the @ inside an email (emails are masked first anyway)
  socialHandles: /(?<![A-Za-z0-9_])@[A-Za-z0-9_][A-Za-z0-9_.]+/g
};

const replacements = [
  ['email', patterns.emails, '[email removed]'],
  ['url', patterns.urls, '[link removed]'],
  ['phone', patterns.phones, '[phone removed]'],
  ['handle', patterns.socialHandles, '[handle removed]']
];

/**
 * Mask contact details: URLs, emails, phone numbers and social handles.
 * @param {string} content - The message content to filter
 * @returns {object} - Filtered content and what was removed
 */
const filterContent = (content) => {
  if (!content || typeof content !== 'string') {
    return {
      filtered: '',
      removed: [],
      hasRemovedContent: false
    };
  }

  const removed = [];
  let filtered = content;

  for (const [type, pattern, label] of replacements) {
    filtered = filtered.replace(pattern, (match) => {
      removed.push({ type, value: match });
      return label;
    });
  }

  logger.debug('Content filtered', {
    originalLength: content.length,
    filteredLength: filtered.length,
    removedCount: removed.length
  });

  return {
    filtered: filtered.trim(),
    removed,
    hasRemovedContent: removed.length > 0
  };
};

/**
 * Check if content contains any filtered patterns
 * @param {string} content - The content to check
 * @returns {boolean} - True if content contains filtered patterns
 */
const containsFilteredContent = (content) => filterContent(content).hasRemovedContent;

module.exports = {
  filterContent,
  containsFilteredContent,
  patterns
};
