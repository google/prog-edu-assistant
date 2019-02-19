# Upload server

An upload server for demo purpose.

## Usage

    mkdir -p uploads
    go run cmd/server/main.go -port 8080 -upload_dir uploads

## Usage with HTTPS

    mkdir -p uploads
    openssl req -new \
      -newkey rsa:2048 \
      -days 9999 -nodes -x509 -sha256 \
      -subj "/C=JP/CN=localhost" \
      -keyout localhost.key -out localhost.crt
    go run cmd/server/main.go \
      -port 8443 -upload_dir uploads \
      -use_https -ssl_cert_file localhost.crt -ssl_key_file localhost.key
