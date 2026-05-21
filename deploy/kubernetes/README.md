# Kubernetes Integration Tests

This folder contains the shared runner for executing the Go driver integration tests against an ArangoDB deployment managed by [kube-arangodb](https://github.com/arangodb/kube-arangodb).

The runner installs the kube-arangodb operator, creates an `ArangoDeployment`, and then runs the existing Makefile test target. By default it uses `kubectl port-forward`; with `K8S_TEST_RUNNER=pod`, it creates a Kubernetes `Job` and runs the tests inside the cluster.

## Quick Start

Start from a machine with access to a Kubernetes cluster, for example minikube:

```bash
minikube start
make run-k8s-v2-tests
```

By default this runs the v2 Kubernetes single and cluster targets:

- `run-k8s-v2-single`
- `run-k8s-v2-cluster`

Run the default v2 Kubernetes test:

```bash
make run-k8s-v2-tests
```

## CircleCI

CircleCI runs the same Make targets through Kubernetes jobs:

- `run-k8s-integration-tests` installs `kubectl` and `minikube`, starts a Docker-backed minikube cluster, and runs tests from the existing Docker test container through `kubectl port-forward`.
- `run-k8s-pod-integration-tests` installs `kubectl` and `kind`, starts a kind cluster with the repository mounted into the node, and runs tests inside a Kubernetes `Job`.

Both jobs invoke one of:

- `make run-k8s-v2-tests`
- `make run-k8s-v2-single`
- `make run-k8s-v2-cluster`

Additional variants are available through explicit Make targets:

- `make run-k8s-v2-single-without-auth`
- `make run-k8s-v2-single-basic-auth`
- `make run-k8s-v2-single-tls-basic-auth`
- `make run-k8s-v2-cluster-basic-auth`
- `make run-k8s-v2-cluster-tls-basic-auth`

The CircleCI jobs are guarded by the existing pull-request check and skip minikube setup on non-PR pipelines.

Override the target or image:

```bash
ARANGODB=gcr.io/gcr-for-testing/arangodb/enterprise-preview:latest \
make run-k8s-v2-tests
```

## Reusing From Other Drivers

`run-driver-tests.sh` is intended to be reusable by other driver repositories. The script owns the Kubernetes setup and exposes the ArangoDB endpoint to the driver test command.

Other drivers need to provide:

- a Kubernetes cluster in the current `kubectl` context, for example minikube, kind, k3d, or a shared test cluster
- `kubectl` and `make` on `PATH`
- a Make target or wrapper command that can run that driver's integration tests against externally supplied endpoints
- support for endpoint/auth environment variables, or a thin adapter target that maps them to the driver's own test variables

The runner passes these environment variables to the test command:

- `TEST_ENDPOINTS_OVERRIDE`: endpoint for the deployed ArangoDB. In port-forward mode this is an external host endpoint such as `http://host.docker.internal:18529`; in pod mode this is an in-cluster service endpoint such as `http://go-driver-tests.default.svc:8529`.
- `TEST_AUTHENTICATION_OVERRIDE`: `basic:root:<password>`, `jwt:root:<password>`, or empty when auth is disabled
- `TEST_MODE_K8S`: set to `k8s`, so tests can avoid using Kubernetes-internal DNS names directly
- `TEST_NET_OVERRIDE`: Docker networking option used by Dockerized tests to reach the host-side port-forward. This is only used in port-forward mode.

Example adapter target in another driver:

```make
run-k8s-driver-tests:
	@bash ./deploy/kubernetes/run-driver-tests.sh run run-driver-tests

run-driver-tests:
	@ENDPOINTS="$(TEST_ENDPOINTS_OVERRIDE)" \
	  AUTH="$(TEST_AUTHENTICATION_OVERRIDE)" \
	  ./scripts/run-integration-tests.sh
```

## Useful Environment

- `KUBE_ARANGODB_VERSION`: kube-arangodb release to install, default `1.4.3`.
- `K8S_NAMESPACE`: namespace for the temporary `ArangoDeployment`, default `default`. The operator is installed in `default`, so this keeps the test deployment in the namespace watched by that operator.
- `K8S_DEPLOYMENT`: deployment name, default `go-driver-tests`.
- `K8S_MODE`: `Cluster`, `Single`, or `ActiveFailover`, default `Cluster`.
- `K8S_AUTHENTICATION`: set to `false` to disable ArangoDB authentication in the Kubernetes deployment, default `true`.
- `K8S_TEST_AUTHENTICATION`: driver authentication mode, `basic`, `jwt`, or `none`, default `basic`.
- `K8S_TLS`: set to `true` to enable TLS in the `ArangoDeployment` and pass an `https://` endpoint to the tests.
- `K8S_TEST_RUNNER`: `port-forward` to run tests from the host Docker test container, or `pod` to run tests inside a Kubernetes `Job`, default `port-forward`.
- `K8S_TEST_IMAGE`: image used for the Kubernetes test `Job`, default `golang:1.25.10` or `GOIMAGE` when set.
- `K8S_LOCAL_PORT`: local port for `kubectl port-forward`, default `18529`.
- `K8S_TEST_ENDPOINT_HOST`: host name used by Dockerized tests to reach the port-forward, default `host.docker.internal`.
- `K8S_TEST_WORKSPACE_NODE_PATH`: path to the repository inside the Kubernetes node for pod mode, default `/workspace/go-driver`.
- `K8S_TEST_WORKSPACE_MOUNT_PATH`: path where the repository is mounted in the test pod, default `/usr/code`.
- `K8S_STUCK_INIT_TIMEOUT`: delete and let kube-arangodb recreate pods stuck in `init-lifecycle` longer than this, default `5m`.
- `K8S_KEEP_DEPLOYMENT`: set to `true` to keep the deployment after a run.
- `K8S_DELETE_NAMESPACE`: set to `true` to delete a non-default namespace during cleanup.
- `ARANGO_ROOT_PASSWORD`: root password configured in Kubernetes and passed to tests, default `rootpw`.
- `ARANGO_LICENSE_KEY`: optional Enterprise license key. When set, the runner creates the kube-arangodb license secret and references it from the `ArangoDeployment`.
- `ENABLE_VECTOR_INDEX`: set to `true` to add `--vector-index=true` and `--experimental-vector-index=true` to the ArangoDB pods.

In port-forward mode, the runner binds `kubectl port-forward` locally and passes `http://host.docker.internal:${K8S_LOCAL_PORT}` to the Dockerized Go tests. This lets the test container reach the host-side port-forward in local Docker Desktop/WSL and CircleCI Docker environments.

In pod mode, the Kubernetes cluster must expose the repository inside the node at `K8S_TEST_WORKSPACE_NODE_PATH`. The CircleCI kind job does this with kind `extraMounts`, then the runner mounts that path into the test `Job`.

Single mode starts one ArangoDB server. Cluster mode starts 1 Agent, 3 DBServers, and 1 Coordinator. The 3 DBServers are needed because some integration tests update collection replication factor to 3.
