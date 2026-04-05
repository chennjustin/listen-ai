# ListenAI Node gateway
# Env: GATEWAY_PORT, STAT_URL, NLP_URL, JWT_SECRET, DEMO_USER, DEMO_PASS
FROM node:20-alpine

WORKDIR /app

ENV NODE_ENV=production

COPY package.json package-lock.json ./
RUN npm ci --omit=dev

COPY src ./src

EXPOSE 8000

CMD ["node", "src/index.js"]
