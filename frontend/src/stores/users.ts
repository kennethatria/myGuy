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
  created_at?: string
  createdAt?: string // Keep both for compatibility
}

export const useUsersStore = defineStore('users', () => {
  // Cache of users we've already fetched, keyed by user ID
  const userCache = ref<Map<number, User>>(new Map())

  // Get user by ID - fetches from API and caches the result
  const getUserById = async (userId: number): Promise<User | null> => {
    // If we already have this user in the cache, return it
    if (userCache.value.has(userId)) {
      console.log(`✅ Using cached user data for ID ${userId}`);
      return userCache.value.get(userId) || null;
    }

    console.log(`🔄 Fetching user data from API for ID ${userId}`);

    try {
      const authStore = useAuthStore();
      const token = authStore.token;

      if (!token) {
        console.error('❌ No authentication token available');
        throw new Error('Authentication required');
      }

      const apiUrl = `${config.API_URL}/users/${userId}`;
      console.log(`📡 API Request: GET ${apiUrl}`);

      const response = await fetch(apiUrl, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        }
      });

      console.log(`📥 API Response: ${response.status} ${response.statusText}`);

      if (response.ok) {
        const userData = await response.json();
        console.log(`✅ User data loaded successfully:`, {
          id: userData.id,
          username: userData.username,
          fullName: userData.full_name || userData.fullName
        });

        // Normalize field names (backend uses snake_case, frontend uses camelCase)
        const normalizedUser: User = {
          id: userData.id,
          username: userData.username,
          email: userData.email,
          fullName: userData.full_name || userData.fullName,
          bio: userData.bio,
          averageRating: userData.average_rating || userData.averageRating,
          created_at: userData.created_at || userData.createdAt,
          createdAt: userData.created_at || userData.createdAt
        };

        // Cache for future use
        userCache.value.set(userId, normalizedUser);
        return normalizedUser;
      } else {
        // Log the error response
        const errorText = await response.text();
        console.error(`❌ API Error: ${response.status} - ${errorText}`);

        if (response.status === 404) {
          throw new Error(`User with ID ${userId} not found`);
        } else if (response.status === 401) {
          throw new Error('Authentication failed - please log in again');
        } else {
          throw new Error(`Failed to load user: ${response.status} ${response.statusText}`);
        }
      }
    } catch (error) {
      console.error(`❌ Error fetching user ${userId}:`, error);
      // Re-throw the error so it can be handled by the calling component
      throw error;
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

    console.log(`Fetching or creating mock data for ${idsToFetch.length} users`);
    
    // Get each user individually using our getUserById method 
    // which will handle API fetch or mock data creation
    await Promise.all(idsToFetch.map(async (id) => {
      try {
        const user = await getUserById(id);
        if (user) {
          result.set(id, user);
        }
      } catch (error) {
        console.error(`Error getting user ${id}:`, error);
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