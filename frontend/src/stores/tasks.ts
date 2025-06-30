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
      console.log(`Fetching applications for task ID: ${taskId}`);
      
      // Since the applications data is now included in the task itself from the backend
      const task = await getTask(taskId);
      if (task.applications && Array.isArray(task.applications)) {
        console.log(`Using ${task.applications.length} applications from task data`);
        return task.applications;
      }
      
      // Fallback to legacy endpoint if task doesn't include applications
      try {
        const response = await fetch(`${config.ENDPOINTS.TASKS}/${taskId}/applications`, {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        });
        
        if (!response.ok) {
          console.warn(`Failed to fetch applications, status: ${response.status}`);
          return []; // Return empty array instead of throwing
        }
        
        const applications = await response.json();
        console.log(`Fetched ${applications.length} applications via API`);
        return applications;
      } catch (innerError) {
        console.error('Error in API fetch for applications:', innerError);
        return []; // Return empty array to prevent UI breakage
      }
    } catch (error) {
      console.error('Error fetching task applications:', error);
      return []; // Return empty array to prevent UI breakage
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
      const result = await response.json();
      
      // Handle different response formats
      let allTasks;
      if (result.data && Array.isArray(result.data)) {
        allTasks = result.data;
      } else if (Array.isArray(result)) {
        allTasks = result;
      } else {
        console.error('Unexpected response format:', result);
        allTasks = [];
      }
      
      // Additional client-side filtering as backup
      if (userId && Array.isArray(allTasks)) {
        console.log('User ID for filtering:', userId);
        console.log('All tasks before filtering:', allTasks.length);
        
        // Apply client-side filter to exclude user's own tasks
        tasks.value = allTasks.filter(task => 
          String(task.createdBy || task.created_by) !== String(userId) && 
          task.status === 'open'
        );
        
        console.log('Tasks after filtering:', tasks.value.length);
      } else {
        tasks.value = Array.isArray(allTasks) ? allTasks : [];
      }
      
      return { data: tasks.value };
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
      
      const response = await fetch(`${config.API_URL}/user/tasks`, {
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
      
      // Add exclude_self_assigned=true parameter to exclude tasks the user created themselves
      const response = await fetch(`${config.API_URL}/user/tasks/assigned?exclude_self_assigned=true`, {
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
      
      const assignedTasksData = await response.json();
      console.log('DEBUG - Assigned tasks from API:', assignedTasksData);
      
      // Add extra safety check: only include tasks that actually have assignedTo matching user ID
      const currentUserId = authStore.user?.id;
      if (currentUserId) {
        assignedTasks.value = assignedTasksData.filter(task => {
          const isActuallyAssigned = task.assigned_to === currentUserId || 
                                     (typeof task.assigned_to === 'string' && task.assigned_to === String(currentUserId));
          
          if (!isActuallyAssigned) {
            console.warn(`Task ${task.id} was returned by assigned_to API but doesn't have matching assigned_to value:`, 
                          {taskAssignedTo: task.assigned_to, currentUserId, task});
          }
          
          return isActuallyAssigned;
        });
        
        console.log(`DEBUG - After filtering, ${assignedTasks.value.length} of ${assignedTasksData.length} tasks remain`);
      } else {
        assignedTasks.value = assignedTasksData;
      }
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
      // Convert camelCase to snake_case for API
      const apiPayload = {
        proposed_fee: application.proposedFee,
        message: application.message
      }
      
      const response = await fetch(`${config.ENDPOINTS.TASKS}/${taskId}/apply`, {
        method: 'POST',
        headers: { 
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json' 
        },
        body: JSON.stringify(apiPayload)
      })
      
      if (!response.ok) {
        const errorText = await response.text();
        console.error('Application failed:', response.status, errorText);
        try {
          const errorData = JSON.parse(errorText);
          throw new Error(errorData.error || 'Failed to apply for task');
        } catch (e) {
          throw new Error(`Failed to apply for task: ${response.status} ${response.statusText}`);
        }
      }
      
      // Check if response has content
      const responseText = await response.text();
      if (!responseText) {
        // API returned empty response but it was successful
        return { success: true };
      }
      
      try {
        return JSON.parse(responseText);
      } catch (e) {
        console.warn('Could not parse response as JSON:', responseText);
        return { success: true };
      }
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
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Unknown error' }))
        console.error('Server error response:', errorData)
        throw new Error(errorData.error || 'Failed to respond to application')
      }
      
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
