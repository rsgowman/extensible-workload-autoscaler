# Contributing to xAS

Thank you for your interest in contributing to the Kubernetes *Extensible Workload Autoscaler* (xAS)!

> [!NOTE]
> xAS is an early-stage prototype focused on establishing the core architecture, identifying use cases, and refining API definitions. For design background, see the [architecture document](docs/architecture.md) and [README](README.md).

## Before You Begin

- **Contributor License Agreement (CLA):** Contributions must be accompanied by a [Google CLA](https://cla.developers.google.com/about). Visit <https://cla.developers.google.com/> to view existing agreements or sign a new one. See [docs/contributing.md](docs/contributing.md) for details.
- **Community Guidelines:** We follow [Google's Open Source Community Guidelines](https://opensource.google/conduct/).

## Development Workflow

1. **Fork and branch:** Fork the repository and create a branch from `main`.
2. **Make your changes:**
   - New Go files should include the Apache 2.0 license header from [`hack/boilerplate.go.txt`](hack/boilerplate.go.txt).
   - If you modify Kubernetes API types (`api/xas`) or Protobuf definitions (`api/server`), regenerate code and manifests:
     ```sh
     make codegen
     ```
3. **Verify your changes:** Run the full verification suite (formatting, `go vet`, build, unit tests, and codegen checks):
   ```sh
   make verify
   ```
   You can also install a pre-commit hook to run `make verify` automatically:
   ```sh
   hack/install-git-hooks.sh
   ```
4. **Test on a local cluster (optional):** Follow the [Quickstart](README.md#quickstart) to spin up a KinD cluster with `hack/create-kind-cluster.sh` and deploy with `hack/deploy.sh`.

## Pull Requests

All changes are submitted and reviewed through GitHub pull requests. Please include tests for new functionality and a clear description of what changed and why.

By contributing, you agree that your contributions will be licensed under the [Apache 2.0 License](LICENSE).
