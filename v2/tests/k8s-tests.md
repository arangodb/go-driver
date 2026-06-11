# v2 Kubernetes Integration Tests

These tests run the v2 Go driver integration suite against an ArangoDB deployment managed by kube-arangodb.

The shared Kubernetes setup lives in `deploy/kubernetes/run-driver-tests.sh`. It creates the kind cluster, installs ingress-nginx, installs kube-arangodb, creates the ArangoDeployment, and then delegates to the v2 Make targets listed below.

## Local Run

Create or reuse a local kind cluster:

```bash
bash ./deploy/kubernetes/run-driver-tests.sh setup-kind
```

Run the full v2 Kubernetes test set:

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-tests
```

Run only the v2 cluster basic-auth scenario:

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 VERBOSE=1 ENABLE_VECTOR_INDEX=true make run-k8s-v2-cluster-basic-auth
```

## Make Targets

- `make run-k8s-v2-tests`
- `make run-k8s-v2-single`
- `make run-k8s-v2-cluster`
- `make run-k8s-v2-single-without-auth`
- `make run-k8s-v2-single-basic-auth`
- `make run-k8s-v2-single-tls-basic-auth`
- `make run-k8s-v2-cluster-basic-auth`
- `make run-k8s-v2-cluster-tls-basic-auth`

## Cleanup

The runner removes the temporary ArangoDeployment, Ingress, and secrets after `run` unless `K8S_KEEP_DEPLOYMENT=true` is set.

To delete the kind cluster as well:

```bash
bash ./deploy/kubernetes/run-driver-tests.sh cleanup-kind
```

For a fully clean run that deletes the kind cluster after tests complete:

```bash
K8S_DELETE_KIND_CLUSTER=true K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-tests
```
