import { defineStore } from 'pinia'
import { ref } from 'vue'
import config from '@/config'
import { useAuthStore } from './auth'

export interface Message {
  id: number
  task_id: number
  application_id?: number
  sender_id: number
  recipient_id: number
  content: string
  is_read: boolean
  read_at?: string
  is_edited: boolean
  edited_at?: string
  is_deleted: boolean
  deleted_at?: string
  created_at: string
  has_removed_content?: boolean
  sender: {
    id: number
    username: string
  }
  recipient: {
    id: number
    username: string
  }
}

export interface ConversationSummary {
  task_id?: number
  application_id?: number
  task_title?: string
  application_title?: string
  task_description?: string
  task_status?: string
  last_message: string
  last_message_time: string
  other_user_id: number
  other_user_name: string
  unread_count: number
  conversation_type: 'task' | 'application'
}

export const useMessagesStore = defineStore('messages', () => {
  const messages = ref<Message[]>([])

  const fetchTaskMessages = async (taskId: number): Promise<Message[]> => {
    const authStore = useAuthStore();
    const token = authStore.token;
    
    try {
      console.log(`Fetching messages for task ID: ${taskId}`);
      const response = await fetch(`${config.ENDPOINTS.TASKS}/${taskId}/messages`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        }
      });
      
      if (!response.ok) {
        console.warn(`Failed to fetch messages, status: ${response.status}`);
        // Return empty array instead of throwing to prevent UI breakage
        messages.value = [];
        return [];
      }
      
      const data = await response.json();
      console.log(`Fetched ${data.length} messages for task ${taskId}`);
      
      // Ensure all messages have the required properties
      const validatedData = data.map((msg: any) => ({
        ...msg,
        id: msg.id || Math.random(), // Ensure ID exists
        sender: msg.sender || { id: 0, username: 'Unknown User' }, // Ensure sender exists
        content: msg.content || '',
        created_at: msg.created_at || new Date().toISOString()
      }));
      
      messages.value = validatedData;
      return validatedData;
    } catch (error) {
      console.error('Error fetching messages:', error);
      // Return empty array instead of throwing to prevent UI breakage
      messages.value = [];
      return [];
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
        body: JSON.stringify({ recipient_id: recipientId, content }),
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

  const fetchApplicationMessages = async (applicationId: number): Promise<Message[]> => {
    const authStore = useAuthStore();
    const token = authStore.token;
    
    try {
      console.log(`Fetching messages for application ID: ${applicationId}`);
      const response = await fetch(`${config.ENDPOINTS.APPLICATIONS}/${applicationId}/messages`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        }
      })
      
      if (!response.ok) {
        throw new Error('Failed to fetch application messages')
      }
      
      const messagesData = await response.json()
      console.log(`Fetched ${messagesData.length} messages for application`);
      return messagesData
    } catch (error) {
      console.error('Error fetching application messages:', error)
      throw error
    }
  }

  const sendApplicationMessage = async (applicationId: number, content: string): Promise<Message> => {
    const authStore = useAuthStore();
    const token = authStore.token;
    
    try {
      const response = await fetch(`${config.ENDPOINTS.APPLICATIONS}/${applicationId}/messages`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ content }),
      })
      
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to send message')
      }
      
      const newMessage = await response.json()
      return newMessage
    } catch (error) {
      console.error('Error sending application message:', error)
      throw error
    }
  }

  return {
    messages,
    fetchTaskMessages,
    sendMessage,
    fetchApplicationMessages,
    sendApplicationMessage,
  }
})
