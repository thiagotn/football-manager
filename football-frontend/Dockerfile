FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --legacy-peer-deps
COPY . .
# VITE_API_URL é embutida em tempo de build pelo Vite
ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL
# Dados do controlador legal (Termos de Uso e Política de Privacidade)
ARG PUBLIC_LEGAL_CONTROLLER_NAME
ARG PUBLIC_LEGAL_CONTROLLER_DOC
ARG PUBLIC_LEGAL_FORUM_CITY
ARG PUBLIC_LEGAL_CONTACT_EMAIL
ENV PUBLIC_LEGAL_CONTROLLER_NAME=$PUBLIC_LEGAL_CONTROLLER_NAME
ENV PUBLIC_LEGAL_CONTROLLER_DOC=$PUBLIC_LEGAL_CONTROLLER_DOC
ENV PUBLIC_LEGAL_FORUM_CITY=$PUBLIC_LEGAL_FORUM_CITY
ENV PUBLIC_LEGAL_CONTACT_EMAIL=$PUBLIC_LEGAL_CONTACT_EMAIL
# Generate .svelte-kit (needed for path aliases)
RUN npx svelte-kit sync || true
RUN npm run build

FROM node:22-alpine AS production
WORKDIR /app
COPY --from=builder /app/build ./build
COPY --from=builder /app/package.json /app/package-lock.json ./
RUN npm ci --omit=dev --legacy-peer-deps
EXPOSE 3000
ENV HOST=0.0.0.0 PORT=3000 NODE_ENV=production
CMD ["node", "build"]
