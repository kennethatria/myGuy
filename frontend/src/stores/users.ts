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
  // Mock user data for development (using real database user data)
  const mockUsers: User[] = [
    { id: 1, username: "test_user", fullName: "Test User", email: "testuser@example.com", averageRating: 4.5, created_at: "2025-05-26T18:00:41.084585Z" },
    { id: 2, username: "alice_dev", fullName: "Alice Developer", email: "alice@example.com", averageRating: 4.2, created_at: "2025-05-26T18:37:57.166095Z" },
    { id: 3, username: "bob_designer", fullName: "Bob Designer", email: "bob@example.com", averageRating: 4.8, created_at: "2025-05-26T18:37:57.228835Z" },
    { id: 4, username: "charlie_writer", fullName: "Charlie Writer", email: "charlie@example.com", averageRating: 3.9, created_at: "2025-05-26T18:37:57.287916Z" },
    { id: 5, username: "diana_coder", fullName: "Diana Coder", email: "diana@example.com", averageRating: 4.7, created_at: "2025-05-26T18:37:57.342056Z" }
  ];

  // Cache of users we've already fetched, keyed by user ID
  const userCache = ref<Map<number, User>>(new Map())
  
  // Initialize cache with mock users
  mockUsers.forEach(user => {
    userCache.value.set(user.id, user);
  })

  // Get user by ID - returns from cache if available, otherwise uses mock data
  const getUserById = async (userId: number): Promise<User | null> => {
    // If we already have this user in the cache, return it
    if (userCache.value.has(userId)) {
      console.log(`Using cached user data for ID ${userId}`);
      return userCache.value.get(userId) || null;
    }

    // For development - generate mock user if needed
    console.log(`Creating mock user data for ID ${userId}`);
    
    try {
      // First try to fetch from API
      const authStore = useAuthStore();
      const token = authStore.token;
      
      try {
        // USER API endpoint - try it first, but expect it may not exist
        const response = await fetch(`${config.API_URL}/users/${userId}`, {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          }
        });

        if (response.ok) {
          const userData = await response.json();
          // Cache for future use
          userCache.value.set(userId, userData);
          return userData;
        }
      } catch (apiError) {
        console.warn(`API endpoint for users likely doesn't exist:`, apiError);
      }
      
      // Generate a mock user as fallback
      const randomNames = ["Alex", "Morgan", "Jordan", "Taylor", "Riley", "Casey", "Jamie", "Avery"];
      const randomLastNames = ["Smith", "Johnson", "Williams", "Jones", "Brown", "Davis", "Miller", "Wilson"];
      const randomName = randomNames[Math.floor(Math.random() * randomNames.length)];
      const randomLastName = randomLastNames[Math.floor(Math.random() * randomLastNames.length)];
      
      // Generate a random join date between 1-18 months ago
      const monthsAgo = Math.floor(Math.random() * 18) + 1;
      const joinDate = new Date();
      joinDate.setMonth(joinDate.getMonth() - monthsAgo);
      
      const mockUser: User = {
        id: userId,
        username: `${randomName.toLowerCase()}${userId}`,
        fullName: `${randomName} ${randomLastName}`,
        email: `${randomName.toLowerCase()}${userId}@example.com`,
        averageRating: (3 + Math.random() * 2).toFixed(1) as unknown as number, // 3.0-5.0 rating
        created_at: joinDate.toISOString(),
      };
      
      // Cache the mock user
      userCache.value.set(userId, mockUser);
      console.log(`Created mock user:`, mockUser);
      
      return mockUser;
    } catch (error) {
      console.error(`Error creating mock user ${userId}:`, error);
      
      // Absolute fallback - just return a minimal user object
      const fallbackUser = {
        id: userId,
        username: `User ${userId}`,
      };
      
      // Cache the fallback
      userCache.value.set(userId, fallbackUser);
      
      return fallbackUser;
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