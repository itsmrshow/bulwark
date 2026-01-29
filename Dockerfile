# Build frontend
FROM node:20-alpine AS web-build
WORKDIR /app/web
COPY web/package.json ./
RUN npm install
COPY web/ ./
RUN npm run build

# Build backend
FROM golang:1.24-bookworm AS go-build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=web-build /app/web/dist ./web/dist
RUN CGO_ENABLED=1 go build -o /out/bulwark ./cmd/bulwark

# Runtime image
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=go-build /out/bulwark /usr/local/bin/bulwark
COPY --from=web-build /app/web/dist /app/web/dist
ENV BULWARK_UI_DIST=/app/web/dist
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/bulwark"]
CMD ["serve"]
