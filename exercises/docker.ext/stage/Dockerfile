# Start the build image to build nsjail.
FROM debian:testing
RUN apt-get -y update && apt-get -y upgrade
# Install the certificates
RUN apt install -y ca-certificates && update-ca-certificates

# The requirements for building nsjail.
RUN apt-get -y install \
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
  
# Build nsjail
RUN git clone https://github.com/google/nsjail.git /root/nsjail
COPY nsjail.patch /tmp/nsjail.patch
RUN cd /root/nsjail && patch -p0 < /tmp/nsjail.patch && make

# Start the deployment image to run Python.
FROM debian:testing
RUN apt-get -y update && apt-get -y upgrade && touch /tmp/force-rebuild
# Install the certificates
RUN apt install -y ca-certificates && update-ca-certificates
# Install the python environment (and nsjail runtime dependencies).
# TODO(salikh): Move this to a separate stage.
RUN apt-get install -y --fix-missing \
    python3 python3-dev python3-scipy python3-pip python-virtualenv \
    libprotobuf17  libnl-route-3-200 \
    freetype* bash mecab libmecab-dev swig \
  && ln -s /usr/bin/pip3 /usr/bin/pip \
  && ln -sf /usr/bin/python3 /usr/bin/python
RUN pip install --no-cache-dir \
    six arrow scikit-learn pandas numpy scikit-image Jinja2 pygments plotly_express mecab-python3

# Copy the nsjail binary from the dev image 0.
COPY --from=0  /root/nsjail/nsjail /usr/local/bin/nsjail
COPY bin/uploadserver /usr/local/bin/uploadserver
COPY static /static
COPY autograder /autograder
RUN mkdir /tmp/uploads /tmp/scratch

# By default use port 8000.
EXPOSE 8000:8000/tcp

ENTRYPOINT ["/usr/local/bin/uploadserver", "--upload_dir=/tmp/uploads", "--logtostderr", "--v=5", "--static_dir=/static", "--autograder_dir=/autograder", "--scratch_dir=/tmp/scratch", "--python_path=/usr/bin/python", "--nsjail_path=/usr/local/bin/nsjail", "--allow_cors", "--disable_cleanup", "--auto_remove", "--grade_locally", "--log_to_bucket", "--use_jwt", "--secure_cookie", "--use_openid"]
