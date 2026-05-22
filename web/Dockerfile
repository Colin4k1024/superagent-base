# Stage 1: Build the React app
FROM harbor-test.haier.net/paas/node:20-alpine AS builder

WORKDIR /app

# Install dependencies first (layer cache)
COPY package.json ./
RUN npm install --frozen-lockfile

# Copy source and build
COPY . .
RUN npm run build

# Stage 2: Serve with Nginx
FROM harbor-test.haier.net/paas/nginx:1.25-alpine

# Remove default nginx config
RUN rm /etc/nginx/conf.d/default.conf

# Copy built assets and proxy config
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/superagent.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
