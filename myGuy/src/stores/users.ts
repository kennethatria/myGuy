import { defineStore } from 'pinia'
import { ref } from 'vue'
import config from '@/config'
import { useAuthStore } from './auth'

interface User {
  id: number
  username: string
  email?: string
  fullName?: string
  bio?: string
  averageRating?: number
  createdAt?: string
}

export const useUsersStore = defineStore('users', () => {
  // Cache of users we've already fetched, keyed by user ID
  const userCache = ref<Map<number, User>>(new Map())

  // Get user by ID - returns from cache if available, otherwise fetches from API
  const getUserById = async (userId: number): Promise<User | null> => {
    // If we already have this user in the cache, return it
    if (userCache.value.has(userId)) {
      console.log(`Using cached user data for ID ${userId}`);
      return userCache.value.get(userId) || null;
    }

    // Otherwise fetch from API
    console.log(`Fetching user data for ID ${userId}`);
    try {
      const authStore = useAuthStore();
      const token = authStore.token;

      // USER API endpoint - assuming there's a /users/:id endpoint
      // If your API has a different structure, adjust this accordingly
      const response = await fetch(`${config.API_URL}/users/${userId}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        }
      });

      if (!response.ok) {
        console.warn(`Failed to fetch user with ID ${userId}`);
        // Create a minimal placeholder user as fallback
        const fallbackUser = {
          id: userId,
          username: `User ${userId}`,
        };
        // Don't cache the fallback to allow future fetch attempts
        return fallbackUser;
      }

      const userData = await response.json();
      
      // Cache for future use
      userCache.value.set(userId, userData);
      
      return userData;
    } catch (error) {
      console.error(`Error fetching user ${userId}:`, error);
      // Return minimal user object as fallback
      return {
        id: userId,
        username: `User ${userId}`,
      };
    }
  }

  // Get multiple users at once - useful for lists of items
  const getUsersByIds = async (userIds: number[]): Promise<Map<number, User>> => {
    const uniqueIds = [...new Set(userIds)]; // Remove duplicates
    const result = new Map<number, User>();

    // First add any users we already have in the cache
    uniqueIds.forEach(id => {
      if (userCache.value.has(id)) {
        result.set(id, userCache.value.get(id)!);
      }
    });

    // Determine which IDs we still need to fetch
    const idsToFetch = uniqueIds.filter(id => !userCache.value.has(id));
    
    if (idsToFetch.length === 0) {
      return result; // All users were in cache
    }

    // Fetch the remaining users (implementation depends on API)
    // If your API supports batch fetching, you could use that here
    await Promise.all(idsToFetch.map(async (id) => {
      try {
        const user = await getUserById(id);
        if (user) {
          result.set(id, user);
        }
      } catch (error) {
        console.error(`Error fetching user ${id}:`, error);
      }
    }));

    return result;
  }

  return {
    userCache,
    getUserById,
    getUsersByIds
  }
})