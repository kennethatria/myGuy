# Browser Push Notifications Design
**Date:** January 3, 2026
**Status:** 📋 Design Document

---

## Overview

Browser push notifications allow the application to send notifications to users even when:
- The browser tab is in the background
- The browser is minimized
- The user is on a different tab
- The browser is closed (on some platforms)

This would significantly improve message notification delivery for item owners.

---

## How Browser Push Notifications Work

### Core Technologies

1. **Web Push API** - Browser API for sending push notifications
2. **Service Worker** - Background script that receives push messages
3. **Notification API** - Displays notifications to the user
4. **VAPID Keys** - Voluntary Application Server Identification for authentication

### The Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    1. INITIAL SETUP (One Time)                   │
└─────────────────────────────────────────────────────────────────┘

User opens app for first time
         │
         ▼
┌────────────────────────┐
│ Request Permission     │  ← Notification.requestPermission()
└───────────┬────────────┘
            │
            ▼
   ┌────────────────┐
   │ User Grants    │
   │ Permission     │
   └────────┬───────┘
            │
            ▼
┌────────────────────────────────────────────────────────────┐
│ Register Service Worker                                    │
│ navigator.serviceWorker.register('/sw.js')                │
└────────────────┬───────────────────────────────────────────┘
                 │
                 ▼
┌────────────────────────────────────────────────────────────┐
│ Subscribe to Push Service                                  │
│ swRegistration.pushManager.subscribe({...})               │
│                                                             │
│ Returns: Push Subscription Object                          │
│ {                                                           │
│   endpoint: "https://fcm.googleapis.com/fcm/send/...",    │
│   keys: {                                                   │
│     p256dh: "...",                                         │
│     auth: "..."                                            │
│   }                                                         │
│ }                                                           │
└────────────────┬───────────────────────────────────────────┘
                 │
                 ▼
┌────────────────────────────────────────────────────────────┐
│ Send Subscription to Backend                                │
│ POST /api/v1/push/subscribe                                │
│ Store in database: user_id + subscription object           │
└─────────────────────────────────────────────────────────────┘


┌─────────────────────────────────────────────────────────────────┐
│              2. WHEN NEW MESSAGE ARRIVES                         │
└─────────────────────────────────────────────────────────────────┘

Buyer sends message about item
         │
         ▼
┌────────────────────────────────────────────────────────────┐
│ Chat Service (Backend)                                      │
│ - Save message to database                                  │
│ - Emit WebSocket events (existing flow)                    │
└────────────────┬───────────────────────────────────────────┘
                 │
                 ▼
┌────────────────────────────────────────────────────────────┐
│ Check if Recipient is Online                                │
│ - Check if user has active WebSocket connection            │
└────────────────┬───────────────────────────────────────────┘
                 │
                 ▼
         ┌───────────────┐
         │ Is User       │
         │ Online?       │
         └───┬───────┬───┘
             │       │
        NO   │       │  YES
             │       │
             │       └──────────────────────────────────┐
             │                                          │
             ▼                                          ▼
┌─────────────────────────────┐         ┌──────────────────────────┐
│ Send Push Notification      │         │ Skip Push (user will get │
│                             │         │ WebSocket notification)  │
│ 1. Get user's push         │         └──────────────────────────┘
│    subscription from DB     │
│                             │
│ 2. Send push via Web Push   │
│    library (web-push npm)   │
│                             │
│ POST to subscription.endpoint│
│ with encrypted payload      │
└─────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ Push Service (Browser Vendor)                               │
│ - Chrome: Firebase Cloud Messaging (FCM)                    │
│ - Firefox: Mozilla Push Service                             │
│ - Safari: Apple Push Notification Service (APNS)            │
│                                                              │
│ Delivers push to user's device                              │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ Service Worker on User's Device                             │
│ self.addEventListener('push', event => {                    │
│   const data = event.data.json()                           │
│   self.registration.showNotification(data.title, {         │
│     body: data.body,                                       │
│     icon: data.icon,                                       │
│     badge: data.badge,                                     │
│     data: data.data                                        │
│   })                                                        │
│ })                                                          │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ Browser Notification Displayed                               │
│ ┌─────────────────────────────────────────────────────┐    │
│ │ 🔔 MyGuy                                            │    │
│ │ New message from JohnDoe                            │    │
│ │ "Is this camera still available?"                   │    │
│ └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                 │
                 ▼
         User clicks notification
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ Service Worker handles click                                │
│ self.addEventListener('notificationclick', event => {       │
│   event.notification.close()                               │
│   clients.openWindow('/messages/' + event.data.itemId)     │
│ })                                                          │
└─────────────────────────────────────────────────────────────┘
```

---

## Implementation Details

### 1. Frontend Implementation

#### **A. Service Worker Registration** (`public/sw.js`)

```javascript
// public/sw.js - Service Worker file

