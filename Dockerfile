
FROM golang:1.26.0-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY database ./database
COPY pkg ./pkg

ENV CGO_ENABLED=1
RUN go build -o /app/server ./cmd/server


RUN go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

FROM alpine:3.20

# sqlite is here so you can open the database inside the container:
# docker exec -it social-backend sqlite3 db/social_network.db
RUN apk add --no-cache ca-certificates sqlite sqlite-libs tzdata

WORKDIR /app

COPY --from=builder /app/server ./server
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate

COPY pkg/db/migrations ./pkg/db/migrations
# the default avatar new accounts start with
COPY images ./images

RUN mkdir -p ./db ./uploads/avatars ./uploads/posts ./uploads/groups/avatars ./uploads/comments ./images/avatar

EXPOSE 4000

CMD ["./server"]
