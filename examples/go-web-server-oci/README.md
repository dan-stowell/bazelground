# Go Echo HTTP Server (Bazel + OCI)

This example shows how to build a tiny Go HTTP echo server with Bazel and package it as an OCI image using rules_oci.

## Build

- Build the Go binary:
  - `bazel build //examples/go-web-server-oci:echo_server`
- Build the OCI image (and its digest):
  - `bazel build //examples/go-web-server-oci:echo_server_image.digest`

## Run (container engine)

- Load the image into Docker/Podman via Bazel:
  - `bazel run //examples/go-web-server-oci:echo_server_image.load`
- Then run it (example with Docker):
  - `docker run --rm -p 8080:8080 examples/go-web-server-oci/echo_server`

Open http://localhost:8080 and send requests; the server echoes method, path, headers, and body as JSON.

Notes:
- The `MODULE.bazel` at repo root declares `rules_go` (0.57.0), `rules_oci` (2.2.6), and a distroless base via `oci.pull`.
- Network access is required on first build to fetch the Go SDK and OCI toolchains.
- The image uses `@distroless_base` and sets entrypoint to `/echo_server`.
