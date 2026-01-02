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

# Copy and set permissions for entrypoint script BEFORE switching user
COPY docker-entrypoint.sh /app/
RUN chmod +x /app/docker-entrypoint.sh

# Change ownership
RUN chown -R nodejs:nodejs /app

USER nodejs

# Expose port
EXPOSE 8082

# Start the server with migrations
CMD ["/app/docker-entrypoint.sh"]