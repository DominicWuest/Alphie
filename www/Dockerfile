FROM node:alpine3.15 AS builder

COPY . .

RUN yarn
RUN yarn build

# -------------------------------- #

FROM node:alpine3.15 AS prod

WORKDIR /app

RUN apk add --no-cache python3
COPY checkEnv.py .

COPY package.json yarn.lock ./
RUN yarn --prod

COPY --from=builder build build

EXPOSE 3000

CMD python3 checkEnv.py && node build/build/index.js