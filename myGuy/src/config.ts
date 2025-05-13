// API configuration
const API_URL = 'http://localhost:8080/api/v1';

export default {
  API_URL,
  ENDPOINTS: {
    LOGIN: `${API_URL}/login`,
    REGISTER: `${API_URL}/register`,
    PROFILE: `${API_URL}/profile`,
    TASKS: `${API_URL}/tasks`,
  }
};