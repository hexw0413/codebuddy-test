FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci || npm install

FROM node:20-alpine AS build
WORKDIR /app
ENV VITE_API_BASE=/api
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:20-alpine
WORKDIR /app
ENV HOST=0.0.0.0
ENV PORT=5173
COPY --from=build /app .
EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0", "--port", "5173"]
