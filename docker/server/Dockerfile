FROM alpine

# Install the certificates
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
RUN update-ca-certificates

COPY bin/uploadserver /usr/local/bin/uploadserver
RUN mkdir /tmp/uploads


ENTRYPOINT ["/usr/local/bin/uploadserver", "--upload_dir=/tmp/uploads", "--logtostderr", "--v=3", "--queue_spec", "amqp://guest:guest@rabbitmq:5672/"]
