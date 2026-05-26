FROM oven/bun:1 AS build
WORKDIR /app
COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile
COPY . .
ARG API_URL=http://api:4000
RUN bun run build

FROM oven/bun:1-slim
RUN groupadd -r nuage && useradd -r -g nuage nuage
WORKDIR /app
COPY --from=build /app/build ./build
COPY --from=build /app/package.json ./
RUN chown -R nuage:nuage /app
USER nuage
ENV PORT=3000
ENV ORIGIN=http://localhost:3000
ENV BODY_SIZE_LIMIT=Infinity
EXPOSE 3000
CMD ["bun", "./build/index.js"]
