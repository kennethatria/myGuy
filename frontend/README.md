# Frontend Application

This directory contains the frontend application for the MyGuy platform, built as a modern single-page application (SPA).

## Technology Stack

-   **Framework**: [Vue.js 3](https://vuejs.org/) with the Composition API
-   **Language**: TypeScript
-   **Build Tool**: [Vite](https://vitejs.dev/)
-   **Routing**: [Vue Router](https://router.vuejs.org/)
-   **State Management**: [Pinia](https://pinia.vuejs.org/)
-   **Styling**: Standard CSS with a responsive design approach.

## Project Structure

```
frontend/
├── public/              # Static assets
├── src/
│   ├── assets/          # CSS, images, etc.
│   ├── components/      # Reusable Vue components
│   ├── views/           # Page-level components
│   ├── stores/          # Pinia state management stores
│   ├── router/          # Vue Router configuration
│   ├── App.vue          # Main application component
│   └── main.ts          # Application entry point
├── index.html
├── package.json
└── vite.config.ts
```

## Getting Started

### Prerequisites
- Node.js (v18 or later recommended)
- npm or yarn

### Local Development
1.  **Navigate to the directory:**
    ```sh
    cd frontend
    ```
2.  **Install dependencies:**
    ```sh
    npm install
    ```
3.  **Configure Environment Variables:**
    - Create a `.env.local` file in the `frontend` directory.
    - This file will be used to configure the API endpoints that the frontend connects to. You can copy the contents from `.env` if one exists, or create it from scratch.
      ```env
      VITE_API_URL=http://localhost:8080/api/v1
      VITE_STORE_API_URL=http://localhost:8081/api/v1
      VITE_CHAT_API_URL=http://localhost:8082/api/v1
      VITE_CHAT_WS_URL=http://localhost:8082
      ```
    *Note: These are the default URLs when running the entire platform via Docker Compose.*

4.  **Run the development server:**
    ```sh
    npm run dev
    ```
    The application will be accessible at [http://localhost:5173](http://localhost:5173).

## Available Scripts

-   `npm run dev`: Starts the development server with hot-reloading.
-   `npm run build`: Compiles and minifies the application for production.
-   `npm run preview`: Serves the production build locally for testing.
-   `npm run test:unit`: Runs unit tests.
-   `npm run test:e2e`: Runs end-to-end tests.
-   `npm run lint`: Lints and formats the codebase.