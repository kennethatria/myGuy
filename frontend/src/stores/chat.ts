import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { io, Socket } from 'socket.io-client';
import { useAuthStore } from './auth';
import config from '@/config';
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
  const chatUnavailable = ref(false);
  const connectionError = ref<string | null>(null);
  const reconnectAttempts = ref(0);
  const conversations = ref<ConversationSummary[]>([]);
  const activeConversation = ref<ConversationSummary | null>(null);
  const messages = ref<Map<number, Message[]>>(new Map());
  const storeMessages = ref<Map<number, Message[]>>(new Map()); // Store messages by item ID
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
    const conversationId = activeConversation.value.task_id || activeConversation.value.application_id || activeConversation.value.item_id;
    
    // Return store messages if it's a store conversation
    if (activeConversation.value.conversation_type === 'store') {
      return storeMessages.value.get(conversationId!) || [];
    }
    
    // Return regular messages for task/application conversations
    return messages.value.get(conversationId!) || [];
  });
  
  const activeTypingUsers = computed(() => {
    if (!activeConversation.value) return [];
    const conversationId = activeConversation.value.task_id || activeConversation.value.application_id || activeConversation.value.item_id;
    return typingUsers.value.get(conversationId!) || [];
  });
  
  const activeHasMoreMessages = computed(() => {
    if (!activeConversation.value) return false;
    const conversationId = activeConversation.value.task_id || activeConversation.value.application_id || activeConversation.value.item_id;
    
    // Get total message count for this conversation
    const totalCount = totalMessageCounts.value.get(conversationId!) || 0;
    
    // Only show "Load more" if there are more than 20 messages AND there are more pages to load
    const hasMorePages = hasMoreMessages.value.get(conversationId!) || false;
    
    return totalCount > 20 && hasMorePages;
  });
  
  // Socket connection
  // Store message methods
  async function joinStoreConversation(itemId: number) {
    if (!socket.value?.connected) await connectSocket();
    socket.value?.emit('join:conversation', { itemId });
  }

  async function sendStoreMessage(content: string, recipientId: number, itemId: number) {
    if (!socket.value?.connected) return;
    
    socket.value.emit('message:send', {
      itemId,
      recipientId,
      content
    });
  }

  function getStoreMessages(itemId: number): Message[] {
    return storeMessages.value.get(itemId) || [];
  }

  function connectSocket() {
    if (socket.value?.connected) return;

    try {
      const chatUrl = config.CHAT_WS_URL;

      socket.value = io(chatUrl, {
        auth: {
          token: authStore.token
        },
        reconnection: true,
        reconnectionDelay: 1000,
        reconnectionDelayMax: 5000,
        reconnectionAttempts: 10, // Increased from 3 to 10
        timeout: 5000,
        transports: ['websocket', 'polling']
      });
    } catch (error) {
      console.error('Failed to initialize WebSocket connection:', error);
      chatUnavailable.value = true;
      connectionError.value = error instanceof Error ? error.message : 'Unknown error';
      return;
    }

    // Connection events
    socket.value.on('connect', () => {
      connected.value = true;
      chatUnavailable.value = false;
      connectionError.value = null;
      reconnectAttempts.value = 0;
      console.log('✓ Chat WebSocket connected');

      // Load conversations on connect with delay to ensure auth is processed
      setTimeout(() => {
        console.log('Requesting conversations via WebSocket...');
        socket.value?.emit('conversations:list');

        // Test WebSocket communication
        console.log('Testing WebSocket with ping...');
        socket.value?.emit('test:ping');
      }, 500);
    });

    socket.value.on('disconnect', (reason: string) => {
      connected.value = false;
      console.log('WebSocket disconnected:', reason);

      // Mark as unavailable if disconnected by server or failed to connect
      if (reason === 'io server disconnect' || reason === 'transport close') {
        chatUnavailable.value = true;
        connectionError.value = `Disconnected: ${reason}`;
      }
    });

    socket.value.on('connect_error', (error: any) => {
      reconnectAttempts.value++;
      console.warn(`Chat connection attempt ${reconnectAttempts.value} failed:`, error.message);
      connectionError.value = error.message;

      // Mark as unavailable after multiple failed attempts
      if (reconnectAttempts.value >= 5) {
        chatUnavailable.value = true;
        console.error('⚠️ Chat service unavailable after multiple connection attempts');
      }
    });

    socket.value.on('error', (error: any) => {
      console.error('WebSocket error:', error);
      connectionError.value = error?.message || 'Unknown error';

      // Don't break the app on WebSocket errors
      if (error?.message) {
        console.warn('Chat service error:', error.message);
      }
    });
    
    // Message events
    socket.value.on('message:new', (message: Message) => {
      if (message.store_item_id) {
        // Handle store message
        const messages = storeMessages.value.get(message.store_item_id) || [];
        storeMessages.value.set(message.store_item_id, [...messages, message]);
        
        // Add to conversations if not exists
        const existingConv = conversations.value.find(c => c.item_id === message.store_item_id);
        if (!existingConv) {
          conversations.value.push({
            item_id: message.store_item_id,
            item_title: message.store_item_title || 'Store Item',
            last_message: message.content,
            last_message_time: message.created_at,
            other_user_id: message.sender_id === authStore.user?.id ? message.recipient_id : message.sender_id,
            other_user_name: message.sender_id === authStore.user?.id ? 
              (message.recipient?.username || 'Unknown User') : 
              (message.sender?.username || 'Unknown User'),
            unread_count: message.sender_id !== authStore.user?.id ? 1 : 0,
            conversation_type: 'store'
          });
        } else {
          // Update existing conversation
          existingConv.last_message = message.content;
          existingConv.last_message_time = message.created_at;
          if (message.sender_id !== authStore.user?.id) {
            existingConv.unread_count = (existingConv.unread_count || 0) + 1;
          }
        }
      } else {
        // Handle task/application message
        handleNewMessage(message);
      }
    });

    socket.value.on('message:sent', (message: Message) => {
      if (message.store_item_id) {
        // Handle store message
        const messages = storeMessages.value.get(message.store_item_id) || [];
        storeMessages.value.set(message.store_item_id, [...messages, message]);
        
        // Add to conversations if not exists
        const existingConv = conversations.value.find(c => c.item_id === message.store_item_id);
        if (!existingConv) {
          conversations.value.push({
            item_id: message.store_item_id,
            item_title: 'Store Item',  // We'll update this with the actual title when available
            last_message: message.content,
            last_message_time: message.created_at,
            other_user_id: message.recipient_id,
            other_user_name: message.recipient?.username || 'Unknown User',
            unread_count: 0,
            conversation_type: 'store'
          });
        } else {
          // Update existing conversation
          existingConv.last_message = message.content;
          existingConv.last_message_time = message.created_at;
        }
      } else {
        // Handle task/application message
        handleMessageSent(message);
      }
    });
    
    socket.value.on('message:edited', handleMessageEdited);
    socket.value.on('message:deleted', handleMessageDeleted);
    socket.value.on('message:read', handleMessageRead);
    socket.value.on('message:filtered', handleMessageFiltered);
    socket.value.on('message:notification', handleMessageNotification);
    
    // Conversation events
    socket.value.on('conversations:list', handleConversationsList);
    socket.value.on('conversations:refresh', handleConversationsRefresh);
    socket.value.on('messages:list', handleMessagesList);
    socket.value.on('test:pong', (data) => {
      console.log('Received test:pong:', data);
    });
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
    const conversationId = message.task_id || message.application_id || message.item_id;
    if (!conversationId) return;
    
    const conversationMessages = messages.value.get(conversationId) || [];
    messages.value.set(conversationId, [...conversationMessages, message]);
    
    // Update total message count
    const currentCount = totalMessageCounts.value.get(conversationId) || 0;
    totalMessageCounts.value.set(conversationId, currentCount + 1);
    
    // Update conversation last message
    const conv = conversations.value.find(c => 
      c.task_id === conversationId || 
      c.application_id === conversationId || 
      c.item_id === conversationId
    );
    if (conv) {
      conv.last_message = message.content;
      conv.last_message_time = message.created_at;
      
      // Increment unread count if not the active conversation
      const activeConvId = activeConversation.value?.task_id || activeConversation.value?.application_id || activeConversation.value?.item_id;
      if (activeConvId !== conversationId && message.sender_id !== authStore.user?.id) {
        const currentUnread = unreadCounts.value.get(conversationId) || 0;
        unreadCounts.value.set(conversationId, currentUnread + 1);
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
    const conversationId = message.task_id || message.application_id || message.item_id;
    if (!conversationId) return;
    
    const conversationMessages = messages.value.get(conversationId) || [];
    const index = conversationMessages.findIndex(m => m.id === message.id);
    if (index !== -1) {
      conversationMessages[index] = message;
      messages.value.set(conversationId, [...conversationMessages]);
    }
  }
  
  function handleMessageDeleted({ messageId }: { messageId: number }) {
    messages.value.forEach((conversationMessages, conversationId) => {
      const index = conversationMessages.findIndex(m => m.id === messageId);
      if (index !== -1) {
        conversationMessages[index].content = '[Message deleted]';
        conversationMessages[index].is_deleted = true;
        messages.value.set(conversationId, [...conversationMessages]);
      }
    });
  }
  
  function handleMessageRead({ messageId, readAt }: { messageId: number; readAt: string }) {
    messages.value.forEach((conversationMessages, conversationId) => {
      const message = conversationMessages.find(m => m.id === messageId);
      if (message) {
        message.is_read = true;
        message.read_at = readAt;
        messages.value.set(conversationId, [...conversationMessages]);
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
      const conversationId = conv.task_id || conv.application_id || conv.item_id;
      if (conversationId) {
        unreadCounts.value.set(conversationId, conv.unread_count);
      }
    });
  }
  
  function handleConversationsRefresh() {
    // Refresh conversations list by emitting the request
    if (socket.value) {
      socket.value.emit('conversations:list');
    }
  }

  // HTTP fallback for loading conversations
  async function loadConversationsHttp() {
    const authStore = useAuthStore();
    const token = authStore.token;
    
    if (!token) return;
    
    try {
      console.log('Loading conversations via HTTP...');
      const response = await fetch(`${config.CHAT_API_URL}/conversations`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        }
      });
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      
      const conversations = await response.json();
      console.log('Loaded conversations via HTTP:', conversations);
      
      // Process conversations the same way as WebSocket
      handleConversationsList(conversations);
      
    } catch (error) {
      console.error('Error loading conversations via HTTP:', error);
    }
  }
  
  function handleMessagesList({ taskId, applicationId, itemId, messages: msgs, offset, totalCount }: { taskId?: number; applicationId?: number; itemId?: number; messages: Message[]; offset: number; totalCount?: number }) {
    const conversationId = taskId || applicationId || itemId;
    if (!conversationId) return;
    
    if (offset === 0) {
      messages.value.set(conversationId, msgs);
    } else {
      // Prepend older messages
      const existing = messages.value.get(conversationId) || [];
      messages.value.set(conversationId, [...msgs, ...existing]);
    }
    
    // Store total count if provided
    if (totalCount !== undefined) {
      totalMessageCounts.value.set(conversationId, totalCount);
    }
    
    // If we got less than requested, there are no more messages
    hasMoreMessages.value.set(conversationId, msgs.length === 5);
    isLoadingMessages.value = false;
  }
  
  function handleConversationMarkedRead({ taskId, applicationId, itemId, count }: { taskId?: number; applicationId?: number; itemId?: number; count: number }) {
    const conversationId = taskId || applicationId || itemId;
    if (!conversationId) return;
    
    unreadCounts.value.set(conversationId, 0);
    const conv = conversations.value.find(c => c.task_id === conversationId || c.application_id === conversationId || c.item_id === conversationId);
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
    
    const conv = conversations.value.find(c => c.task_id === conversationId || c.application_id === conversationId || c.item_id === conversationId);
    if (conv) {
      // Store reference to previous conversation BEFORE updating activeConversation
      const previousConversation = activeConversation.value;
      
      // Leave previous conversation if it exists and is different
      if (previousConversation && 
          (previousConversation.task_id !== conv.task_id || 
           previousConversation.application_id !== conv.application_id ||
           previousConversation.item_id !== conv.item_id)) {
        const prevId = previousConversation.task_id || previousConversation.application_id || previousConversation.item_id;
        if (previousConversation.task_id) {
          socket.value.emit('leave:conversation', { taskId: prevId });
        } else if (previousConversation.application_id) {
          socket.value.emit('leave:conversation', { applicationId: prevId });
        } else if (previousConversation.item_id) {
          socket.value.emit('leave:conversation', { itemId: prevId });
        }
      }
      
      // Set new active conversation
      activeConversation.value = conv;
      
      // Join new conversation
      if (conv.task_id) {
        socket.value.emit('join:conversation', { taskId: conversationId });
      } else if (conv.application_id) {
        socket.value.emit('join:conversation', { applicationId: conversationId });
      } else if (conv.item_id) {
        socket.value.emit('join:conversation', { itemId: conversationId });
      }
      
      // Load messages if not already loaded
      if (!messages.value.has(conversationId)) {
        isLoadingMessages.value = true;
        if (conv.task_id) {
          socket.value.emit('messages:get', { taskId: conversationId, limit: 5, offset: 0 });
        } else if (conv.application_id) {
          socket.value.emit('messages:get', { applicationId: conversationId, limit: 5, offset: 0 });
        } else if (conv.item_id) {
          socket.value.emit('messages:get', { itemId: conversationId, limit: 5, offset: 0 });
        }
      }
      
      // Mark as read
      if (conv.unread_count > 0) {
        if (conv.task_id) {
          socket.value.emit('conversation:read', { taskId: conversationId });
        } else if (conv.application_id) {
          socket.value.emit('conversation:read', { applicationId: conversationId });
        } else if (conv.item_id) {
          socket.value.emit('conversation:read', { itemId: conversationId });
        }
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
    } else if (activeConversation.value.item_id) {
      socket.value.emit('message:send', {
        itemId: activeConversation.value.item_id,
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
    
    const conversationId = activeConversation.value.task_id || activeConversation.value.application_id || activeConversation.value.item_id;
    if (!conversationId) return;
    
    const currentMessages = messages.value.get(conversationId) || [];
    
    if (!hasMoreMessages.value.get(conversationId)) return;
    
    isLoadingMessages.value = true;
    
    if (activeConversation.value.task_id) {
      socket.value.emit('messages:get', {
        taskId: conversationId,
        limit: 5,
        offset: currentMessages.length
      });
    } else if (activeConversation.value.application_id) {
      socket.value.emit('messages:get', {
        applicationId: conversationId,
        limit: 5,
        offset: currentMessages.length
      });
    } else if (activeConversation.value.item_id) {
      socket.value.emit('messages:get', {
        itemId: conversationId,
        limit: 5,
        offset: currentMessages.length
      });
    }
  }
  
  function startTyping() {
    if (!socket.value || !activeConversation.value) return;
    
    if (activeConversation.value.task_id) {
      socket.value.emit('typing:start', {
        taskId: activeConversation.value.task_id
      });
    } else if (activeConversation.value.application_id) {
      socket.value.emit('typing:start', {
        applicationId: activeConversation.value.application_id
      });
    } else if (activeConversation.value.item_id) {
      socket.value.emit('typing:start', {
        itemId: activeConversation.value.item_id
      });
    }
  }
  
  function stopTyping() {
    if (!socket.value || !activeConversation.value) return;
    
    if (activeConversation.value.task_id) {
      socket.value.emit('typing:stop', {
        taskId: activeConversation.value.task_id
      });
    } else if (activeConversation.value.application_id) {
      socket.value.emit('typing:stop', {
        applicationId: activeConversation.value.application_id
      });
    } else if (activeConversation.value.item_id) {
      socket.value.emit('typing:stop', {
        itemId: activeConversation.value.item_id
      });
    }
  }
  
  async function loadDeletionWarnings() {
    try {
      if (!authStore.token) {
        console.warn('No auth token available, skipping deletion warnings');
        return;
      }
      
      const chatApiUrl = config.CHAT_API_URL;
      const response = await fetch(`${chatApiUrl}/deletion-warnings`, {
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
      const chatApiUrl = config.CHAT_API_URL;
      await fetch(`${chatApiUrl}/deletion-warnings/${warningId}/shown`, {
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
    chatUnavailable,
    connectionError,
    reconnectAttempts,
    conversations,
    activeConversation,
    messages,
    storeMessages,
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
    
    // Store message methods
    getStoreMessages,
    joinStoreConversation,
    sendStoreMessage,
    
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
    dismissWarning,
    loadConversationsHttp
  };
});