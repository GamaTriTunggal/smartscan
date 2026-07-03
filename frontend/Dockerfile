FROM node:20-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies as root (before switching user)
RUN npm install

# Copy source code
COPY . .

# Create necessary directories and clean any cached files
RUN mkdir -p /app/.nuxt /app/.output && \
    rm -rf /app/node_modules/.vite /app/node_modules/.cache

# Note: Running as root for development to handle volume mount permissions
# In production, use multi-stage build with proper user permissions

# Expose port
EXPOSE 3000

# Start development server
# npm ci is strict — it reads package-lock.json verbatim and wipes node_modules
# if state drifts from the lockfile. This prevents volume-drift bugs where
# newly added packages silently stay missing (observed 2026-04-24 with driver.js
# and 4 other packages missing from the named volume despite being in lockfile).
# --prefer-offline reuses cache for speed; --no-audit skips startup audit noise.
CMD ["sh", "-c", "npm ci --prefer-offline --no-audit && npm run dev"]
