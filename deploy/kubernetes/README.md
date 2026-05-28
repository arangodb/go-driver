# Kubernetes Integration Tests

This folder contains the shared runner for executing driver integration tests against an ArangoDB deployment managed by [kube-arangodb](https://github.com/arangodb/kube-arangodb).

The runner installs the kube-arangodb operator, creates an `ArangoDeployment`, creates a TLS Ingress, and then runs the provided test command against that external endpoint.

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

CircleCI runs the same Make targets through `run-k8s-integration-tests`. The job installs `kubectl` and `minikube`, starts a Docker-backed minikube cluster with the ingress addon, and runs tests from the existing Docker test container through the Kubernetes Ingress endpoint. It invokes one of:

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
- `kubectl` on `PATH`
- a command that can run that driver's integration tests against externally supplied endpoints
- support for endpoint/auth environment variables, or a thin adapter target that maps them to the driver's own test variables

The runner passes these environment variables to the test command:

- `TEST_ENDPOINTS_OVERRIDE`: endpoint for the deployed ArangoDB, for example `https://arangodb.local`.
- `TEST_AUTHENTICATION_OVERRIDE`: `basic:root:<password>`, `jwt:root:<password>`, or empty when auth is disabled
- `TEST_MODE_K8S`: set to `k8s`, so tests can avoid using Kubernetes-internal DNS names directly
- `TEST_NET_OVERRIDE`: Docker networking option used by Dockerized tests to reach the Ingress hostname.

Example adapter target in another driver:

```make
run-k8s-driver-tests:
	@bash ./deploy/kubernetes/run-driver-tests.sh run \
	  sh -c 'ENDPOINTS="$${TEST_ENDPOINTS_OVERRIDE}" AUTH="$${TEST_AUTHENTICATION_OVERRIDE}" ./scripts/run-integration-tests.sh'
```

For Make-based projects, pass `make` explicitly:

```bash
bash ./deploy/kubernetes/run-driver-tests.sh run make run-v2-tests-single-with-auth
```

## Useful Environment

- `KUBE_ARANGODB_VERSION`: kube-arangodb release to install, default `1.4.3`.
- `K8S_NAMESPACE`: namespace for the temporary `ArangoDeployment`, default `default`. The operator is installed in `default`, so this keeps the test deployment in the namespace watched by that operator.
- `K8S_DEPLOYMENT`: deployment name, default `arangodb-driver-tests`.
- `K8S_MODE`: `Cluster` or `Single`, default `Cluster`.
- `K8S_AUTHENTICATION`: set to `false` to disable ArangoDB authentication in the Kubernetes deployment, default `true`.
- `K8S_TEST_AUTHENTICATION`: driver authentication mode, `basic`, `jwt`, or `none`, default `basic`.
- `K8S_TLS`: set to `true` to enable TLS in the `ArangoDeployment` and pass an `https://` endpoint to the tests.
- `K8S_INGRESS_HOST`: host name used by ingress mode, default `arangodb.local`.
- `K8S_INGRESS_ADDRESS`: IP address mapped into the Docker test container for `K8S_INGRESS_HOST`. When empty, the runner tries `minikube ip`.
- `K8S_INGRESS_TLS`: set to `false` to expose the Ingress over HTTP instead of HTTPS, default `true`.
- `K8S_STUCK_INIT_TIMEOUT`: delete and let kube-arangodb recreate pods stuck in `init-lifecycle` longer than this, default `5m`.
- `K8S_KEEP_DEPLOYMENT`: set to `true` to keep the deployment after a run.
- `K8S_DELETE_NAMESPACE`: set to `true` to delete a non-default namespace during cleanup.
- `ARANGO_ROOT_PASSWORD`: root password configured in Kubernetes and passed to tests, default `rootpw`.
- `ARANGO_LICENSE_KEY`: optional Enterprise license key. When set, the runner creates the kube-arangodb license secret and references it from the `ArangoDeployment`.
- `ENABLE_VECTOR_INDEX`: set to `true` to add `--vector-index=true` and `--experimental-vector-index=true` to the ArangoDB pods.

The runner creates a self-signed TLS secret and an Ingress for `K8S_INGRESS_HOST`, then passes `https://K8S_INGRESS_HOST` to the Dockerized tests with a Docker `--add-host` mapping to the ingress IP.

Single mode starts one ArangoDB server. Cluster mode starts 1 Agent, 3 DBServers, and 1 Coordinator. The 3 DBServers are needed because some integration tests update collection replication factor to 3.
