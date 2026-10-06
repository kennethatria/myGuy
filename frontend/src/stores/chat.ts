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

type ConversationRef = {
  task_id?: number | null;
  application_id?: number | null;
  item_id?: number | null;
  other_user_id?: number | null;
};

/**
 * Stable identity of a conversation: its context plus the other participant.
 * Task, application and store item ids overlap, and a store item can have
 * several buyers, so a bare id is never enough to tell conversations apart.
 */
export function conversationKey(c: ConversationRef): string | null {
  const other = c.other_user_id ?? '';
  if (c.task_id) return `task:${c.task_id}:${other}`;
  if (c.application_id) return `application:${c.application_id}:${other}`;
  if (c.item_id) return `store:${c.item_id}:${other}`;
  return null;
}

// Context ids as the chat socket API expects them.
function contextParams(c: ConversationRef) {
  if (c.task_id) return { taskId: c.task_id };
  if (c.application_id) return { applicationId: c.application_id };
  return { itemId: c.item_id ?? undefined };
}

function conversationParams(c: ConversationRef) {
  return { ...contextParams(c), otherUserId: c.other_user_id ?? undefined };
}

const PAGE_SIZE = 20;

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
  // All maps below are keyed by conversationKey().
  const messages = ref<Map<string, Message[]>>(new Map());
  const typingUsers = ref<Map<string, TypingUser[]>>(new Map());
  const isLoadingMessages = ref(false);
  const hasMoreMessages = ref<Map<string, boolean>>(new Map());
  const totalMessageCounts = ref<Map<string, number>>(new Map());
  const deletionWarnings = ref<{ id: number; task_id: number; task_title: string; deletion_scheduled_at: string }[]>([]);

  // Helpers
  const myId = () => authStore.user?.id;

  function messageConversation(message: Message): ConversationRef {
    return {
      task_id: message.task_id,
      application_id: message.application_id,
      item_id: message.store_item_id ?? message.item_id,
      other_user_id: message.sender_id === myId() ? message.recipient_id : message.sender_id
    };
  }

  function findConversation(key: string | null) {
    return key ? conversations.value.find(c => conversationKey(c) === key) : undefined;
  }

  // Computed
  const activeKey = computed(() => activeConversation.value ? conversationKey(activeConversation.value) : null);

  const totalUnreadCount = computed(() =>
    conversations.value.reduce((total, conv) => total + (Number(conv.unread_count) || 0), 0)
  );

  const activeMessages = computed(() => (activeKey.value && messages.value.get(activeKey.value)) || []);

  const activeTypingUsers = computed(() => (activeKey.value && typingUsers.value.get(activeKey.value)) || []);

  const activeHasMoreMessages = computed(() => !!activeKey.value && hasMoreMessages.value.get(activeKey.value) === true);

  // Socket connection
  // Store message methods
  // Opens a store chat about itemId. Buyers have one chat per item (with the
  // seller); a seller has one per buyer, so a seller's link must name the
  // buyer (otherUserId) — without it nothing is opened rather than a guess.
  async function joinStoreConversation(itemId: number, otherUserId?: number) {
    if (!socket.value?.connected) await connectSocket();

    // Wait briefly for conversations to load if empty
    if (conversations.value.length === 0) {
      await new Promise(resolve => setTimeout(resolve, 500));
    }

    if (otherUserId) {
      const existing = findConversation(conversationKey({ item_id: itemId, other_user_id: otherUserId }));
      if (existing) {
        openConversation(existing);
        return;
      }
    } else {
      const matches = conversations.value.filter(c => c.item_id === itemId);
      if (matches.length === 1) {
        openConversation(matches[0]!);
        return;
      }
      if (matches.length > 1) return; // seller with several buyers: let them choose
    }

    // No conversation yet: start one with the seller (buyer side), or with the
    // named user.
    try {
      const response = await fetch(`${config.STORE_API_URL}/items/${itemId}`, {
        headers: { Authorization: `Bearer ${authStore.token}` }
      });
      if (!response.ok) return;
      const item = await response.json();

      const other = otherUserId ?? item.seller_id;
      if (!other || other === myId()) return; // a seller can't start a chat with themselves

      conversations.value.push({
        item_id: itemId,
        item_title: item.title,
        last_message: '',
        last_message_time: new Date().toISOString(),
        other_user_id: other,
        other_user_name: other === item.seller_id
          ? (item.seller?.name || item.seller?.username || 'Seller')
          : 'Buyer',
        unread_count: 0,
        conversation_type: 'store'
      });
      const conv = findConversation(conversationKey({ item_id: itemId, other_user_id: other }));
      if (conv) {
        openConversation(conv);
        if (other !== item.seller_id) enrichConversations();
      }
    } catch (error) {
      console.error('Failed to create conversation placeholder:', error);
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

  // Messages of the open chat for itemId, or of the first chat about it.
  function getStoreMessages(itemId: number): Message[] {
    if (activeConversation.value?.item_id === itemId) return activeMessages.value;
    for (const [key, list] of messages.value) {
      if (key.startsWith(`store:${itemId}:`)) return list;
    }
    return [];
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
    
    // Message events. The server delivers these only to the two participants.
    socket.value.on('message:new', receiveMessage);
    socket.value.on('message:sent', receiveMessage);
    socket.value.on('message:edited', handleMessageEdited);
    socket.value.on('message:updated', handleMessageUpdated);
    socket.value.on('message:deleted', handleMessageDeleted);
    socket.value.on('message:read', handleMessageRead);
    socket.value.on('message:filtered', handleMessageFiltered);
    
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
  function receiveMessage(message: Message) {
    const ref = messageConversation(message);
    const key = conversationKey(ref);
    if (!key) return;

    // Append only to an already-loaded thread; an unloaded one fetches its
    // full history when opened. The same message can arrive twice (e.g.
    // message:sent plus another tab), so de-duplicate by id.
    const loaded = messages.value.get(key);
    if (loaded && !loaded.some(m => m.id === message.id)) {
      enrichMessages([message]);
      messages.value.set(key, [...loaded, message]);
      totalMessageCounts.value.set(key, (totalMessageCounts.value.get(key) || 0) + 1);
    }

    let conv = findConversation(key);
    if (!conv) {
      conversations.value.push({
        task_id: ref.task_id ?? undefined,
        application_id: ref.application_id ?? undefined,
        item_id: ref.item_id ?? undefined,
        last_message: message.content,
        last_message_time: message.created_at,
        other_user_id: ref.other_user_id!,
        other_user_name: 'Unknown User', // Replaced by enrichConversations
        unread_count: 0,
        conversation_type: message.task_id ? 'task' : message.application_id ? 'application' : 'store'
      });
      conv = findConversation(key)!;
      enrichConversations();
    }

    conv.last_message = message.content;
    conv.last_message_type = message.message_type;
    conv.last_message_time = message.created_at;

    if (message.sender_id !== myId()) {
      if (key === activeKey.value) {
        // Being read right now: tell the server so it isn't unread on reload.
        socket.value?.emit('conversation:read', conversationParams(conv));
      } else {
        conv.unread_count = (Number(conv.unread_count) || 0) + 1;
      }
    }

    // Sort conversations by last message time
    conversations.value.sort((a, b) =>
      new Date(b.last_message_time).getTime() - new Date(a.last_message_time).getTime()
    );
  }

  // Apply a change to a message wherever it is loaded.
  function patchMessage(messageId: number, patch: Partial<Message>) {
    messages.value.forEach((list, key) => {
      const index = list.findIndex(m => m.id === messageId);
      if (index !== -1) {
        const next = [...list];
        next[index] = { ...next[index], ...patch };
        messages.value.set(key, next);
      }
    });
  }

  function handleMessageEdited(message: Message) {
    patchMessage(message.id, message);
  }

  function handleMessageUpdated(message: Message) {
    // Booking status updates and other message updates
    patchMessage(message.id, message);
  }

  function handleMessageDeleted({ messageId }: { messageId: number }) {
    patchMessage(messageId, { content: '[Message deleted]', is_deleted: true });
  }

  function handleMessageRead({ messageId, readAt }: { messageId: number; readAt: string }) {
    patchMessage(messageId, { is_read: true, read_at: readAt });
  }

  function handleMessageFiltered({ warning }: { messageId?: number; warning: string }) {
    // Show warning to user
    alert(warning);
  }

  function handleConversationsList(convs: ConversationSummary[]) {
    conversations.value = convs.map(conv => ({ ...conv, unread_count: Number(conv.unread_count) || 0 }));

    // Keep the open conversation pointing at the refreshed object; one that
    // has no messages yet isn't in the server list, so keep it listed.
    const key = activeKey.value;
    if (key && activeConversation.value) {
      const refreshed = findConversation(key);
      if (refreshed) {
        activeConversation.value = refreshed;
      } else {
        conversations.value.push(activeConversation.value);
      }
    }

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
  
  function handleMessagesList({ taskId, applicationId, itemId, otherUserId, messages: msgs, offset, totalCount }: { taskId?: number; applicationId?: number; itemId?: number; otherUserId?: number; messages: Message[]; offset: number; totalCount?: number }) {
    const key = conversationKey({
      task_id: taskId ? Number(taskId) : undefined,
      application_id: applicationId ? Number(applicationId) : undefined,
      item_id: itemId ? Number(itemId) : undefined,
      other_user_id: otherUserId ? Number(otherUserId) : activeConversation.value?.other_user_id
    });
    if (!key) return;

    // Enrich messages with sender/recipient data
    enrichMessages(msgs);

    if (offset === 0) {
      messages.value.set(key, msgs);
    } else {
      // Prepend older messages, skipping any already present
      const existing = messages.value.get(key) || [];
      const existingIds = new Set(existing.map(m => m.id));
      messages.value.set(key, [...msgs.filter(m => !existingIds.has(m.id)), ...existing]);
    }

    if (totalCount !== undefined) {
      totalMessageCounts.value.set(key, totalCount);
    }

    // A full page means there may be older messages
    hasMoreMessages.value.set(key, msgs.length === PAGE_SIZE);
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
  
  function handleConversationMarkedRead({ taskId, applicationId, itemId, otherUserId }: { taskId?: number; applicationId?: number; itemId?: number; otherUserId?: number; count?: number }) {
    const conv = findConversation(conversationKey({
      task_id: taskId, application_id: applicationId, item_id: itemId,
      other_user_id: otherUserId ?? activeConversation.value?.other_user_id
    }));
    if (conv) conv.unread_count = 0;
  }

  // The typer is the other participant from this user's point of view.
  function typingKey({ userId, taskId, applicationId, itemId }: { userId: number; taskId?: number; applicationId?: number; itemId?: number }) {
    return conversationKey({ task_id: taskId, application_id: applicationId, item_id: itemId, other_user_id: userId });
  }

  function handleUserTyping(event: { userId: number; userName: string; taskId?: number; applicationId?: number; itemId?: number }) {
    if (event.userId === myId()) return;
    const key = typingKey(event);
    if (!key) return;

    const users = typingUsers.value.get(key) || [];
    if (!users.find(u => u.userId === event.userId)) {
      typingUsers.value.set(key, [...users, { userId: event.userId, userName: event.userName }]);
    }

    // Remove after 3 seconds
    setTimeout(() => handleUserStoppedTyping(event), 3000);
  }

  function handleUserStoppedTyping(event: { userId: number; taskId?: number; applicationId?: number; itemId?: number }) {
    const key = typingKey(event);
    if (!key) return;
    const users = typingUsers.value.get(key) || [];
    typingUsers.value.set(key, users.filter(u => u.userId !== event.userId));
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
  function openConversation(conv: ConversationSummary) {
    if (!socket.value) return;
    const key = conversationKey(conv);
    if (!key) return;

    const previous = activeConversation.value;
    if (previous && conversationKey(previous) !== key) {
      socket.value.emit('leave:conversation', contextParams(previous));
    }

    activeConversation.value = conv;
    socket.value.emit('join:conversation', contextParams(conv));

    if (!messages.value.has(key)) {
      isLoadingMessages.value = true;
      socket.value.emit('messages:get', { ...conversationParams(conv), limit: PAGE_SIZE, offset: 0 });
    }

    if (Number(conv.unread_count) > 0) {
      socket.value.emit('conversation:read', conversationParams(conv));
    }
  }

  // Opens the conversation about a task or application with one specific
  // person, creating it if it has no messages yet, so the first message can
  // be sent and the right thread shows even before the conversation list has
  // loaded or when several people have written about the same task.
  function openConversationWith(target: {
    taskId?: number;
    applicationId?: number;
    otherUserId: number;
    otherUserName?: string;
  }) {
    const ref: ConversationRef = {
      task_id: target.taskId,
      application_id: target.applicationId,
      other_user_id: target.otherUserId
    };
    const key = conversationKey(ref);
    if (!key) return;

    let conv = findConversation(key);
    if (!conv) {
      conversations.value.push({
        task_id: target.taskId,
        application_id: target.applicationId,
        last_message: '',
        last_message_time: new Date().toISOString(),
        other_user_id: target.otherUserId,
        other_user_name: target.otherUserName || 'User',
        unread_count: 0,
        conversation_type: target.taskId ? 'task' : 'application'
      });
      conv = findConversation(key)!;
    }
    openConversation(conv);
  }

  // Accepts a conversation, or a bare task/application/item id (e.g. from a
  // URL), which resolves to the first matching conversation.
  function joinConversation(target: ConversationSummary | number) {
    const conv = typeof target === 'number'
      ? conversations.value.find(c => c.task_id === target)
        ?? conversations.value.find(c => c.application_id === target)
        ?? conversations.value.find(c => c.item_id === target)
      : target;
    if (conv) openConversation(conv);
  }

  function sendMessage(content: string, recipientId: number) {
    if (!socket.value || !activeConversation.value) return;

    socket.value.emit('message:send', {
      ...contextParams(activeConversation.value),
      recipientId,
      content
    });
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
    const conv = activeConversation.value;
    if (!socket.value || !conv || isLoadingMessages.value) return;
    const key = conversationKey(conv);
    if (!key || !hasMoreMessages.value.get(key)) return;

    isLoadingMessages.value = true;
    socket.value.emit('messages:get', {
      ...conversationParams(conv),
      limit: PAGE_SIZE,
      offset: (messages.value.get(key) || []).length
    });
  }

  function startTyping() {
    const conv = activeConversation.value;
    if (!socket.value || !conv) return;
    socket.value.emit('typing:start', { ...contextParams(conv), recipientId: conv.other_user_id });
  }

  function stopTyping() {
    const conv = activeConversation.value;
    if (!socket.value || !conv) return;
    socket.value.emit('typing:stop', { ...contextParams(conv), recipientId: conv.other_user_id });
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
    typingUsers,
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
    openConversationWith,
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