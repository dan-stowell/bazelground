**rules_oci v2.2.6 Cheatsheet**

- Build, pull, load, push, sign and attest OCI container images with Bazel.
- Public entry points live under `@rules_oci//oci:*` and `@rules_oci//cosign:*`.

**What You Get**

- Rules/Macros (BUILD):
  - `oci_image` → build an OCI image from tar layer(s); produces `<name>` directory and `<name>.digest` file target.
  - `oci_image_index` → multi-arch image from one or more `oci_image`s; produces `<name>` directory and `<name>.digest`.
  - `oci_load` → runtime loader to import an image to Docker/Podman; optional tarball output group.
  - `oci_push` → push an image or index to a registry; supports remote tagging.
  - `cosign_sign` (preview) → sign an image digest in a registry.
  - `cosign_attest` (preview) → attach attestations (SLSA/SPDX/etc.) to an image in a registry.

- Repository rules/macros:
  - `oci_pull` → fetch image layers/metadata into an external repo; exposes a target usable as an `oci_image` base.
  - `oci_alias` → helper alias repo behind `oci_pull` for single/multi-platform resolution (created by `oci_pull`).

- Toolchain types:
  - `@rules_oci//oci:crane_toolchain_type` (go-containerregistry “crane”)
  - `@rules_oci//oci:regctl_toolchain_type` (regctl)
  - `@rules_oci//cosign:toolchain_type` (sigstore cosign)
  - Requires toolchains from `aspect_bazel_lib` (jq, tar, zstd, coreutils) for most rules.

- Module extension (bzlmod):
  - `@rules_oci//oci:extensions.bzl%oci` with tags `pull` and `toolchains` to set up OCI toolchains and pulled images.

-----------------------------------------------------------------------

**Bzlmod Setup (MODULE.bazel)**

Minimal, matching this release’s own module usage:

```
bazel_dep(name = "rules_oci", version = "2.2.6")
bazel_dep(name = "aspect_bazel_lib", version = "2.7.2")
bazel_dep(name = "bazel_features", version = "1.10.0")
bazel_dep(name = "bazel_skylib", version = "1.5.0")
bazel_dep(name = "platforms", version = "0.0.8")

# Register rules_oci's crane/regctl toolchains
oci = use_extension("@rules_oci//oci:extensions.bzl", "oci")
oci.toolchains()
use_repo(oci, "oci_crane_toolchains", "oci_regctl_toolchains")
register_toolchains("@oci_crane_toolchains//:all", "@oci_regctl_toolchains//:all")

# Toolchains from aspect_bazel_lib required by rules_oci (jq, tar, zstd)
zstd = use_extension("@aspect_bazel_lib//lib:extensions.bzl", "toolchains")
zstd.zstd()
use_repo(zstd, "zstd_toolchains")
register_toolchains("@zstd_toolchains//:all")

bazel_lib = use_extension("@aspect_bazel_lib//lib:extensions.bzl", "toolchains")
bazel_lib.jq()
bazel_lib.tar()
use_repo(bazel_lib, "jq_toolchains", "bsd_tar_toolchains")
register_toolchains("@jq_toolchains//:all", "@bsd_tar_toolchains//:all")
```

- Pulling images with bzlmod:
  - In MODULE.bazel: `oci.pull(name = "...", image = "...", digest = "...", platforms = [...])`
  - Then: `use_repo(oci, "<name>", "<name>_linux_amd64", ...)`
  - Use the pulled repo as a base (e.g. `base = "@<name>"`) in BUILD files.

-----------------------------------------------------------------------

**WORKSPACE Setup (non-bzlmod)**

