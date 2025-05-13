import { defineStore } from 'pinia'
import { ref } from 'vue'
import config from '@/config'
import { useAuthStore } from './auth'

interface Message {
  id: number
  taskId: number
  sender: {
    id: number
    username: string
  }
  content: string
  createdAt: string
}

export const useMessagesStore = defineStore('messages', () => {
  const messages = ref<Message[]>([])

  const fetchTaskMessages = async (taskId: number): Promise<Message[]> => {
    const authStore = useAuthStore();
    const token = authStore.token;
    
    try {
      const response = await fetch(`${config.ENDPOINTS.TASKS}/${taskId}/messages`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        }
      })
      if (!response.ok) throw new Error('Failed to fetch messages')
      const data = await response.json()
      messages.value = data
      return data
    } catch (error) {
      console.error('Error fetching messages:', error)
      throw error
    }
  }

  const sendMessage = async (taskId: number, recipientId: number, content: string): Promise<Message> => {
    const authStore = useAuthStore();
    const token = authStore.token;
    
    try {
      const response = await fetch(`${config.ENDPOINTS.TASKS}/${taskId}/messages`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ recipientId, content }),
      })
      if (!response.ok) throw new Error('Failed to send message')
      const newMessage = await response.json()
      messages.value.push(newMessage)
      return newMessage
    } catch (error) {
      console.error('Error sending message:', error)
      throw error
    }
  }

  return {
    messages,
    fetchTaskMessages,
    sendMessage,
  }
})
