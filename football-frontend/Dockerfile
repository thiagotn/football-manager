FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json ./
RUN npm install --legacy-peer-deps
COPY . .
# VITE_API_URL é embutida em tempo de build pelo Vite
ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL
# Generate .svelte-kit (needed for path aliases)
RUN npx svelte-kit sync || true
RUN npm run build

FROM node:22-alpine AS production
WORKDIR /app
COPY --from=builder /app/build ./build
COPY --from=builder /app/package.json ./
RUN npm install --omit=dev --legacy-peer-deps
EXPOSE 3000
ENV HOST=0.0.0.0 PORT=3000 NODE_ENV=production
CMD ["node", "build"]
