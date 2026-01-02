const axios = require('axios');
const logger = require('../utils/logger');

/**
 * ValidationService
 *
 * Handles cross-service validation for references to entities in other databases.
 * Since chat service has separate database (my_guy_chat), we validate references
 * to users, tasks, and store items via API calls to their respective services.
 */
class ValidationService {
    constructor() {
        this.mainApiUrl = process.env.MAIN_API_URL || 'http://api:8080/api/v1';
        this.storeApiUrl = process.env.STORE_API_URL || 'http://store-service:8081/api/v1';

        // Cache for validation results (short TTL to reduce API calls)
        this.cache = new Map();
        this.cacheTTL = 60000; // 1 minute

        logger.info('ValidationService initialized', {
            mainApiUrl: this.mainApiUrl,
            storeApiUrl: this.storeApiUrl
        });
    }

    /**
     * Get cached validation result if available and not expired
     */
    _getCached(key) {
        const cached = this.cache.get(key);
        if (cached && Date.now() - cached.timestamp < this.cacheTTL) {
            return cached.value;
        }
        this.cache.delete(key);
        return null;
    }

    /**
     * Store validation result in cache
     */
    _setCache(key, value) {
        this.cache.set(key, {
            value,
            timestamp: Date.now()
        });
    }

    /**
     * Validate if a user exists
     * @param {number} userId - User ID to validate
     * @param {string} token - JWT token for authentication
     * @returns {Promise<boolean>} - True if user exists
     */
    async validateUser(userId, token) {
        const cacheKey = `user:${userId}`;
        const cached = this._getCached(cacheKey);
        if (cached !== null) {
            return cached;
        }

        try {
            const response = await axios.get(
                `${this.mainApiUrl}/users/${userId}`,
                {
                    headers: { Authorization: `Bearer ${token}` },
                    timeout: 5000
                }
            );

            const exists = response.status === 200;
            this._setCache(cacheKey, exists);

            logger.debug('User validation', { userId, exists });
            return exists;
        } catch (error) {
            if (error.response && error.response.status === 404) {
                this._setCache(cacheKey, false);
                logger.debug('User not found', { userId });
                return false;
            }

            logger.warn('User validation failed', {
                userId,
                error: error.message,
                status: error.response?.status
            });

            // On error, assume it exists to avoid blocking (fail open)
            // This allows service to continue if Main API is temporarily down
            return true;
        }
    }

    /**
     * Validate if a task exists
     * @param {number} taskId - Task ID to validate
     * @param {string} token - JWT token for authentication
     * @returns {Promise<boolean>} - True if task exists
     */
    async validateTask(taskId, token) {
        const cacheKey = `task:${taskId}`;
        const cached = this._getCached(cacheKey);
        if (cached !== null) {
            return cached;
        }

        try {
            const response = await axios.get(
                `${this.mainApiUrl}/tasks/${taskId}`,
                {
                    headers: { Authorization: `Bearer ${token}` },
                    timeout: 5000
                }
            );

            const exists = response.status === 200;
            this._setCache(cacheKey, exists);

            logger.debug('Task validation', { taskId, exists });
            return exists;
        } catch (error) {
            if (error.response && error.response.status === 404) {
                this._setCache(cacheKey, false);
                logger.debug('Task not found', { taskId });
                return false;
            }

            logger.warn('Task validation failed', {
                taskId,
                error: error.message,
                status: error.response?.status
            });

            // Fail open on errors
            return true;
        }
    }

    /**
     * Validate if an application exists
     * @param {number} applicationId - Application ID to validate
     * @param {string} token - JWT token for authentication
     * @returns {Promise<boolean>} - True if application exists
     */
    async validateApplication(applicationId, token) {
        const cacheKey = `application:${applicationId}`;
        const cached = this._getCached(cacheKey);
        if (cached !== null) {
            return cached;
        }

        try {
            // Applications are accessed via tasks endpoint
            // We'll need to query differently or skip this validation
            // For now, return true (applications are managed by main API)
            logger.debug('Application validation (skipped)', { applicationId });
            return true;
        } catch (error) {
            logger.warn('Application validation failed', {
                applicationId,
                error: error.message
            });
            return true;
        }
    }

    /**
     * Validate if a store item exists
     * @param {number} itemId - Store item ID to validate
     * @param {string} token - JWT token for authentication
     * @returns {Promise<boolean>} - True if store item exists
     */
    async validateStoreItem(itemId, token) {
        const cacheKey = `item:${itemId}`;
        const cached = this._getCached(cacheKey);
        if (cached !== null) {
            return cached;
        }

        try {
            const response = await axios.get(
                `${this.storeApiUrl}/items/${itemId}`,
                {
                    headers: { Authorization: `Bearer ${token}` },
                    timeout: 5000
                }
            );

            const exists = response.status === 200;
            this._setCache(cacheKey, exists);

            logger.debug('Store item validation', { itemId, exists });
            return exists;
        } catch (error) {
            if (error.response && error.response.status === 404) {
                this._setCache(cacheKey, false);
                logger.debug('Store item not found', { itemId });
                return false;
            }

            logger.warn('Store item validation failed', {
                itemId,
                error: error.message,
                status: error.response?.status
            });

            // Fail open on errors
            return true;
        }
    }

    /**
     * Validate message context (task, application, or store item)
     * @param {Object} context - Message context
     * @param {number} context.taskId - Task ID (optional)
     * @param {number} context.applicationId - Application ID (optional)
     * @param {number} context.storeItemId - Store item ID (optional)
     * @param {string} token - JWT token for authentication
     * @returns {Promise<Object>} - Validation results
     */
    async validateMessageContext(context, token) {
        const results = {
            valid: true,
            errors: []
        };

        // Validate task if provided
        if (context.taskId) {
            const taskExists = await this.validateTask(context.taskId, token);
            if (!taskExists) {
                results.valid = false;
                results.errors.push(`Task ${context.taskId} not found`);
            }
        }

        // Validate application if provided
        if (context.applicationId) {
            const appExists = await this.validateApplication(context.applicationId, token);
            if (!appExists) {
                results.valid = false;
                results.errors.push(`Application ${context.applicationId} not found`);
            }
        }

        // Validate store item if provided
        if (context.storeItemId) {
            const itemExists = await this.validateStoreItem(context.storeItemId, token);
            if (!itemExists) {
                results.valid = false;
                results.errors.push(`Store item ${context.storeItemId} not found`);
            }
        }

        return results;
    }

    /**
     * Clear validation cache
     */
    clearCache() {
        this.cache.clear();
        logger.info('Validation cache cleared');
    }

    /**
     * Get cache statistics
     */
    getCacheStats() {
        return {
            size: this.cache.size,
            entries: Array.from(this.cache.keys())
        };
    }
}

// Export singleton instance
module.exports = new ValidationService();
