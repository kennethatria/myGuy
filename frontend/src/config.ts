// API configuration
const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';
const STORE_API_URL = import.meta.env.VITE_STORE_API_URL || 'http://localhost:8081/api/v1';
const STORE_API_BASE_URL = import.meta.env.VITE_STORE_API_BASE_URL || 'http://localhost:8081';
const CHAT_API_URL = import.meta.env.VITE_CHAT_API_URL || 'http://localhost:8082/api/v1';
const CHAT_WS_URL = import.meta.env.VITE_CHAT_WS_URL || 'http://localhost:8082';

export default {
  API_URL,
  STORE_API_URL,
  STORE_API_BASE_URL,
  CHAT_API_URL,
  CHAT_WS_URL,
  ENDPOINTS: {
    LOGIN: `${API_URL}/login`,
    REGISTER: `${API_URL}/register`,
    PROFILE: `${API_URL}/profile`,
    TASKS: `${API_URL}/tasks`,
    APPLICATIONS: `${API_URL}/applications`,
    USERS: `${API_URL}/users`,
    // Chat endpoints
    TASK_MESSAGES: `${CHAT_API_URL}/tasks`,
    APPLICATION_MESSAGES: `${CHAT_API_URL}/applications`,
    STORE_MESSAGES: `${CHAT_API_URL}/store-messages`,
  }
};