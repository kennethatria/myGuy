import { defineStore } from 'pinia'
import { ref } from 'vue'

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
    try {
      const response = await fetch(`/api/tasks/${taskId}`)
      if (!response.ok) throw new Error('Failed to fetch task')
      return await response.json()
    } catch (error) {
      console.error('Error fetching task:', error)
      throw error
    }
  }

  const getTaskApplications = async (taskId: number): Promise<Application[]> => {
    try {
      const response = await fetch(`/api/tasks/${taskId}/applications`)
      if (!response.ok) throw new Error('Failed to fetch task applications')
      return await response.json()
    } catch (error) {
      console.error('Error fetching task applications:', error)
      throw error
    }
  }

  const fetchTasks = async () => {
    try {
      const response = await fetch('/api/tasks')
      if (!response.ok) throw new Error('Failed to fetch tasks')
      tasks.value = await response.json()
    } catch (error) {
      console.error('Error fetching tasks:', error)
      throw error
    }
  }

  const fetchUserTasks = async () => {
    try {
      const response = await fetch('/api/users/me/tasks')
      if (!response.ok) throw new Error('Failed to fetch user tasks')
      userTasks.value = await response.json()
    } catch (error) {
      console.error('Error fetching user tasks:', error)
      throw error
    }
  }

  const fetchAssignedTasks = async () => {
    try {
      const response = await fetch('/api/users/me/assigned-tasks')
      if (!response.ok) throw new Error('Failed to fetch assigned tasks')
      assignedTasks.value = await response.json()
    } catch (error) {
      console.error('Error fetching assigned tasks:', error)
      throw error
    }
  }

  const createTask = async (task: Omit<Task, 'id' | 'status' | 'createdBy' | 'assignedTo' | 'created_at'>) => {
    try {
      const response = await fetch('/api/tasks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(task)
      })
      if (!response.ok) throw new Error('Failed to create task')
      return await response.json()
    } catch (error) {
      console.error('Error creating task:', error)
      throw error
    }
  }

  const updateTaskStatus = async (taskId: number, status: Task['status']) => {
    try {
      const response = await fetch(`/api/tasks/${taskId}/status`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status })
      })
      if (!response.ok) throw new Error('Failed to update task status')
      return await response.json()
    } catch (error) {
      console.error('Error updating task status:', error)
      throw error
    }
  }

  const applyForTask = async (taskId: number, application: ApplicationInput) => {
    try {
      const response = await fetch(`/api/tasks/${taskId}/apply`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(application)
      })
      if (!response.ok) throw new Error('Failed to apply for task')
      return await response.json()
    } catch (error) {
      console.error('Error applying for task:', error)
      throw error
    }
  }

  const respondToApplication = async (taskId: number, applicationId: number, status: 'accepted' | 'declined') => {
    try {
      const response = await fetch(`/api/tasks/${taskId}/applications/${applicationId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
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
