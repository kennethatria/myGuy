import { defineStore } from 'pinia';
import { ref } from 'vue';
import config from '@/config';
import { useAuthStore } from './auth';

export interface User {
  id: number;
  username: string;
  email?: string;
  name?: string;
}

export const useUserStore = defineStore('user', () => {
  // Cache of user data: Map<userId, User>
  const userCache = ref<Map<number, User>>(new Map());

  // Track ongoing fetch requests to prevent duplicate API calls
  const pendingFetches = ref<Map<number, Promise<User | null>>>(new Map());

  /**
   * Get a user from the cache by ID
   */
  function getUserById(userId: number): User | undefined {
    return userCache.value.get(userId);
  }

  /**
   * Fetch a single user by ID and add to cache
   */
  async function fetchUser(userId: number): Promise<User | null> {
    // Return from cache if already loaded
    if (userCache.value.has(userId)) {
      return userCache.value.get(userId)!;
    }

    // Return pending fetch if already in progress
    if (pendingFetches.value.has(userId)) {
      return pendingFetches.value.get(userId)!;
    }

    // Start new fetch
    const fetchPromise = performUserFetch(userId);
    pendingFetches.value.set(userId, fetchPromise);

    try {
      const user = await fetchPromise;
      return user;
    } finally {
      pendingFetches.value.delete(userId);
    }
  }

  /**
   * Internal function to perform the actual API call
   */
  async function performUserFetch(userId: number): Promise<User | null> {
    const authStore = useAuthStore();
    const token = authStore.token;

    if (!token) {
      console.warn('No auth token available for fetching user');
      return null;
    }

    try {
      const response = await fetch(`${config.API_URL}/users/${userId}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        }
      });

      if (!response.ok) {
        console.warn(`Failed to fetch user ${userId}: ${response.status}`);
        return null;
      }

      const user: User = await response.json();

      // Add to cache
      userCache.value.set(userId, user);

      return user;
    } catch (error) {
      console.error(`Error fetching user ${userId}:`, error);
      return null;
    }
  }

  /**
   * Fetch multiple users in batch
   * Note: This uses individual requests since the backend doesn't have a batch endpoint yet
   */
  async function fetchUsers(userIds: number[]): Promise<Map<number, User>> {
    const uniqueIds = [...new Set(userIds)];
    const results = new Map<number, User>();

    // Fetch all users in parallel
    await Promise.all(
      uniqueIds.map(async (userId) => {
        const user = await fetchUser(userId);
        if (user) {
          results.set(userId, user);
        }
      })
    );

    return results;
  }

  /**
   * Preload the current user into the cache
   */
  function cacheCurrentUser() {
    const authStore = useAuthStore();
    if (authStore.user) {
      const user: User = {
        id: authStore.user.id,
        username: authStore.user.username,
        email: authStore.user.email,
        name: authStore.user.name
      };
      userCache.value.set(user.id, user);
    }
  }

  /**
   * Clear the user cache (useful for logout)
   */
  function clearCache() {
    userCache.value.clear();
    pendingFetches.value.clear();
  }

  return {
    userCache,
    getUserById,
    fetchUser,
    fetchUsers,
    cacheCurrentUser,
    clearCache
  };
});
