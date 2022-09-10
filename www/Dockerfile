FROM node:alpine3.15 AS builder

COPY . .

RUN yarn build

# -------------------------------- #

FROM node:alpine3.15 AS prod

RUN apk add --no-cache python3
COPY checkEnv.py .

COPY package.json .
RUN yarn --prod

COPY --from=builder build .

EXPOSE 3000

CMD python3 checkEnv.py && node index.js