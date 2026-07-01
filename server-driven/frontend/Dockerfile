FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:22-alpine
RUN npm install -g serve
COPY --from=builder /app/dist /app
EXPOSE 5173
CMD ["serve", "-s", "/app", "-l", "5173"]
