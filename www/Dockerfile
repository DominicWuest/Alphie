FROM node:alpine3.15 AS builder

COPY . .

RUN yarn build

# -------------------------------- #

FROM node:alpine3.15 AS prod

COPY package.json .
RUN yarn --prod

COPY --from=builder build .

EXPOSE 3000

CMD node index.js