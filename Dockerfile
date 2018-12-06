FROM golang:alpine AS build-env
WORKDIR /app
ADD main.go /app
RUN cd /app && \
    apk -U add git && \
    go get k8s.io/client-go/... && \
    go build -o go-k8s-example

FROM alpine
WORKDIR /app
COPY --from=build-env /app/go-k8s-example /app
CMD [ "./go-k8s-example" ]