// Listen for push events
self.addEventListener('push', (event) => {
  console.log('Push received:', event);

  // Parse the push data
  const data = event.data ? event.data.json() : {};

  const title = data.title || 'New Message';
  const options = {
    body: data.body || 'You have a new message',
    icon: '/logo-192x192.png',
    badge: '/badge-72x72.png',
    tag: data.tag || 'message-notification',
    data: {
      url: data.url || '/messages',
      itemId: data.itemId,
      conversationId: data.conversationId,
      messageId: data.messageId
    },
    requireInteraction: false, // Auto-dismiss after ~20 seconds
    silent: false,
    vibrate: [200, 100, 200], // Vibration pattern for mobile
    actions: [
      {
        action: 'view',
        title: 'View Message',
        icon: '/icons/view.png'
      },
      {
        action: 'dismiss',
        title: 'Dismiss',
        icon: '/icons/dismiss.png'
      }
    ]
  };

  // Show the notification
  event.waitUntil(
    self.registration.showNotification(title, options)
  );
});

// Listen for notification clicks
self.addEventListener('notificationclick', (event) => {
  console.log('Notification clicked:', event);

  event.notification.close();

  // Handle action buttons
  if (event.action === 'view') {
    // Open the message
    const urlToOpen = event.notification.data.url;

    event.waitUntil(
      clients.matchAll({ type: 'window', includeUncontrolled: true })
        .then((clientList) => {
          // Check if app is already open
          for (const client of clientList) {
            if (client.url.includes('/messages') && 'focus' in client) {
              return client.focus();
            }
          }
          // Open new window if not already open
          if (clients.openWindow) {
            return clients.openWindow(urlToOpen);
          }
        })
    );
  } else if (event.action === 'dismiss') {
    // Just close the notification (already done above)
  } else {
    // Default click (no action button) - open the message
    const urlToOpen = event.notification.data.url;
    event.waitUntil(
      clients.openWindow(urlToOpen)
    );
  }
});

// Listen for push subscription changes
self.addEventListener('pushsubscriptionchange', (event) => {
  console.log('Push subscription changed:', event);

  event.waitUntil(
    // Resubscribe with new subscription
    self.registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array('YOUR_VAPID_PUBLIC_KEY')
    }).then((subscription) => {
      // Send new subscription to backend
      return fetch('/api/v1/push/subscribe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(subscription)
      });
    })
  );
});

// Utility function to convert VAPID key
function urlBase64ToUint8Array(base64String) {
  const padding = '='.repeat((4 - base64String.length % 4) % 4);
  const base64 = (base64String + padding)
    .replace(/-/g, '+')
    .replace(/_/g, '/');

  const rawData = atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}
```

#### **B. Push Notification Store** (`frontend/src/stores/pushNotifications.ts`)

```typescript
import { defineStore } from 'pinia';
import { ref } from 'vue';
import { useAuthStore } from './auth';
import config from '@/config';

