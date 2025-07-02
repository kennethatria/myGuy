import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { io, Socket } from 'socket.io-client';
import { useAuthStore } from './auth';
import type { Message, ConversationSummary } from './messages';

interface TypingUser {
  userId: number;
  userName: string;
}

export const useChatStore = defineStore('chat', () => {
  const authStore = useAuthStore();
  
  // State
  const socket = ref<Socket | null>(null);
  const connected = ref(false);
  const conversations = ref<ConversationSummary[]>([]);
  const activeConversation = ref<ConversationSummary | null>(null);
  const messages = ref<Map<number, Message[]>>(new Map());
  const typingUsers = ref<Map<number, TypingUser[]>>(new Map());
  const unreadCounts = ref<Map<number, number>>(new Map());
  const isLoadingMessages = ref(false);
  const hasMoreMessages = ref<Map<number, boolean>>(new Map());
  const totalMessageCounts = ref<Map<number, number>>(new Map());
  const deletionWarnings = ref<any[]>([]);
  
  // Computed
  const totalUnreadCount = computed(() => {
    let total = 0;
    unreadCounts.value.forEach(count => total += count);
    return total;
  });
  
  const activeMessages = computed(() => {
    if (!activeConversation.value) return [];
    const conversationId = activeConversation.value.task_id || activeConversation.value.application_id;
    return messages.value.get(conversationId!) || [];
  });
  
  const activeTypingUsers = computed(() => {
    if (!activeConversation.value) return [];
    const conversationId = activeConversation.value.task_id || activeConversation.value.application_id;
    return typingUsers.value.get(conversationId!) || [];
  });
  
  const activeHasMoreMessages = computed(() => {
    if (!activeConversation.value) return false;
    const conversationId = activeConversation.value.task_id || activeConversation.value.application_id;
    
    // Get total message count for this conversation
    const totalCount = totalMessageCounts.value.get(conversationId!) || 0;
    
    // Only show "Load more" if there are more than 20 messages AND there are more pages to load
    const hasMorePages = hasMoreMessages.value.get(conversationId!) || false;
    
    return totalCount > 20 && hasMorePages;
  });
  
  // Socket connection
  function connectSocket() {
    if (socket.value?.connected) return;
    
    try {
      socket.value = io('http://localhost:8082', {
        auth: {
          token: authStore.token
        },
        reconnection: true,
        reconnectionDelay: 1000,
        reconnectionAttempts: 3,
        timeout: 5000
      });
    } catch (error) {
      console.error('Failed to initialize WebSocket connection:', error);
      return;
    }
    
    // Connection events
    socket.value.on('connect', () => {
      connected.value = true;
      console.log('WebSocket connected');
      
      // Load conversations on connect
      socket.value?.emit('conversations:list');
    });
    
    socket.value.on('disconnect', () => {
      connected.value = false;
      console.log('WebSocket disconnected');
    });
    
    socket.value.on('error', (error: any) => {
      console.error('WebSocket error:', error);
      // Don't break the app on WebSocket errors
      if (error?.message) {
        console.warn('Chat service unavailable:', error.message);
      }
    });
    
    // Message events
    socket.value.on('message:new', handleNewMessage);
    socket.value.on('message:sent', handleMessageSent);
    socket.value.on('message:edited', handleMessageEdited);
    socket.value.on('message:deleted', handleMessageDeleted);
    socket.value.on('message:read', handleMessageRead);
    socket.value.on('message:filtered', handleMessageFiltered);
    socket.value.on('message:notification', handleMessageNotification);
    
    // Conversation events
    socket.value.on('conversations:list', handleConversationsList);
    socket.value.on('messages:list', handleMessagesList);
    socket.value.on('conversation:marked-read', handleConversationMarkedRead);
    
    // Typing events
    socket.value.on('user:typing', handleUserTyping);
    socket.value.on('user:stopped-typing', handleUserStoppedTyping);
    
    // User presence
    socket.value.on('user:lastseen', handleUserLastSeen);
  }
  
  function disconnectSocket() {
    if (socket.value) {
      socket.value.disconnect();
      socket.value = null;
      connected.value = false;
    }
  }
  
  // Event handlers
  function handleNewMessage(message: Message) {
    const taskId = message.task_id;
    const taskMessages = messages.value.get(taskId) || [];
    messages.value.set(taskId, [...taskMessages, message]);
    
    // Update total message count
    const currentCount = totalMessageCounts.value.get(taskId) || 0;
    totalMessageCounts.value.set(taskId, currentCount + 1);
    
    // Update conversation last message
    const conv = conversations.value.find(c => c.task_id === taskId);
    if (conv) {
      conv.last_message = message.content;
      conv.last_message_time = message.created_at;
      
      // Increment unread count if not the active conversation
      if (activeConversation.value?.task_id !== taskId && message.sender_id !== authStore.user?.id) {
        const currentUnread = unreadCounts.value.get(taskId) || 0;
        unreadCounts.value.set(taskId, currentUnread + 1);
        conv.unread_count = currentUnread + 1;
      }
    }
    
    // Sort conversations by last message time
    conversations.value.sort((a, b) => 
      new Date(b.last_message_time).getTime() - new Date(a.last_message_time).getTime()
    );
  }
  
  function handleMessageSent(message: Message) {
    handleNewMessage(message);
  }
  
  function handleMessageEdited(message: Message) {
    const taskId = message.task_id;
    const taskMessages = messages.value.get(taskId) || [];
    const index = taskMessages.findIndex(m => m.id === message.id);
    if (index !== -1) {
      taskMessages[index] = message;
      messages.value.set(taskId, [...taskMessages]);
    }
  }
  
  function handleMessageDeleted({ messageId }: { messageId: number }) {
    messages.value.forEach((taskMessages, taskId) => {
      const index = taskMessages.findIndex(m => m.id === messageId);
      if (index !== -1) {
        taskMessages[index].content = '[Message deleted]';
        taskMessages[index].is_deleted = true;
        messages.value.set(taskId, [...taskMessages]);
      }
    });
  }
  
  function handleMessageRead({ messageId, readAt }: { messageId: number; readAt: string }) {
    messages.value.forEach((taskMessages, taskId) => {
      const message = taskMessages.find(m => m.id === messageId);
      if (message) {
        message.is_read = true;
        message.read_at = readAt;
        messages.value.set(taskId, [...taskMessages]);
      }
    });
  }
  
  function handleMessageFiltered({ messageId, warning }: { messageId: number; warning: string }) {
    // Show warning to user
    alert(warning);
  }
  
  function handleMessageNotification({ message, conversationId }: { message: Message; conversationId: number }) {
    // Update unread count for the conversation
    if (message.sender_id !== authStore.user?.id) {
      const currentUnread = unreadCounts.value.get(conversationId) || 0;
      unreadCounts.value.set(conversationId, currentUnread + 1);
      
      const conv = conversations.value.find(c => c.task_id === conversationId);
      if (conv) {
        conv.unread_count = currentUnread + 1;
      }
    }
  }
  
  function handleConversationsList(convs: ConversationSummary[]) {
    conversations.value = convs;
    
    // Update unread counts
    convs.forEach(conv => {
      const conversationId = conv.task_id || conv.application_id;
      if (conversationId) {
        unreadCounts.value.set(conversationId, conv.unread_count);
      }
    });
  }
  
  function handleMessagesList({ taskId, messages: msgs, offset, totalCount }: { taskId: number; messages: Message[]; offset: number; totalCount?: number }) {
    if (offset === 0) {
      messages.value.set(taskId, msgs);
    } else {
      // Prepend older messages
      const existing = messages.value.get(taskId) || [];
      messages.value.set(taskId, [...msgs, ...existing]);
    }
    
    // Store total count if provided
    if (totalCount !== undefined) {
      totalMessageCounts.value.set(taskId, totalCount);
    }
    
    // If we got less than requested, there are no more messages
    hasMoreMessages.value.set(taskId, msgs.length === 5);
    isLoadingMessages.value = false;
  }
  
  function handleConversationMarkedRead({ taskId, count }: { taskId: number; count: number }) {
    unreadCounts.value.set(taskId, 0);
    const conv = conversations.value.find(c => c.task_id === taskId || c.application_id === taskId);
    if (conv) {
      conv.unread_count = 0;
    }
  }
  
  function handleUserTyping({ userId, userName, conversationId }: { userId: number; userName: string; conversationId: number }) {
    if (userId === authStore.user?.id) return;
    
    const users = typingUsers.value.get(conversationId) || [];
    if (!users.find(u => u.userId === userId)) {
      typingUsers.value.set(conversationId, [...users, { userId, userName }]);
    }
    
    // Remove after 3 seconds
    setTimeout(() => {
      handleUserStoppedTyping({ userId, conversationId });
    }, 3000);
  }
  
  function handleUserStoppedTyping({ userId, conversationId }: { userId: number; conversationId: number }) {
    const users = typingUsers.value.get(conversationId) || [];
    typingUsers.value.set(conversationId, users.filter(u => u.userId !== userId));
  }
  
  function handleUserLastSeen({ userId, lastSeen }: { userId: number; lastSeen: string }) {
    // Update user's last seen in conversations
    conversations.value.forEach(conv => {
      if (conv.other_user_id === userId) {
        // Store last seen data
      }
    });
  }
  
  // Actions
  function joinConversation(conversationId: number) {
    if (!socket.value) return;
    
    const conv = conversations.value.find(c => c.task_id === conversationId || c.application_id === conversationId);
    if (conv) {
      // Store reference to previous conversation BEFORE updating activeConversation
      const previousConversation = activeConversation.value;
      
      // Leave previous conversation if it exists and is different
      if (previousConversation && 
          (previousConversation.task_id !== conv.task_id || 
           previousConversation.application_id !== conv.application_id)) {
        const prevId = previousConversation.task_id || previousConversation.application_id;
        if (previousConversation.task_id) {
          socket.value.emit('leave:conversation', { taskId: prevId });
        } else {
          socket.value.emit('leave:conversation', { applicationId: prevId });
        }
      }
      
      // Set new active conversation
      activeConversation.value = conv;
      
      // Join new conversation
      if (conv.task_id) {
        socket.value.emit('join:conversation', { taskId: conversationId });
      } else {
        socket.value.emit('join:conversation', { applicationId: conversationId });
      }
      
      // Load messages if not already loaded
      if (!messages.value.has(conversationId)) {
        isLoadingMessages.value = true;
        socket.value.emit('messages:get', { taskId: conversationId, limit: 5, offset: 0 });
      }
      
      // Mark as read
      if (conv.unread_count > 0) {
        socket.value.emit('conversation:read', { taskId: conversationId });
      }
    }
  }
  
  function sendMessage(content: string, recipientId: number) {
    if (!socket.value || !activeConversation.value) return;
    
    if (activeConversation.value.task_id) {
      socket.value.emit('message:send', {
        taskId: activeConversation.value.task_id,
        recipientId,
        content
      });
    } else if (activeConversation.value.application_id) {
      socket.value.emit('message:send', {
        applicationId: activeConversation.value.application_id,
        recipientId,
        content
      });
    }
  }
  
  function editMessage(messageId: number, content: string) {
    if (!socket.value) return;
    
    socket.value.emit('message:edit', {
      messageId,
      content
    });
  }
  
  function deleteMessage(messageId: number) {
    if (!socket.value) return;
    
    socket.value.emit('message:delete', {
      messageId
    });
  }
  
  function markMessageAsRead(messageId: number) {
    if (!socket.value) return;
    
    socket.value.emit('message:read', {
      messageId
    });
  }
  
  function loadMoreMessages() {
    if (!socket.value || !activeConversation.value || isLoadingMessages.value) return;
    
    const taskId = activeConversation.value.task_id;
    const currentMessages = messages.value.get(taskId) || [];
    
    if (!hasMoreMessages.value.get(taskId)) return;
    
    isLoadingMessages.value = true;
    socket.value.emit('messages:get', {
      taskId,
      limit: 5,
      offset: currentMessages.length
    });
  }
  
  function startTyping() {
    if (!socket.value || !activeConversation.value) return;
    
    socket.value.emit('typing:start', {
      taskId: activeConversation.value.task_id
    });
  }
  
  function stopTyping() {
    if (!socket.value || !activeConversation.value) return;
    
    socket.value.emit('typing:stop', {
      taskId: activeConversation.value.task_id
    });
  }
  
  async function loadDeletionWarnings() {
    try {
      if (!authStore.token) {
        console.warn('No auth token available, skipping deletion warnings');
        return;
      }
      
      const response = await fetch('http://localhost:8082/api/v1/deletion-warnings', {
        headers: {
          'Authorization': `Bearer ${authStore.token}`
        }
      });
      
      if (response.ok) {
        deletionWarnings.value = await response.json();
      } else if (response.status === 401) {
        console.warn('Unauthorized to access deletion warnings');
      } else {
        console.error('Failed to load deletion warnings:', response.status, response.statusText);
      }
    } catch (error) {
      console.error('Failed to load deletion warnings:', error);
    }
  }
  
  async function dismissWarning(warningId: number) {
    try {
      await fetch(`http://localhost:8082/api/v1/deletion-warnings/${warningId}/shown`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${authStore.token}`
        }
      });
      
      deletionWarnings.value = deletionWarnings.value.filter(w => w.id !== warningId);
    } catch (error) {
      console.error('Failed to dismiss warning:', error);
    }
  }
  
  return {
    // State
    socket,
    connected,
    conversations,
    activeConversation,
    messages,
    typingUsers,
    unreadCounts,
    isLoadingMessages,
    hasMoreMessages,
    deletionWarnings,
    
    // Computed
    totalUnreadCount,
    activeMessages,
    activeTypingUsers,
    activeHasMoreMessages,
    
    // Actions
    connectSocket,
    disconnectSocket,
    joinConversation,
    sendMessage,
    editMessage,
    deleteMessage,
    markMessageAsRead,
    loadMoreMessages,
    startTyping,
    stopTyping,
    loadDeletionWarnings,
    dismissWarning
  };
});