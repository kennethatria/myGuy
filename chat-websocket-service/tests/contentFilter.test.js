jest.mock('../src/utils/logger', () => ({
  debug: jest.fn(),
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn()
}));

const { filterContent, containsFilteredContent } = require('../src/utils/contentFilter');

describe('contentFilter', () => {
  describe('filterContent', () => {
    it('handles null input', () => {
      const result = filterContent(null);
      expect(result.filtered).toBe('');
      expect(result.removed).toEqual([]);
      expect(result.hasRemovedContent).toBe(false);
    });

    it('handles undefined input', () => {
      const result = filterContent(undefined);
      expect(result.filtered).toBe('');
      expect(result.hasRemovedContent).toBe(false);
    });

    it('handles non-string input (number)', () => {
      const result = filterContent(123);
      expect(result.filtered).toBe('');
      expect(result.hasRemovedContent).toBe(false);
    });

    it('passes through clean content unchanged', () => {
      const result = filterContent('Hello, how are you doing today?');
      expect(result.filtered).toBe('Hello, how are you doing today?');
      expect(result.removed).toEqual([]);
      expect(result.hasRemovedContent).toBe(false);
    });

    it('removes http URLs', () => {
      const result = filterContent('Check out http://example.com for details');
      expect(result.filtered).toContain('[link removed]');
      expect(result.removed.some(r => r.type === 'url')).toBe(true);
      expect(result.hasRemovedContent).toBe(true);
    });

    it('removes https URLs', () => {
      const result = filterContent('Visit https://secure.example.com/path');
      expect(result.filtered).toContain('[link removed]');
      expect(result.hasRemovedContent).toBe(true);
    });

    it('removes www URLs', () => {
      const result = filterContent('Go to www.example.com/items to learn more');
      expect(result.filtered).toContain('[link removed]');
      expect(result.hasRemovedContent).toBe(true);
    });

    it('removes email addresses', () => {
      const result = filterContent('Email me at john.doe@example.com anytime');
      expect(result.filtered).toContain('[email removed]');
      expect(result.removed.some(r => r.type === 'email')).toBe(true);
      expect(result.hasRemovedContent).toBe(true);
    });

    it('removes 10-digit phone numbers', () => {
      const result = filterContent('My number is 5551234567 call anytime');
      expect(result.filtered).toContain('[phone removed]');
      expect(result.removed.some(r => r.type === 'phone')).toBe(true);
      expect(result.hasRemovedContent).toBe(true);
    });

    it('removes dash-formatted phone numbers', () => {
      const result = filterContent('Call me at 555-867-5309 please');
      expect(result.filtered).toContain('[phone removed]');
      expect(result.hasRemovedContent).toBe(true);
    });

    it('does not remove short number sequences (prices, years)', () => {
      const result = filterContent('The price is 25 dollars and year 2024 applies');
      expect(result.hasRemovedContent).toBe(false);
    });

    it('removes multiple PII types from the same message', () => {
      const result = filterContent('Email john@test.com or call 5551234567 or see http://test.com');
      expect(result.removed.length).toBeGreaterThanOrEqual(2);
      expect(result.hasRemovedContent).toBe(true);
    });

    it('trims whitespace from filtered result', () => {
      const result = filterContent('  hello world  ');
      expect(result.filtered).toBe('hello world');
    });

    it('handles empty string (falsy)', () => {
      const result = filterContent('');
      expect(result.filtered).toBe('');
      expect(result.hasRemovedContent).toBe(false);
    });

    it('records url match value in removed array', () => {
      const result = filterContent('See https://example.com now');
      const urlRemoval = result.removed.find(r => r.type === 'url');
      expect(urlRemoval).toBeDefined();
      expect(urlRemoval.value).toContain('https://example.com');
    });

    it('records email match value in removed array', () => {
      const result = filterContent('Reach user@domain.org for info');
      const emailRemoval = result.removed.find(r => r.type === 'email');
      expect(emailRemoval).toBeDefined();
      expect(emailRemoval.value).toContain('@domain.org');
    });
  });

  describe('containsFilteredContent', () => {
    it('returns false for null', () => {
      expect(containsFilteredContent(null)).toBe(false);
    });

    it('returns false for undefined', () => {
      expect(containsFilteredContent(undefined)).toBe(false);
    });

    it('returns false for non-string value', () => {
      expect(containsFilteredContent(42)).toBe(false);
    });

    it('returns false for clean content', () => {
      expect(containsFilteredContent('Just a normal message here, no PII')).toBe(false);
    });

    it('returns true for content with http URL', () => {
      expect(containsFilteredContent('Visit http://malicious.com now')).toBe(true);
    });

    it('returns true for content with email', () => {
      expect(containsFilteredContent('reach me at contact@test.org')).toBe(true);
    });

    it('returns true for content with https URL', () => {
      expect(containsFilteredContent('Go to https://site.com/page')).toBe(true);
    });
  });
});