export const usePushNotificationsStore = defineStore('pushNotifications', () => {
  const authStore = useAuthStore();

  const isSupported = ref(false);
  const isSubscribed = ref(false);
  const permission = ref<NotificationPermission>('default');
  const subscription = ref<PushSubscription | null>(null);

  // Check if push notifications are supported
  function checkSupport() {
    isSupported.value =
      'serviceWorker' in navigator &&
      'PushManager' in window &&
      'Notification' in window;

    if (isSupported.value) {
      permission.value = Notification.permission;
    }

    return isSupported.value;
  }

  // Request notification permission
  async function requestPermission(): Promise<boolean> {
    if (!isSupported.value) {
      console.warn('Push notifications not supported');
      return false;
    }

    if (permission.value === 'granted') {
      return true;
    }

    try {
      const result = await Notification.requestPermission();
      permission.value = result;

      if (result === 'granted') {
        console.log('✓ Notification permission granted');
        await subscribeToPush();
        return true;
      } else {
        console.warn('Notification permission denied');
        return false;
      }
    } catch (error) {
      console.error('Error requesting notification permission:', error);
      return false;
    }
  }

  // Subscribe to push notifications
  async function subscribeToPush(): Promise<PushSubscription | null> {
    if (!isSupported.value || permission.value !== 'granted') {
      console.warn('Cannot subscribe: not supported or permission not granted');
      return null;
    }

    try {
      // Register service worker
      const registration = await navigator.serviceWorker.register('/sw.js');
      console.log('Service worker registered');

      // Wait for service worker to be ready
      await navigator.serviceWorker.ready;

      // Get VAPID public key from backend
      const vapidResponse = await fetch(`${config.API_URL}/push/vapid-public-key`);
      const { publicKey } = await vapidResponse.json();

      // Subscribe to push notifications
      const pushSubscription = await registration.pushManager.subscribe({
        userVisibleOnly: true, // Required - all pushes must show a notification
        applicationServerKey: urlBase64ToUint8Array(publicKey)
      });

      console.log('Push subscription created:', pushSubscription);
      subscription.value = pushSubscription;

      // Send subscription to backend
      await sendSubscriptionToBackend(pushSubscription);

      isSubscribed.value = true;
      return pushSubscription;

    } catch (error) {
      console.error('Error subscribing to push:', error);
      return null;
    }
  }

  // Send subscription to backend
  async function sendSubscriptionToBackend(pushSubscription: PushSubscription) {
    try {
      const response = await fetch(`${config.API_URL}/push/subscribe`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${authStore.token}`
        },
        body: JSON.stringify({
          subscription: pushSubscription.toJSON(),
          deviceType: getDeviceType(),
          userAgent: navigator.userAgent
        })
      });

      if (!response.ok) {
        throw new Error('Failed to save subscription');
      }

      console.log('✓ Push subscription saved to backend');
    } catch (error) {
      console.error('Error sending subscription to backend:', error);
      throw error;
    }
  }

  // Unsubscribe from push notifications
  async function unsubscribe() {
    if (!subscription.value) return;

    try {
      // Unsubscribe from push service
      await subscription.value.unsubscribe();

      // Remove from backend
      await fetch(`${config.API_URL}/push/unsubscribe`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${authStore.token}`
        },
        body: JSON.stringify({
          endpoint: subscription.value.endpoint
        })
      });

      subscription.value = null;
      isSubscribed.value = false;

      console.log('✓ Unsubscribed from push notifications');
    } catch (error) {
      console.error('Error unsubscribing:', error);
    }
  }

  // Check current subscription status
  async function checkSubscription() {
    if (!isSupported.value) return;

    try {
      const registration = await navigator.serviceWorker.ready;
      const pushSubscription = await registration.pushManager.getSubscription();

      if (pushSubscription) {
        subscription.value = pushSubscription;
        isSubscribed.value = true;
        console.log('Existing push subscription found');
      } else {
        isSubscribed.value = false;
        console.log('No existing push subscription');
      }
    } catch (error) {
      console.error('Error checking subscription:', error);
    }
  }

  // Utility functions
  function urlBase64ToUint8Array(base64String: string): Uint8Array {
    const padding = '='.repeat((4 - base64String.length % 4) % 4);
    const base64 = (base64String + padding)
      .replace(/-/g, '+')
      .replace(/_/g, '/');

    const rawData = atob(base64);
    const outputArray = new Uint8Array(rawData.length);

    for (let i = 0; i < rawData.length; ++i) {
      outputArray[i] = rawData.charCodeAt(i);
    }
    return outputArray;
  }

  function getDeviceType(): string {
    const ua = navigator.userAgent;
    if (/mobile/i.test(ua)) return 'mobile';
    if (/tablet/i.test(ua)) return 'tablet';
    return 'desktop';
  }

  return {
    isSupported,
    isSubscribed,
    permission,
    subscription,
    checkSupport,
    requestPermission,
    subscribeToPush,
    unsubscribe,
    checkSubscription
  };
});
```

#### **C. UI Component for Permission** (`frontend/src/components/PushNotificationPrompt.vue`)

```vue
<template>
  <div v-if="showPrompt" class="push-prompt">
    <div class="push-prompt-content">
      <div class="push-prompt-icon">🔔</div>
      <h3>Enable Notifications</h3>
      <p>Get notified instantly when someone messages you about your items, even when the app is in the background.</p>

      <div class="push-prompt-actions">
        <button @click="enableNotifications" class="btn-primary">
          Enable Notifications
        </button>
        <button @click="dismiss" class="btn-secondary">
          Not Now
        </button>
      </div>

      <label class="push-prompt-checkbox">
        <input type="checkbox" v-model="dontShowAgain" />
        Don't show this again
      </label>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { usePushNotificationsStore } from '@/stores/pushNotifications';