```
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
  name = "rules_oci",
  sha256 = "<release-sha>",
  strip_prefix = "rules_oci-2.2.6",
  urls = ["https://github.com/bazel-contrib/rules_oci/releases/download/v2.2.6/rules_oci-v2.2.6.tar.gz"],
)

load("@rules_oci//oci:dependencies.bzl", "rules_oci_dependencies")
rules_oci_dependencies()

load("@rules_oci//oci:repositories.bzl", "oci_register_toolchains")
oci_register_toolchains(name = "oci")  # also registers jq/tar/coreutils/zstd
```

Optional signing toolchain (for cosign_sign/attest):
```
load("@rules_oci//cosign:repositories.bzl", "cosign_register_toolchains")
cosign_register_toolchains(name = "oci_cosign")
```

-----------------------------------------------------------------------

**Pull A Base Image (bzlmod or WORKSPACE)**

- MODULE.bazel:
  ```
  oci.pull(
    name = "distroless_cc",
    image = "gcr.io/distroless/cc",
    digest = "sha256:…",
    platforms = ["linux/amd64", "linux/arm64/v8"],  # multi-arch
  )
  use_repo(oci, "distroless_cc", "distroless_cc_linux_amd64", "distroless_cc_linux_arm64_v8")
  ```

- Or WORKSPACE:
  ```
  load("@rules_oci//oci:pull.bzl", "oci_pull")
  oci_pull(
    name = "distroless_cc",
    image = "gcr.io/distroless/cc",
    digest = "sha256:…",
    platforms = ["linux/amd64", "linux/arm64/v8"],
  )
  ```

Use `base = "@distroless_cc"` (single-platform) or a platform-specific repo like `@distroless_cc_linux_arm64_v8` in your BUILD.

-----------------------------------------------------------------------

**Build An Image (BUILD.bazel)**

A typical flow: build your app, package to a tar layer, assemble with `oci_image`.

```
load("@rules_pkg//pkg:tar.bzl", "pkg_tar")
load("@rules_oci//oci:defs.bzl", "oci_image", "oci_load", "oci_push", "oci_image_index")

# 1) Your program/library rules here (e.g., go_binary, py_binary, java_binary, …)

# 2) Package the app output(s) into a tar layer
pkg_tar(
  name = "app_layer",
  srcs = [":my_binary"],           # or a filegroup of runtime files
  package_dir = "/app",            # where to place in container FS
  extension = "tar.gz",            # gzip (zstd also supported by rules)
)

# 3) Compose an image (single-arch)
oci_image(
  name = "app_image",
  base = "@distroless_cc",         # from oci_pull or another oci_image
  tars = [":app_layer"],
  entrypoint = ["/app/my_binary"], # list OK
  env = {"FOO": "bar"},            # dict OK
  labels = {"org.opencontainers.image.title": "My App"},
  annotations = {"org.opencontainers.image.description": "Example image"},
  # If no base is provided, you must set os/architecture (and optional variant)
  # os = "linux", architecture = "amd64",
)

# A sha256 digest target is provided: :app_image.digest (default output is <name>.json.sha256)
```

-----------------------------------------------------------------------

**Load Locally (docker/podman)**

```
oci_load(
  name = "app_load",
  image = ":app_image",
  repo_tags = ["example/app:latest"],  # or label to a file with tags one-per-line
  # format = "docker" (default) or "oci"
)

# Load and run:
# bazel run //path:app_load
# docker run --rm example/app:latest
```

- Default output is an mtree spec; to build the tarball artifact:
  - CLI: `bazel build //path:app_load --output_groups=+tarball`
  - Or wrap with a filegroup using `output_group = "tarball"`.

-----------------------------------------------------------------------

**Push To A Registry**

```
oci_push(
  name = "app_push",
  image = ":app_image",                     # or an oci_image_index
  repository = "index.docker.io/myorg/app",
  remote_tags = ["v1.2.3", "latest"],      # or a label to a file with tags
)

# Run and override or add tags:
# bazel run //path:app_push -- --repository index.docker.io/myorg/app --tag latest
```

- Auth follows typical local config (Docker/Podman/crane); see docs for `--credential_helper` and WWW-Authenticate helpers.

