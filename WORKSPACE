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
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/rules_go/releases/download/0.18.6/rules_go-0.18.6.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/0.18.6/rules_go-0.18.6.tar.gz",
    ],
    sha256 = "f04d2373bcaf8aa09bccb08a98a57e721306c8f6043a2a0ee610fd6853dcde3d",
)
load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains()

http_archive(
    name = "bazel_gazelle",
    sha256 = "3c681998538231a2d24d0c07ed5a7658cb72bfb5fd4bf9911157c0e9ac6a2687",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.17.0/bazel-gazelle-0.17.0.tar.gz"],
)
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
gazelle_dependencies()

go_repository(
    name = "com_github_sergi_go_diff",
    commit = "1744e2970ca51c86172c8190fadad617561ed6e7",  # v1.0.0
    importpath = "github.com/sergi/go-diff",
    remote = "https://github.com/sergi/go-diff",
    vcs = "git",
)

go_repository(
    name = "in_gopkg_yaml_v2",
    commit = "51d6538a90f86fe93ac480b35f37b2be17fef232",  # v2.2.2
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
    commit = "777200caa7fb8936aed0f12b1fd79af64cc83ec9",
    importpath = "cloud.google.com/go",
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
