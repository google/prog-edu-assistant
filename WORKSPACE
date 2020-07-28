# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "8663604808d2738dc615a2c3eb70eba54a9a982089dd09f6ffe5d0e75771bc4f",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.23.6/rules_go-v0.23.6.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.23.6/rules_go-v0.23.6.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "cdb02a887a7187ea4d5a27452311a75ed8637379a1287d8eeb952138ea485f7d",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.21.1/bazel-gazelle-v0.21.1.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.21.1/bazel-gazelle-v0.21.1.tar.gz",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

git_repository(
    name = "com_google_protobuf",
    commit = "09745575a923640154bcf307fba8aedff47f240a",
    remote = "https://github.com/protocolbuffers/protobuf",
    shallow_since = "1558721209 -0700",
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

load("@bazel_gazelle//:deps.bzl", "go_repository")

go_repository(
    name = "org_golang_google_grpc",
    build_file_proto_mode = "disable",
    importpath = "google.golang.org/grpc",
    commit = "master"
)

go_repository(
    name = "org_golang_x_net",
    importpath = "golang.org/x/net",
    sum = "h1:oWX7TPOiFAMXLq8o0ikBYfCJVlRHBcsciT5bXOrH628=",
    version = "v0.0.0-20190311183353-d8887717615a",
)

go_repository(
    name = "org_golang_x_text",
    importpath = "golang.org/x/text",
    sum = "h1:g61tztE5qeGQ89tm6NTjjM9VPIm088od1l6aSorWRWg=",
    version = "v0.3.0",
)

go_repository(
    name = "com_github_sergi_go_diff",
    commit = "1744e2970ca51c86172c8190fadad617561ed6e7",  # v1.0.0
    importpath = "github.com/sergi/go-diff",
    remote = "https://github.com/sergi/go-diff",
    vcs = "git",
)

go_repository(
    name = "com_github_andreyvit_diff",
    commit = "c7f18ee00883bfd3b00e0a2bf7607827e0148ad4",  # HEAD from 2017-04-06
    importpath = "github.com/andreyvit/diff",
    remote = "https://github.com/andreyvit/diff",
    vcs = "git",
)

go_repository(
    name = "in_gopkg_yaml_v2",
    commit = "53403b58ad1b561927d19068c655246f2db79d48",  # v2.2.8
    importpath = "gopkg.in/yaml.v2",
)

go_repository(
    name = "com_github_golang_glog",
    commit = "23def4e6c14b4da8ac2ed8007337bc5eb5007998",
    importpath = "github.com/golang/glog",
)

go_repository(
    name = "org_golang_x_oauth2",
    commit = "9f3314589c9a9136388751d9adae6b0ed400978a",
    importpath = "golang.org/x/oauth2",
)

go_repository(
    name = "com_google_cloud_go",
    importpath = "cloud.google.com/go",
    tag = "v0.47.0",
)

go_repository(
    name = "com_github_google_uuid",
    commit = "v1.1.1",
    importpath = "github.com/google/uuid",
)

go_repository(
    name = "com_github_streadway_amqp",
    commit = "75d898a42a940fbc854dfd1a4199eabdc00cf024",
    importpath = "github.com/streadway/amqp",
)

go_repository(
    name = "com_github_gorilla_sessions",
    commit = "v1.1.3",
    importpath = "github.com/gorilla/sessions",
)

go_repository(
    name = "com_github_gorilla_securecookie",
    commit = "v1.1.1",
    importpath = "github.com/gorilla/securecookie",
)

go_repository(
    name = "com_github_gorilla_context",
    commit = "v1.1.1",
    importpath = "github.com/gorilla/context",
)

go_repository(
    name = "org_golang_google_api",
    commit = "master",
    importpath = "google.golang.org/api",
)

go_repository(
    name = "com_github_googleapis_gax_go",
    commit = "master",
    importpath = "github.com/googleapis/gax-go",
)

go_repository(
    name = "io_opencensus_go",
    importpath = "go.opencensus.io",
    tag = "master",
)

go_repository(
    name = "com_github_golang_groupcache",
    commit = "master",
    importpath = "github.com/golang/groupcache",
)

# Dependency of github.com/sourcegraph/syntaxhighlight.
go_repository(
    name = "com_github_sourcegraph_annotate",
    commit = "f4cad6c6324d3f584e1743d8b3e0e017a5f3a636",
    importpath = "github.com/sourcegraph/annotate",
    remote = "https://github.com/sourcegraph/annotate",
    vcs = "git",
)

# github.com/sourcegraph/syntaxhighlight is used by inline tests to highlight
# python source code submission.
go_repository(
    name = "com_github_sourcegraph_syntaxhighlight",
    commit = "bd320f5d308e1a3c4314c678d8227a0d72574ae7",
    importpath = "github.com/sourcegraph/syntaxhighlight",
    remote = "https://github.com/sourcegraph/syntaxhighlight",
    vcs = "git",
)

# JOSE dependency
go_repository(
    name = "org_golang_x_crypto",
    commit = "d585fd2cc9195196078f516b69daff6744ef5e84",
    importpath = "golang.org/x/crypto",
)

go_repository(
    name = "in_gopkg_square_go_jose_v2",
    commit = "master",
    importpath = "gopkg.in/square/go-jose.v2",
)

go_repository(
    name = "com_github_square_go_jose",
    importpath = "github.com/square/go-jose",
    tag = "v2.4.0",
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "5dcd5820604c5b7e7c5f7db6e2b0cd1cf59eb0a30a0076fe3a4b86198365479a",
    strip_prefix = "rules_docker-21c19afed2bfbbee7e266bcbef98d70df33670d9",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/21c19afed2bfbbee7e266bcbef98d70df33670d9.tar.gz"],
)

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)
load(
    "@io_bazel_rules_docker//go:image.bzl",
    _go_image_repos = "repositories",
)

container_repositories()

_go_image_repos()

load("@io_bazel_rules_docker//container:container.bzl", "container_pull")

container_pull(
    name = "alpine_with_bash",
    registry = "gcr.io",
    repository = "google-containers/alpine-with-bash",
    tag = "1.0",
)

container_pull(
    name = "debian_testing",
    registry = "index.docker.io",
    repository = "library/debian",
    tag = "testing",
)
