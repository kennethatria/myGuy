import { defineStore } from 'pinia';
import { ref } from 'vue';
import config from '@/config';
import { useAuthStore } from './auth';

export interface Task {
  id: number;
  title: string;
  description?: string;
  status?: string;
}

export interface StoreItem {
  id: number;
  title: string;
  description?: string;
  price?: number;
}

export const useContextStore = defineStore('context', () => {
  // Caches for context data
  const taskCache = ref<Map<number, Task>>(new Map());
  const itemCache = ref<Map<number, StoreItem>>(new Map());

  // Track ongoing fetches
  const pendingTaskFetches = ref<Map<number, Promise<Task | null>>>(new Map());
  const pendingItemFetches = ref<Map<number, Promise<StoreItem | null>>>(new Map());

  /**
   * Get a task from cache by ID
   */
  function getTaskById(taskId: number): Task | undefined {
    return taskCache.value.get(taskId);
  }

  /**
   * Get a store item from cache by ID
   */
  function getItemById(itemId: number): StoreItem | undefined {
    return itemCache.value.get(itemId);
  }

  /**
   * Fetch a single task by ID and add to cache
   */
  async function fetchTask(taskId: number): Promise<Task | null> {
    // Return from cache if already loaded
    if (taskCache.value.has(taskId)) {
      return taskCache.value.get(taskId)!;
    }

    // Return pending fetch if already in progress
    if (pendingTaskFetches.value.has(taskId)) {
      return pendingTaskFetches.value.get(taskId)!;
    }

    // Start new fetch
    const fetchPromise = performTaskFetch(taskId);
    pendingTaskFetches.value.set(taskId, fetchPromise);

    try {
      const task = await fetchPromise;
      return task;
    } finally {
      pendingTaskFetches.value.delete(taskId);
    }
  }

  /**
   * Internal function to perform task API call
   */
  async function performTaskFetch(taskId: number): Promise<Task | null> {
    const authStore = useAuthStore();
    const token = authStore.token;

    if (!token) {
      console.warn('No auth token available for fetching task');
      return null;
    }

    try {
      const response = await fetch(`${config.API_URL}/tasks/${taskId}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        }
      });

      if (!response.ok) {
        console.warn(`Failed to fetch task ${taskId}: ${response.status}`);
        return null;
      }

      const task: Task = await response.json();

      // Add to cache
      taskCache.value.set(taskId, task);

      return task;
    } catch (error) {
      console.error(`Error fetching task ${taskId}:`, error);
      return null;
    }
  }

  /**
   * Fetch a single store item by ID and add to cache
   */
  async function fetchItem(itemId: number): Promise<StoreItem | null> {
    // Return from cache if already loaded
    if (itemCache.value.has(itemId)) {
      return itemCache.value.get(itemId)!;
    }

    // Return pending fetch if already in progress
    if (pendingItemFetches.value.has(itemId)) {
      return pendingItemFetches.value.get(itemId)!;
    }

    // Start new fetch
    const fetchPromise = performItemFetch(itemId);
    pendingItemFetches.value.set(itemId, fetchPromise);

    try {
      const item = await fetchPromise;
      return item;
    } finally {
      pendingItemFetches.value.delete(itemId);
    }
  }

  /**
   * Internal function to perform store item API call
   */
  async function performItemFetch(itemId: number): Promise<StoreItem | null> {
    const authStore = useAuthStore();
    const token = authStore.token;

    if (!token) {
      console.warn('No auth token available for fetching store item');
      return null;
    }

    try {
      const response = await fetch(`${config.STORE_API_URL}/items/${itemId}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        }
      });

      if (!response.ok) {
        console.warn(`Failed to fetch store item ${itemId}: ${response.status}`);
        return null;
      }

      const item: StoreItem = await response.json();

      // Add to cache
      itemCache.value.set(itemId, item);

      return item;
    } catch (error) {
      console.error(`Error fetching store item ${itemId}:`, error);
      return null;
    }
  }

  /**
   * Fetch multiple tasks in batch
   */
  async function fetchTasks(taskIds: number[]): Promise<Map<number, Task>> {
    const uniqueIds = [...new Set(taskIds)];
    const results = new Map<number, Task>();

    await Promise.all(
      uniqueIds.map(async (taskId) => {
        const task = await fetchTask(taskId);
        if (task) {
          results.set(taskId, task);
        }
      })
    );

    return results;
  }

  /**
   * Fetch multiple store items in batch
   */
  async function fetchItems(itemIds: number[]): Promise<Map<number, StoreItem>> {
    const uniqueIds = [...new Set(itemIds)];
    const results = new Map<number, StoreItem>();

    await Promise.all(
      uniqueIds.map(async (itemId) => {
        const item = await fetchItem(itemId);
        if (item) {
          results.set(itemId, item);
        }
      })
    );

    return results;
  }

  /**
   * Clear all caches
   */
  function clearCache() {
    taskCache.value.clear();
    itemCache.value.clear();
    pendingTaskFetches.value.clear();
    pendingItemFetches.value.clear();
  }

  return {
    taskCache,
    itemCache,
    getTaskById,
    getItemById,
    fetchTask,
    fetchItem,
    fetchTasks,
    fetchItems,
    clearCache
  };
});
