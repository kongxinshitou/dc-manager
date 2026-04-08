FROM node:20-alpine AS builder

WORKDIR /app

# 先复制依赖声明，利用层缓存
COPY package.json package-lock.json* ./
RUN npm install --registry=https://registry.npmmirror.com

COPY . .
RUN npm run build

# ---

FROM node:20-alpine
RUN npm install -g serve --registry=https://registry.npmmirror.com
WORKDIR /app
COPY --from=builder /app/dist ./dist

EXPOSE 9000

CMD ["serve", "-s", "dist", "-l", "9000"]
