import { defineStore } from 'pinia'
import { ref } from 'vue'
import config from '@/config'
import { useAuthStore } from './auth'

interface Task {
  id: number
  title: string
  description: string
  status: 'open' | 'in_progress' | 'completed'
  createdBy: number
  assignedTo?: number
  deadline: string
  fee?: number
  created_at: string
  
  // Related data from database preloading
  creator?: {
    id: number
    username: string
    fullName?: string
  }
  assignee?: {
    id: number
    username: string
    fullName?: string
  }
  applications?: Application[]
}

interface Application {
  id: number
  taskId: number
  applicant: {
    id: number
    username: string
  }
  proposedFee: number
  status: 'pending' | 'accepted' | 'declined'
  message?: string
  created_at: string
}

type ApplicationInput = Omit<Application, 'id' | 'status' | 'created_at' | 'taskId' | 'applicant'>

export const useTasksStore = defineStore('tasks', () => {
  const tasks = ref<Task[]>([])
  const userTasks = ref<Task[]>([])
  const assignedTasks = ref<Task[]>([])

  const getTask = async (taskId: number): Promise<Task> => {
    const authStore = useAuthStore();
    const token = authStore.token;
    
    try {
      console.log(`Fetching task with ID: ${taskId}`);
      
      // First check if we already have this task in our local state
      const cachedTask = tasks.value.find(t => t.id === taskId) || 
                         userTasks.value.find(t => t.id === taskId) ||
                         assignedTasks.value.find(t => t.id === taskId);
      
      if (cachedTask) {
        console.log(`Using cached task data for ID ${taskId}`);
        return cachedTask;
      }
      
      // If not in cache, fetch from API
      const response = await fetch(`${config.ENDPOINTS.TASKS}/${taskId}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (!response.ok) {
        console.error(`Error response for task ${taskId}:`, response.status, response.statusText);
        throw new Error('Failed to fetch task');
      }
      
      const taskData = await response.json();
      console.log(`Successfully fetched task data for ID ${taskId}:`, taskData);
      return taskData;
    } catch (error) {
      console.error(`Error fetching task ${taskId}:`, error);
      throw error;
    }
  }

  const getTaskApplications = async (taskId: number): Promise<Application[]> => {
    const authStore = useAuthStore();
    const token = authStore.token;
    
    try {
      // Since the applications data is now included in the task itself from the backend
      const task = await getTask(taskId)
      if (task.applications && Array.isArray(task.applications)) {
        return task.applications
      }
      
      // Fallback to legacy endpoint if task doesn't include applications
      const response = await fetch(`${config.ENDPOINTS.TASKS}/${taskId}/applications`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      })
      if (!response.ok) throw new Error('Failed to fetch task applications')
      return await response.json()
    } catch (error) {
      console.error('Error fetching task applications:', error)
      throw error
    }
  }

  const fetchTasks = async () => {
    const authStore = useAuthStore();
    const token = authStore.token;
    const userId = authStore.user?.id;
    
    try {
      // Add query parameter to exclude tasks created by current user
      const url = new URL(config.ENDPOINTS.TASKS);
      
      // Critical: Add parameter to only fetch tasks NOT created by current user
      if (userId) {
        url.searchParams.append('exclude_created_by', String(userId));
        url.searchParams.append('status', 'open');
      }
      
      console.log('Fetching tasks with URL:', url.toString());
      
      const response = await fetch(url.toString(), {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      })
      if (!response.ok) throw new Error('Failed to fetch tasks')
      
      // Get tasks from API
      const allTasks = await response.json();
      
      // Additional client-side filtering as backup
      if (userId) {
        console.log('User ID for filtering:', userId);
        console.log('All tasks before filtering:', allTasks.length);
        
        // Apply client-side filter to exclude user's own tasks
        tasks.value = allTasks.filter(task => 
          String(task.createdBy) !== String(userId) && 
          task.status === 'open'
        );
        
        console.log('Tasks after filtering:', tasks.value.length);
      } else {
        tasks.value = allTasks;
      }
    } catch (error) {
      console.error('Error fetching tasks:', error)
      throw error
    }
  }

  const fetchUserTasks = async () => {
    const authStore = useAuthStore();
    const token = authStore.token;
    
    try {
      // Get the current user ID
      const userId = authStore.user?.id;
      if (!userId) {
        // If we don't have the user ID yet, try to get the profile
        await authStore.checkAuth();
      }
      
      const response = await fetch(`${config.ENDPOINTS.TASKS}?created_by=${userId || authStore.user?.id}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      })
      
      if (!response.ok) {
        if (response.status === 401) {
          // Token expired or invalid, trigger a relogin
          authStore.logout();
          throw new Error('Please log in again to continue')
        }
        throw new Error('Failed to fetch user tasks')
      }
      
      userTasks.value = await response.json()
    } catch (error) {
      console.error('Error fetching user tasks:', error)
      throw error
    }
  }

  const fetchAssignedTasks = async () => {
    const authStore = useAuthStore();
    const token = authStore.token;
    
    try {
      // Get the current user ID
      const userId = authStore.user?.id;
      if (!userId) {
        // If we don't have the user ID yet, try to get the profile
        await authStore.checkAuth();
      }
      
      const response = await fetch(`${config.ENDPOINTS.TASKS}?assigned_to=${userId || authStore.user?.id}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      })
      
      if (!response.ok) {
        if (response.status === 401) {
          // Token expired or invalid, trigger a relogin
          authStore.logout();
          throw new Error('Please log in again to continue')
        }
        throw new Error('Failed to fetch assigned tasks')
      }
      
      assignedTasks.value = await response.json()
    } catch (error) {
      console.error('Error fetching assigned tasks:', error)
      throw error
    }
  }

  const createTask = async (task: Omit<Task, 'id' | 'status' | 'createdBy' | 'assignedTo' | 'created_at'>) => {
    const authStore = useAuthStore();
    const token = authStore.token;
    
    try {
      const response = await fetch(config.ENDPOINTS.TASKS, {
        method: 'POST',
        headers: { 
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json' 
        },
        body: JSON.stringify(task)
      })
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to create task');
      }
      
      return await response.json()
    } catch (error) {
      console.error('Error creating task:', error)
      throw error
    }
  }

  const updateTaskStatus = async (taskId: number, status: Task['status']) => {
    const authStore = useAuthStore();
    const token = authStore.token;
    
    try {
      const response = await fetch(`${config.ENDPOINTS.TASKS}/${taskId}/status`, {
        method: 'PATCH',
        headers: { 
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json' 
        },
        body: JSON.stringify({ status })
      })
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to update task status');
      }
      
      return await response.json()
    } catch (error) {
      console.error('Error updating task status:', error)
      throw error
    }
  }

  const applyForTask = async (taskId: number, application: ApplicationInput) => {
    const authStore = useAuthStore();
    const token = authStore.token;
    
    try {
      const response = await fetch(`${config.ENDPOINTS.TASKS}/${taskId}/apply`, {
        method: 'POST',
        headers: { 
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json' 
        },
        body: JSON.stringify(application)
      })
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to apply for task');
      }
      
      return await response.json()
    } catch (error) {
      console.error('Error applying for task:', error)
      throw error
    }
  }

  const respondToApplication = async (taskId: number, applicationId: number, status: 'accepted' | 'declined') => {
    const authStore = useAuthStore();
    const token = authStore.token;
    
    try {
      const response = await fetch(`${config.ENDPOINTS.TASKS}/${taskId}/applications/${applicationId}`, {
        method: 'PATCH',
        headers: { 
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json' 
        },
        body: JSON.stringify({ status })
      })
      if (!response.ok) throw new Error('Failed to respond to application')
      return await response.json()
    } catch (error) {
      console.error('Error responding to application:', error)
      throw error
    }
  }

  return {
    tasks,
    userTasks,
    assignedTasks,
    getTask,
    getTaskApplications,
    fetchTasks,
    fetchUserTasks,
    fetchAssignedTasks,
    createTask,
    updateTaskStatus,
    applyForTask,
    respondToApplication
  }
})