const pushStore = usePushNotificationsStore();
const showPrompt = ref(false);
const dontShowAgain = ref(false);

onMounted(() => {
  // Check if we should show the prompt
  const dismissed = localStorage.getItem('push-prompt-dismissed');
  const hasPermission = pushStore.permission === 'granted';

  if (!dismissed && !hasPermission && pushStore.isSupported) {
    // Show prompt after 10 seconds (don't be annoying immediately)
    setTimeout(() => {
      showPrompt.value = true;
    }, 10000);
  }
});

async function enableNotifications() {
  const granted = await pushStore.requestPermission();

  if (granted) {
    showPrompt.value = false;
    // Show success message
    alert('✓ Notifications enabled! You\'ll now receive alerts for new messages.');
  }
}

function dismiss() {
  showPrompt.value = false;

  if (dontShowAgain.value) {
    localStorage.setItem('push-prompt-dismissed', 'true');
  }
}
</script>

<style scoped>
.push-prompt {
  position: fixed;
  bottom: 20px;
  right: 20px;
  max-width: 400px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  padding: 24px;
  z-index: 1000;
  animation: slideIn 0.3s ease-out;
}

@keyframes slideIn {
  from {
    transform: translateY(100px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.push-prompt-icon {
  font-size: 48px;
  text-align: center;
  margin-bottom: 16px;
}

.push-prompt h3 {
  margin: 0 0 8px 0;
  font-size: 20px;
}

.push-prompt p {
  margin: 0 0 20px 0;
  color: #666;
  line-height: 1.5;
}

.push-prompt-actions {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.btn-primary, .btn-secondary {
  flex: 1;
  padding: 12px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
}

.btn-primary {
  background: #007bff;
  color: white;
}

.btn-primary:hover {
  background: #0056b3;
}

.btn-secondary {
  background: #f0f0f0;
  color: #333;
}

.btn-secondary:hover {
  background: #e0e0e0;
}

.push-prompt-checkbox {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #666;
  cursor: pointer;
}
</style>
```

---

### 2. Backend Implementation

#### **A. Database Schema**

```sql
-- Add to main backend database (my_guy)

CREATE TABLE push_subscriptions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL UNIQUE,
    p256dh_key TEXT NOT NULL,
    auth_key TEXT NOT NULL,
    device_type VARCHAR(20) DEFAULT 'desktop',
    user_agent TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP
);

CREATE INDEX idx_push_subscriptions_user_id ON push_subscriptions(user_id);
CREATE INDEX idx_push_subscriptions_is_active ON push_subscriptions(is_active);
CREATE INDEX idx_push_subscriptions_endpoint ON push_subscriptions(endpoint);

-- Track notification delivery
CREATE TABLE push_notification_log (
    id SERIAL PRIMARY KEY,
    subscription_id INTEGER REFERENCES push_subscriptions(id) ON DELETE SET NULL,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message_id INTEGER,
    notification_type VARCHAR(50) NOT NULL,
    payload JSONB,
    status VARCHAR(20) NOT NULL, -- 'sent', 'failed', 'expired'
    error_message TEXT,
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_push_notification_log_user_id ON push_notification_log(user_id);
CREATE INDEX idx_push_notification_log_sent_at ON push_notification_log(sent_at);
```

#### **B. Push Service (Node.js)** (`chat-websocket-service/src/services/pushService.js`)

```javascript
const webPush = require('web-push');
const db = require('../config/database');
const logger = require('../utils/logger');

class PushService {
  constructor() {
    // Generate VAPID keys once: npx web-push generate-vapid-keys
    // Store in environment variables
    const vapidPublicKey = process.env.VAPID_PUBLIC_KEY;
    const vapidPrivateKey = process.env.VAPID_PRIVATE_KEY;
    const vapidSubject = process.env.VAPID_SUBJECT || 'mailto:support@myguy.com';

    if (!vapidPublicKey || !vapidPrivateKey) {
      logger.error('VAPID keys not configured');
      return;
    }

    webPush.setVapidDetails(
      vapidSubject,
      vapidPublicKey,
      vapidPrivateKey
    );

    logger.info('Push service initialized');
  }

  /**
   * Save push subscription for a user
   */
  async saveSubscription(userId, subscription, deviceType = 'desktop', userAgent = '') {
    try {
      const { endpoint, keys } = subscription;

      const query = `
        INSERT INTO push_subscriptions (user_id, endpoint, p256dh_key, auth_key, device_type, user_agent)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (endpoint)
        DO UPDATE SET
          user_id = EXCLUDED.user_id,
          p256dh_key = EXCLUDED.p256dh_key,
          auth_key = EXCLUDED.auth_key,
          is_active = TRUE,
          updated_at = CURRENT_TIMESTAMP
        RETURNING id
      `;

      const result = await db.query(query, [
        userId,
        endpoint,
        keys.p256dh,
        keys.auth,
        deviceType,
        userAgent
      ]);

      logger.info('Push subscription saved', { userId, subscriptionId: result.rows[0].id });
      return result.rows[0].id;

    } catch (error) {
      logger.error('Error saving push subscription:', error);
      throw error;
    }
  }

  /**
   * Remove push subscription
   */
  async removeSubscription(endpoint) {
    try {
      const query = `
        UPDATE push_subscriptions
        SET is_active = FALSE, updated_at = CURRENT_TIMESTAMP
        WHERE endpoint = $1
      `;

      await db.query(query, [endpoint]);
      logger.info('Push subscription removed', { endpoint });

    } catch (error) {
      logger.error('Error removing subscription:', error);
      throw error;
    }
  }

  /**
   * Get all active subscriptions for a user
   */
  async getUserSubscriptions(userId) {
    try {
      const query = `
        SELECT id, endpoint, p256dh_key, auth_key
        FROM push_subscriptions
        WHERE user_id = $1 AND is_active = TRUE
      `;

      const result = await db.query(query, [userId]);
      return result.rows;

    } catch (error) {
      logger.error('Error getting user subscriptions:', error);
      return [];
    }
  }

  /**
   * Send push notification to user
   */
  async sendNotificationToUser(userId, payload) {
    try {
      const subscriptions = await this.getUserSubscriptions(userId);

      if (subscriptions.length === 0) {
        logger.info('No push subscriptions for user', { userId });
        return { sent: 0, failed: 0 };
      }

      logger.info(`Sending push to ${subscriptions.length} subscription(s)`, { userId });

      const results = await Promise.allSettled(
        subscriptions.map(sub => this.sendPush(sub, payload))
      );

      const sent = results.filter(r => r.status === 'fulfilled').length;
      const failed = results.filter(r => r.status === 'rejected').length;

      logger.info('Push notifications sent', { userId, sent, failed });

      return { sent, failed };

    } catch (error) {
      logger.error('Error sending push notification:', error);
      return { sent: 0, failed: 1 };
    }
  }

  /**
   * Send push to a single subscription
   */
  async sendPush(subscription, payload) {
    const pushSubscription = {
      endpoint: subscription.endpoint,
      keys: {
        p256dh: subscription.p256dh_key,
        auth: subscription.auth_key
      }
    };

    const pushPayload = JSON.stringify(payload);

    const options = {
      TTL: 24 * 60 * 60, // 24 hours
      urgency: 'normal'
    };

    try {
      await webPush.sendNotification(pushSubscription, pushPayload, options);

      // Update last_used_at
      await db.query(
        'UPDATE push_subscriptions SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1',
        [subscription.id]
      );

      // Log success
      await this.logNotification(subscription.id, payload, 'sent');

      logger.info('Push sent successfully', { subscriptionId: subscription.id });

    } catch (error) {
      logger.error('Error sending push:', error);

      // Handle expired subscriptions
      if (error.statusCode === 410) {
        logger.warn('Subscription expired, removing', { subscriptionId: subscription.id });
        await this.removeSubscription(subscription.endpoint);
      }

      // Log failure
      await this.logNotification(subscription.id, payload, 'failed', error.message);

      throw error;
    }
  }

  /**
   * Log notification delivery
   */
  async logNotification(subscriptionId, payload, status, errorMessage = null) {
    try {
      const query = `
        INSERT INTO push_notification_log (subscription_id, user_id, message_id, notification_type, payload, status, error_message)
        SELECT $1, user_id, $2, $3, $4, $5, $6
        FROM push_subscriptions
        WHERE id = $1
      `;

      await db.query(query, [
        subscriptionId,
        payload.messageId || null,
        payload.type || 'message',
        JSON.stringify(payload),
        status,
        errorMessage
      ]);

    } catch (error) {
      logger.error('Error logging notification:', error);
    }
  }

  /**
   * Send message notification
   */
  async sendMessageNotification(userId, message, itemTitle) {
    const payload = {
      type: 'message',
      messageId: message.id,
      itemId: message.store_item_id,
      conversationId: message.task_id || message.application_id || message.store_item_id,
      title: 'New Message',
      body: `${message.sender?.username || 'Someone'} messaged you about "${itemTitle}"`,
      icon: '/logo-192x192.png',
      badge: '/badge-72x72.png',
      tag: `message-${message.id}`,
      url: `/messages/${message.store_item_id || message.task_id || message.application_id}`,
      timestamp: new Date().getTime()
    };

    return await this.sendNotificationToUser(userId, payload);
  }
}

module.exports = new PushService();
```

#### **C. API Endpoints** (`chat-websocket-service/src/server.js`)

```javascript
const pushService = require('./services/pushService');

// Get VAPID public key
app.get('/api/v1/push/vapid-public-key', (req, res) => {
  res.json({
    publicKey: process.env.VAPID_PUBLIC_KEY
  });
});

// Subscribe to push notifications
app.post('/api/v1/push/subscribe', authenticateHTTP, async (req, res) => {
  try {
    const { subscription, deviceType, userAgent } = req.body;
    const userId = req.userId;

    const subscriptionId = await pushService.saveSubscription(
      userId,
      subscription,
      deviceType,
      userAgent
    );

    res.json({
      success: true,
      subscriptionId
    });

  } catch (error) {
    logger.error('Error subscribing to push:', error);
    res.status(500).json({ error: 'Failed to subscribe' });
  }
});

// Unsubscribe from push notifications
app.post('/api/v1/push/unsubscribe', authenticateHTTP, async (req, res) => {
  try {
    const { endpoint } = req.body;

    await pushService.removeSubscription(endpoint);

    res.json({ success: true });

  } catch (error) {
    logger.error('Error unsubscribing from push:', error);
    res.status(500).json({ error: 'Failed to unsubscribe' });
  }
});

// Test push notification
app.post('/api/v1/push/test', authenticateHTTP, async (req, res) => {
  try {
    const userId = req.userId;

    const result = await pushService.sendNotificationToUser(userId, {
      title: 'Test Notification',
      body: 'This is a test push notification from MyGuy',
      icon: '/logo-192x192.png',
      url: '/messages'
    });

    res.json({ success: true, result });

  } catch (error) {
    logger.error('Error sending test push:', error);
    res.status(500).json({ error: 'Failed to send test notification' });
  }
});
```

#### **D. Integration with Message Handler** (`chat-websocket-service/src/handlers/socketHandlers.js`)

```javascript
const pushService = require('../services/pushService');

// In handleSendMessage function, after line 222:

async handleSendMessage(socket, data) {
  try {
    // ... existing message sending code ...

    // Emit to recipient's personal room (for notifications)
    this.io.to(`user:${recipientId}`).emit('message:notification', {
      message: formattedMessage,
      conversationId: taskId || applicationId || itemId
    });

    // NEW: Send push notification if recipient is offline
    const recipientSockets = this.userSockets.get(recipientId);
    const isRecipientOnline = recipientSockets && recipientSockets.size > 0;

    if (!isRecipientOnline) {
      logger.info('Recipient offline, sending push notification', { recipientId });

      // Get item/task title for better notification
      const itemTitle = await this.getContextTitle(taskId, applicationId, itemId);

      // Send push notification
      await pushService.sendMessageNotification(
        recipientId,
        formattedMessage,
        itemTitle
      );
    } else {
      logger.info('Recipient online, skipping push notification', { recipientId });
    }

    // ... rest of existing code ...
  } catch (error) {
    logger.error('Error sending message:', error);
    socket.emit('error', { message: 'Failed to send message' });
  }
}

// Helper to get context title
async getContextTitle(taskId, applicationId, itemId) {
  try {
    if (itemId) {
      // Query store_items table
      const result = await db.query('SELECT title FROM store_items WHERE id = $1', [itemId]);
      return result.rows[0]?.title || 'Store Item';
    } else if (taskId) {
      // Query tasks table
      const result = await db.query('SELECT title FROM tasks WHERE id = $1', [taskId]);
      return result.rows[0]?.title || 'Task';
    }
    return 'Conversation';
  } catch (error) {
    logger.error('Error getting context title:', error);
    return 'Conversation';
  }
}
```

---

### 3. Installation & Setup

#### **A. Install Dependencies**

```bash
cd chat-websocket-service
npm install web-push --save
```

#### **B. Generate VAPID Keys**

```bash
npx web-push generate-vapid-keys
```

Output:
```
=======================================
Public Key:
BEl62iUYgUivxIkv69yViEuiBIa-Ib9-SkvMeAtA3LFgDzkrxZJjSgSnfckjBJuBkr3qBUYIHBQFLXYp5Nksh8U

Private Key:
UUxI4O8-FXScqWEL_Zs0W9CRFAGJNXpBnLPb0ij9JAc
=======================================
```

#### **C. Update Environment Variables**

```bash
# chat-websocket-service/.env
VAPID_PUBLIC_KEY=BEl62iUYgUivxIkv69yViEuiBIa-Ib9-SkvMeAtA3LFgDzkrxZJjSgSnfckjBJuBkr3qBUYIHBQFLXYp5Nksh8U
VAPID_PRIVATE_KEY=UUxI4O8-FXScqWEL_Zs0W9CRFAGJNXpBnLPb0ij9JAc
VAPID_SUBJECT=mailto:support@myguy.com
```

#### **D. Run Database Migrations**

```sql
-- Run the push_subscriptions table creation SQL
psql -U postgres -d my_guy -f migrations/003_push_notifications.sql
```

---

## Browser Support

| Browser | Support | Notes |
|---------|---------|-------|
| Chrome 50+ | ✅ Full | Uses Firebase Cloud Messaging (FCM) |
| Firefox 44+ | ✅ Full | Uses Mozilla Push Service |
| Safari 16+ | ✅ Full | iOS 16.4+ required for web push |
| Edge 79+ | ✅ Full | Chromium-based, uses FCM |
| Opera 37+ | ✅ Full | Chromium-based, uses FCM |
| Safari < 16 | ❌ No | No web push support |
| IE 11 | ❌ No | Not supported |

---

## User Flow Example

### First Time User:
1. User logs in to MyGuy
2. After 10 seconds, prompt appears: "Enable notifications?"
3. User clicks "Enable Notifications"
4. Browser shows native permission dialog
5. User clicks "Allow"
6. Service worker subscribes to push service
7. Subscription saved to database
8. ✅ User is now subscribed

### When Message Arrives:
1. Buyer sends message about item owner's camera
2. Chat service saves message
3. **Checks if item owner is online**
   - **If online:** WebSocket notification only (no push)
   - **If offline:** Send push notification
4. Push service sends notification to browser vendor
5. Browser vendor delivers to user's device
6. **Notification appears on screen** (even if browser closed on some platforms)
7. User clicks notification
8. Browser/tab opens to message conversation
9. ✅ User can respond immediately

---

## Security & Privacy

### 1. **VAPID Authentication**
- Identifies your server to push services
- Prevents unauthorized push sending
- Required by all modern browsers

### 2. **End-to-End Encryption**
- Push payload is encrypted by web-push library
- Only user's browser can decrypt
- Uses p256dh and auth keys

### 3. **User Consent Required**
- Users must explicitly grant permission
- Can revoke permission at any time
- Respects browser notification settings

### 4. **Subscription Security**
- Endpoint URLs are unique per user/device
- Subscriptions can expire (handled automatically)
- Inactive subscriptions cleaned up (410 status)

---

## Advantages

✅ **Works when tab is in background**
✅ **Works when browser is minimized**
✅ **Works on mobile devices**
✅ **Native OS notifications**
✅ **No third-party service required** (uses browser vendors)
✅ **Free** (no cost for push delivery)
✅ **Fast delivery** (milliseconds)
✅ **Reliable** (handled by browser vendors)
✅ **Offline capable** (queued until device is online)

---

## Limitations

❌ **Requires HTTPS** (except localhost for development)
❌ **User must grant permission** (can be denied)
❌ **Not supported on Safari < 16** (older iOS/macOS)
❌ **Can't send push if user denies permission**
❌ **Limited payload size** (4KB max)
❌ **No guarantee of delivery** (best-effort)
❌ **Users can disable at OS level** (you can't override)

---

## Best Practices

### 1. **Don't Ask Immediately**
- Wait for user engagement (after 10+ seconds)
- Explain the benefit before asking
- Provide "Not now" option

### 2. **Provide Settings**
- Let users enable/disable in settings
- Allow granular control (e.g., only important messages)
- Easy unsubscribe option

### 3. **Smart Notification Logic**
- Only send push if user is offline
- Don't spam with every message
- Combine multiple messages into one notification if rapid

### 4. **Graceful Degradation**
- App works without notifications
- Fallback to in-app indicators
- Clear messaging about notification status

### 5. **Handle Errors**
- Clean up expired subscriptions (410 status)
- Retry failed deliveries
- Log delivery status for debugging

---

## Cost Analysis

**Infrastructure Cost:** $0/month
- Uses browser vendor push services (free)
- No external service fees (unlike FCM for mobile apps)
- Just hosting costs for your backend

**Development Cost:** ~8-16 hours
- Frontend: Service worker + UI (~4-6 hours)
- Backend: Push service + API (~4-6 hours)
- Testing: Cross-browser testing (~2-4 hours)

**Maintenance Cost:** Minimal
- Automatic subscription cleanup
- Occasional VAPID key rotation (yearly)
- Monitor delivery logs

---

## Testing

### Local Testing:
```bash
# 1. Start backend with VAPID keys
cd chat-websocket-service
npm start

# 2. Access via HTTPS or localhost
# localhost:3000 ✅ (notifications work on localhost)
# https://yourdomain.com ✅
# http://yourdomain.com ❌ (won't work - needs HTTPS)

# 3. Test subscription
# Open browser DevTools > Application > Service Workers
# Open browser DevTools > Application > Push Messaging

# 4. Send test notification
POST http://localhost:8082/api/v1/push/test
Authorization: Bearer YOUR_TOKEN
```

### Production Testing:
```bash
# Use real HTTPS domain
# Test on multiple browsers
# Test on mobile devices
# Test with browser minimized
# Test with tab in background
```

---

## Rollout Strategy

### Phase 1: Soft Launch (Week 1)
- Deploy to staging
- Test with internal users
- Monitor logs and delivery rates

### Phase 2: Beta Testing (Week 2-3)
- Enable for 10% of users
- Gather feedback
- Fix issues

### Phase 3: Full Rollout (Week 4)
- Enable for all users
- Monitor delivery metrics
- Iterate on messaging

---

## Success Metrics

**Measure:**
- Subscription rate (% of users who enable)
- Delivery success rate (% of pushes delivered)
- Click-through rate (% who click notification)
- Conversion rate (% who respond to message after notification)
- Opt-out rate (% who disable notifications)

**Target Goals:**
- 60%+ subscription rate
- 95%+ delivery success rate
- 40%+ click-through rate
- 20%+ conversion rate
- <5% opt-out rate

---

## Conclusion

Browser push notifications would significantly improve the notification system by:

1. **Reaching offline users** - No need to have app open
2. **Background notifications** - Works even when tab is inactive
3. **Native OS integration** - Professional notification appearance
4. **No cost** - Uses browser vendor services for free
5. **Easy implementation** - Well-documented APIs and libraries

Combined with existing WebSocket notifications for online users, this creates a comprehensive notification system that ensures item owners never miss a potential sale.

---

**Document Version:** 1.0
**Last Updated:** January 3, 2026
**Status:** Ready for Implementation
