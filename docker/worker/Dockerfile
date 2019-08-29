# Start nsjail build stage.
FROM debian:testing
RUN apt-get -y update
# The requirements for building nsjail.
RUN apt-get install -y \
  autoconf \
  bison \
  flex \
  gcc \
  g++ \
  git \
  libprotobuf-dev \
  libnl-route-3-dev \
  libtool \
  make \
  pkg-config \
  protobuf-compiler \
  bash

# Install the python environment (and nsjail runtime dependencies).
# TODO(salikh): Move this to a separate stage.
RUN apt-get install -y \
    python3 python3-dev python3-scipy python3-pip python-virtualenv \
    libprotobuf17  libnl-route-3-200 \
    freetype* bash \
  && ln -s /usr/bin/pip3 /usr/bin/pip \
  && ln -sf /usr/bin/python3 /usr/bin/python
RUN pip install --no-cache-dir \
    six arrow scikit-learn pandas numpy scikit-image Jinja2 pygments plotly_express

# Build nsjail
RUN git clone https://github.com/google/nsjail.git /root/nsjail \
  && cd /root/nsjail && make && mv /root/nsjail/nsjail /usr/local/bin
# Build dependencies for worker daemon
RUN apt-get install -y \
  golang-go
RUN export GOPATH=/go && go get -u -v github.com/streadway/amqp
# TODO(salikh): Build the Go binaries properly.
#RUN git clone https://github.com/google/prog-edu-assistant /root/prog-edu-assistant \
#  && cd /root/prog-edu-assistant \
#  && mkdir -p bin && go build -o bin/worker worker/cmd/worker.go \
#  && mv bin/worker /usr/local/bin/worker

# TODO(salikh): Split out the build and runtime stages.
#FROM debian:testing
#RUN apt-get -y update
# TODO(salikh): Install python environment and nsjail runtime dependencies.
# TODO(salikh): Copy nsjail.
#COPY --from=0 /usr/local/bin/nsjail /usr/local/bin/nsjail

# TODO(salikh): Build the Go binaries properly.
# Copy the pre-built static binary of the worker.
COPY bin/worker /usr/local/bin/

# Copy the pre-built autograder scripts.
# TODO(salikh): Build multiple assignments into a single directory.
COPY autograder /autograder

#CMD ["/bin/bash", "-i"]
CMD ["/usr/local/bin/worker", "--autograder_dir=/autograder", "--logtostderr", "--v=3", "--queue_spec", "amqp://guest:guest@rabbitmq:5672/"]
