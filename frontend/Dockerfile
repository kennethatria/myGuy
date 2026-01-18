# frontend/Dockerfile
FROM node:20-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .

# Build arguments for Vite
ARG VITE_API_URL
ARG VITE_STORE_API_URL
ARG VITE_STORE_API_BASE_URL
ARG VITE_CHAT_API_URL
ARG VITE_CHAT_WS_URL

# Environment variables must be set during build for Vite
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_STORE_API_URL=$VITE_STORE_API_URL
ENV VITE_STORE_API_BASE_URL=$VITE_STORE_API_BASE_URL
ENV VITE_CHAT_API_URL=$VITE_CHAT_API_URL
ENV VITE_CHAT_WS_URL=$VITE_CHAT_WS_URL
ENV NODE_ENV=production

RUN npm run build

# Production stage
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
