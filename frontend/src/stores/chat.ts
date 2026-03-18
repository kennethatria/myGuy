import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { io, Socket } from 'socket.io-client';
import { useAuthStore } from './auth';
import { useUserStore } from './user';
import { useContextStore } from './context';
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
  const deletionWarnings = ref<{ id: number; task_id: number; task_title: string; deletion_scheduled_at: string }[]>([]);
  
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

    // Wait briefly for conversations to load if empty
    if (conversations.value.length === 0) {
      await new Promise(resolve => setTimeout(resolve, 500));
    }

    // Try to find existing conversation
    let conv = conversations.value.find(c => c.item_id === itemId);

    if (!conv) {
      // Conversation doesn't exist yet, create placeholder
      try {
        const response = await fetch(`${config.STORE_API_URL}/items/${itemId}`);
        if (response.ok) {
          const item = await response.json();

          conv = {
            item_id: itemId,
            item_title: item.title,
            last_message: '',
            last_message_time: new Date().toISOString(),
            other_user_id: item.seller_id,
            other_user_name: item.seller?.name || item.seller?.username || 'Seller',
            unread_count: 0,
            conversation_type: 'store'
          };

          conversations.value.push(conv);
        }
      } catch (error) {
        console.error('Failed to create conversation placeholder:', error);
        return; // Exit if we can't create conversation
      }
    }

    // If conv is still undefined (e.g., response was not ok), exit early
    if (!conv) return;

    // Follow same logic as joinConversation
    const previousConversation = activeConversation.value;

    if (previousConversation && previousConversation.item_id !== itemId) {
      const prevId = previousConversation.task_id || previousConversation.application_id || previousConversation.item_id;
      if (previousConversation.item_id) {
        socket.value?.emit('leave:conversation', { itemId: prevId });
      }
    }

    activeConversation.value = conv;
    socket.value?.emit('join:conversation', { itemId });

    // Load messages if not already loaded
    const hasMessages = storeMessages.value.has(itemId);
    if (!hasMessages) {
      isLoadingMessages.value = true;
      socket.value?.emit('messages:get', { itemId, limit: 20, offset: 0 });
    }

    // Mark as read
    if (conv.unread_count > 0) {
      socket.value?.emit('conversation:read', { itemId });
    }
  }

  // Helper for modal: join with retry logic
  async function joinStoreConversationWithRetry(itemId: number, maxRetries = 5): Promise<boolean> {
    for (let attempt = 0; attempt < maxRetries; attempt++) {
      const conv = conversations.value.find(c => c.item_id === itemId);

      if (conv) {
        await joinStoreConversation(itemId);
        return true;
      }

      // Wait with exponential backoff: 1s, 2s, 3s, 4s, 5s
      await new Promise(resolve => setTimeout(resolve, (attempt + 1) * 1000));

      // Refresh conversations list
      socket.value?.emit('conversations:list');
    }

    return false; // Failed after all retries
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

    socket.value.on('connect_error', (error: Error) => {
      reconnectAttempts.value++;
      console.warn(`Chat connection attempt ${reconnectAttempts.value} failed:`, error.message);
      connectionError.value = error.message;

      // Mark as unavailable after multiple failed attempts
      if (reconnectAttempts.value >= 5) {
        chatUnavailable.value = true;
        console.error('⚠️ Chat service unavailable after multiple connection attempts');
      }
    });

    socket.value.on('error', (error: Error) => {
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
            item_title: 'Store Item', // Will be enriched by enrichConversations
            last_message: message.content,
            last_message_time: message.created_at,
            other_user_id: message.sender_id === authStore.user?.id ? message.recipient_id : message.sender_id,
            other_user_name: message.sender_id === authStore.user?.id ?
              (message.recipient?.username || 'Unknown User') :
              (message.sender?.username || 'Unknown User'),
            unread_count: message.sender_id !== authStore.user?.id ? 1 : 0,
            conversation_type: 'store'
          });

          // Enrich the new conversation
          enrichConversations();
        } else {
          // Update existing conversation
          existingConv.last_message = message.content;
          existingConv.last_message_type = message.message_type;
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
          existingConv.last_message_type = message.message_type;
          existingConv.last_message_time = message.created_at;
        }
      } else {
        // Handle task/application message
        handleMessageSent(message);
      }
    });
    
    socket.value.on('message:edited', handleMessageEdited);
    socket.value.on('message:updated', handleMessageUpdated);
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

    // Enrich the new message with sender data
    enrichMessages([message]);

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

  function handleMessageUpdated(message: Message) {
    // Handle booking status updates and other message updates
    const conversationId = message.task_id || message.application_id || message.store_item_id;
    if (!conversationId) return;

    // Update in regular messages map
    const conversationMessages = messages.value.get(conversationId) || [];
    const index = conversationMessages.findIndex(m => m.id === message.id);
    if (index !== -1) {
      conversationMessages[index] = { ...conversationMessages[index], ...message };
      messages.value.set(conversationId, [...conversationMessages]);
    }

    // Also update in store messages if it's a store item
    if (message.store_item_id) {
      const storeMessageList = storeMessages.value.get(message.store_item_id) || [];
      const storeIndex = storeMessageList.findIndex(m => m.id === message.id);
      if (storeIndex !== -1) {
        storeMessageList[storeIndex] = { ...storeMessageList[storeIndex], ...message };
        storeMessages.value.set(message.store_item_id, [...storeMessageList]);
      }
    }

    console.log('✅ Message updated:', message.id, 'metadata:', message.metadata);
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
  
  function handleMessageFiltered({ warning }: { messageId?: number; warning: string }) {
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

    // Enrich conversations with user names and context titles
    enrichConversations();
  }

  /**
   * Enrich conversations with user names and task/item titles
   */
  async function enrichConversations() {
    const userStore = useUserStore();
    const contextStore = useContextStore();

    // Collect all unique user IDs and context IDs
    const userIds = new Set<number>();
    const taskIds: number[] = [];
    const itemIds: number[] = [];

    conversations.value.forEach(conv => {
      if (conv.other_user_id) {
        userIds.add(conv.other_user_id);
      }
      if (conv.task_id) {
        taskIds.push(conv.task_id);
      }
      if (conv.item_id) {
        itemIds.push(conv.item_id);
      }
    });

    // Fetch all users in parallel
    await userStore.fetchUsers([...userIds]);

    // Fetch all tasks and items in parallel
    await Promise.all([
      contextStore.fetchTasks(taskIds),
      contextStore.fetchItems(itemIds)
    ]);

    // Update conversations with enriched data
    conversations.value.forEach(conv => {
      // Enrich user name
      const user = userStore.getUserById(conv.other_user_id);
      if (user) {
        conv.other_user_name = user.username;
      }

      // Enrich task title
      if (conv.task_id) {
        const task = contextStore.getTaskById(conv.task_id);
        if (task) {
          conv.task_title = task.title;
        }
      }

      // Enrich store item title
      if (conv.item_id) {
        const item = contextStore.getItemById(conv.item_id);
        if (item) {
          conv.item_title = item.title;
        }
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
    // Ensure IDs are numbers to match Map keys
    const parsedTaskId = taskId ? Number(taskId) : undefined;
    const parsedApplicationId = applicationId ? Number(applicationId) : undefined;
    const parsedItemId = itemId ? Number(itemId) : undefined;

    const conversationId = parsedTaskId || parsedApplicationId || parsedItemId;
    if (!conversationId) return;

    // Enrich messages with sender/recipient data
    enrichMessages(msgs);

    // Route messages to correct storage Map based on conversation type
    if (parsedItemId) {
      if (offset === 0) {
        storeMessages.value.set(parsedItemId, msgs);
      } else {
        // Prepend older messages
        const existing = storeMessages.value.get(parsedItemId) || [];
        storeMessages.value.set(parsedItemId, [...msgs, ...existing]);
      }
    } else {
      if (offset === 0) {
        messages.value.set(conversationId, msgs);
      } else {
        // Prepend older messages
        const existing = messages.value.get(conversationId) || [];
        messages.value.set(conversationId, [...msgs, ...existing]);
      }
    }

    // Store total count if provided
    if (totalCount !== undefined) {
      totalMessageCounts.value.set(conversationId, totalCount);
    }

    // If we got less than requested, there are no more messages
    hasMoreMessages.value.set(conversationId, msgs.length === 5);
    isLoadingMessages.value = false;
  }

  /**
   * Enrich messages with sender and recipient user data
   */
  async function enrichMessages(msgs: Message[]) {
    const userStore = useUserStore();

    // Collect all unique user IDs from messages
    const userIds = new Set<number>();
    msgs.forEach(msg => {
      if (msg.sender_id) {
        userIds.add(msg.sender_id);
      }
      if (msg.recipient_id) {
        userIds.add(msg.recipient_id);
      }
    });

    // Fetch all users
    await userStore.fetchUsers([...userIds]);

    // Attach sender and recipient data to each message
    msgs.forEach(msg => {
      const sender = userStore.getUserById(msg.sender_id);
      if (sender) {
        msg.sender = {
          id: sender.id,
          username: sender.username
        };
      }

      const recipient = userStore.getUserById(msg.recipient_id);
      if (recipient) {
        msg.recipient = {
          id: recipient.id,
          username: recipient.username
        };
      }
    });
  }
  
  function handleConversationMarkedRead({ taskId, applicationId, itemId }: { taskId?: number; applicationId?: number; itemId?: number; count?: number }) {
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
  
  function handleUserLastSeen({ userId }: { userId: number; lastSeen?: string }) {
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
      
      // Load messages if not already loaded - check correct Map based on conversation type
      const hasMessages = conv.item_id
        ? storeMessages.value.has(conversationId)
        : messages.value.has(conversationId);

      if (!hasMessages) {
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

    // Get messages from correct Map based on conversation type
    const currentMessages = activeConversation.value.item_id
      ? (storeMessages.value.get(conversationId) || [])
      : (messages.value.get(conversationId) || []);

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

  async function handleBookingAction(
    bookingId: number,
    action: 'approve' | 'decline' | 'confirm-received' | 'confirm-delivery' | 'rate-seller' | 'rate-buyer',
    rating?: number,
    review?: string
  ) {
    try {
      const chatApiUrl = config.CHAT_API_URL;
      const body: { bookingId: number; action: string; rating?: number; review?: string } = { bookingId, action };

      // Add rating data if it's a rating action
      if (rating !== undefined) {
        body.rating = rating;
      }
      if (review !== undefined) {
        body.review = review;
      }

      const response = await fetch(`${chatApiUrl}/booking-action`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${authStore.token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(body)
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to ${action} booking`);
      }

      // The WebSocket will receive the updated message automatically
      console.log(`✓ Booking action ${action} completed successfully`);
    } catch (error) {
      console.error(`Failed to ${action} booking:`, error);
      throw error;
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
    joinStoreConversationWithRetry,
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
    loadConversationsHttp,
    handleBookingAction
  };
});