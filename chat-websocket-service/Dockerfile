FROM node:18-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install --omit=dev

# Copy source code
COPY . .

# Create non-root user
RUN addgroup -g 1001 -S nodejs
RUN adduser -S nodejs -u 1001

# Change ownership
RUN chown -R nodejs:nodejs /app

USER nodejs

# Expose port
EXPOSE 8082

# Start script with migrations
COPY docker-entrypoint.sh /app/
RUN chmod +x /app/docker-entrypoint.sh

# Start the server with migrations
CMD ["/app/docker-entrypoint.sh"]