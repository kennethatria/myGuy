const logger = require('./logger');

// Regex patterns for content filtering
const patterns = {
  // URLs with various protocols
  urls: /(?:https?|ftp|ftps):\/\/[^\s]+|www\.[^\s]+\.[^\s]+|[^\s]+\.[a-z]{2,}\/[^\s]*/gi,
  
  // Email addresses
  emails: /[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/g,
  
  // Phone numbers (various formats)
  phones: /(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}|[0-9]{10,15}/g,
  
  // Social media handles
  socialHandles: /@[a-zA-Z0-9_]+/g
};

/**
 * Filter content by removing URLs, emails, and phone numbers
 * @param {string} content - The message content to filter
 * @returns {object} - Object containing filtered content and what was removed
 */
const filterContent = (content) => {
  if (!content || typeof content !== 'string') {
    return {
      filtered: content || '',
      removed: [],
      hasRemovedContent: false
    };
  }

  const removed = [];
  let filtered = content;

  // Check and remove URLs
  const urlMatches = filtered.match(patterns.urls);
  if (urlMatches) {
    removed.push(...urlMatches.map(match => ({ type: 'url', value: match })));
    filtered = filtered.replace(patterns.urls, '[link removed]');
  }

  // Check and remove emails
  const emailMatches = filtered.match(patterns.emails);
  if (emailMatches) {
    removed.push(...emailMatches.map(match => ({ type: 'email', value: match })));
    filtered = filtered.replace(patterns.emails, '[email removed]');
  }

  // Check and remove phone numbers
  const phoneMatches = filtered.match(patterns.phones);
  if (phoneMatches) {
    // Filter out numbers that are likely not phone numbers (e.g., years, prices)
    const likelyPhones = phoneMatches.filter(match => {
      const digits = match.replace(/\D/g, '');
      return digits.length >= 10 && digits.length <= 15;
    });
    
    if (likelyPhones.length > 0) {
      removed.push(...likelyPhones.map(match => ({ type: 'phone', value: match })));
      likelyPhones.forEach(phone => {
        filtered = filtered.replace(phone, '[phone removed]');
      });
    }
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
const containsFilteredContent = (content) => {
  if (!content || typeof content !== 'string') return false;

  return patterns.urls.test(content) || 
         patterns.emails.test(content) || 
         patterns.phones.test(content);
};

module.exports = {
  filterContent,
  containsFilteredContent,
  patterns
};