-----------------------------------------------------------------------

**Multi-Arch Image**

- Approach A: explicit per-arch images
  ```
  oci_image(name = "app_linux_amd64", base = "@distroless_cc_linux_amd64", tars = [":app_layer_amd64"])
  oci_image(name = "app_linux_arm64", base = "@distroless_cc_linux_arm64_v8", tars = [":app_layer_arm64"])

  oci_image_index(
    name = "app_multi",
    images = [":app_linux_amd64", ":app_linux_arm64"],
  )
  ```

- Approach B: with Bazel platforms (experimental)
  ```
  oci_image(name = "image", tars = [":app_layer"])  # buildable for multiple platforms

  oci_image_index(
    name = "image_multi",
    images = [":image"],
    platforms = [
      "@rules_go//go/toolchain:linux_amd64",
      "@rules_go//go/toolchain:linux_arm64",
    ],
  )
  ```

- Push multi-arch:
  ```
  oci_push(
    name = "push_multi",
    image = ":image_multi",
    repository = "ghcr.io/myorg/app",
    remote_tags = ["1.0.0"],
  )
  ```

-----------------------------------------------------------------------

**cosign: Sign and Attest (Developer Preview; WORKSPACE-only)**

Setup (WORKSPACE):
- After `oci_register_toolchains`, also call:
  ```
  load("@rules_oci//cosign:repositories.bzl", "cosign_register_toolchains")
  cosign_register_toolchains(name = "oci_cosign")
  ```

Sign an image digest:
```
load("@rules_oci//cosign:defs.bzl", "cosign_sign")

cosign_sign(
  name = "sign",
  image = ":app_image",
  repository = "index.docker.io/myorg/app",  # no tag/digest here
)

# bazel run //path:sign -- --repository=index.docker.io/myorg/app
```

Attest (e.g. SPDX):
```
load("@rules_oci//cosign:defs.bzl", "cosign_attest")

cosign_attest(
  name = "attest_spdx",
  image = ":app_image",
  type = "spdx",
  predicate = "image.sbom.spdx.json",
  repository = "index.docker.io/myorg/app",
)

# bazel run //path:attest_spdx -- --repository=index.docker.io/myorg/app
```

-----------------------------------------------------------------------

**Useful Targets and Outputs**

- `oci_image` and `oci_image_index`:
  - Default output: a directory tree (OCI layout).
  - Also adds `<name>.digest` target; default output file is `<name>.json.sha256` containing the sha256 digest.

- `oci_pull` external repo:
  - Exposes a target named after the repo for use as `base = "@<name>"`.
  - Provides a `:digest` file target with the pulled image digest.
  - Multi-arch pulls produce per-platform repos like `@<name>_linux_arm64_v8`.

- `oci_load`:
  - Default: mtree spec file.
  - Output group `tarball` produces a loadable tar archive.

-----------------------------------------------------------------------

**Common Bazel Commands**

- Build image and read digest:
  - `bazel build //app:app_image.digest && cat bazel-bin/app/app_image.json.sha256`

- Load locally and run:
  - `bazel run //app:app_load && docker run --rm example/app:latest`

- Push with extra tag:
  - `bazel run //app:app_push -- --tag latest`

- Multi-arch push:
  - `bazel run //app:push_multi -- --repository ghcr.io/myorg/app`

-----------------------------------------------------------------------

**Notes/Tips**

- Auth for `oci_pull` can use `--credential_helper` for registries requiring WWW-Authenticate flows; see `docs/pull.md`.
- When no `base` is provided to `oci_image`, set `os` and `architecture` (and optional `variant`).
- `entrypoint`, `cmd`, `env`, `labels`, `annotations`, `exposed_ports`, `volumes` accept either files or inline lists/dicts (macros write helper files for you).
- For static content images, see `docs/static_content.md` for a complete example using `pkg_tar` with nginx.