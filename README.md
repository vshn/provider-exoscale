# provider-exoscale

[![Build](https://img.shields.io/github/workflow/status/vshn/provider-exoscale/Test)][build]
![Go version](https://img.shields.io/github/go-mod/go-version/vshn/provider-exoscale)
[![Version](https://img.shields.io/github/v/release/vshn/provider-exoscale)][releases]
[![GitHub downloads](https://img.shields.io/github/downloads/vshn/provider-exoscale/total)][releases]

[build]: https://github.com/vshn/provider-exoscale/actions?query=workflow%3ATest
[releases]: https://github.com/vshn/provider-exoscale/releases

Crossplane provider for managing resources on exoscale.com.

Documentation: https://vshn.github.io/provider-exoscale/

## Local Development

### Requirements

* `docker`
* `go`
* `helm`
* `kubectl`
* `yq`
* `sed` (or `gsed` for Mac)

Some other requirements (e.g. `kind`) will be compiled on-the-fly and put in the local cache dir `.kind` as needed.

### Common make targets

* `make build` to build the binary and docker image
* `make generate` to (re)generate additional code artifacts
* `make test` run test suite
* `make local-install` to install the operator in local cluster
* `make install-samples` to run the provider in local cluster and apply sample manifests
* `make run-operator` to run the code in operator mode against your current kubecontext

See all targets with `make help`

### QuickStart Demonstration

1. Get an API token exoscale.com
1. `export EXOSCALE_API_KEY=<the-key>`
1. `export EXOSCALE_API_SECRET=<the-secret>`
1. `make local-install`

### Kubernetes Webhook Troubleshooting

The provider comes with mutating and validation admission webhook server.

To test and troubleshoot the webhooks on the cluster, simply apply your changes with `kubectl`.

1.  To debug the webhook in an IDE, we need to generate certificates:
    ```bash
    make webhook-cert
    ```
2.  Start the operator in your IDE with `WEBHOOK_TLS_CERT_DIR` environment set to `.kind`.

3.  Send an admission request sample of the spec:
    ```bash
    # send an admission request
    curl -k -v -H "Content-Type: application/json" --data @samples/admission.k8s.io_admissionreview.json https://localhost:9443/validate-exoscale-crossplane-io-v1-iamkey
    ```

### Crossplane Provider Mechanics

For detailed information on how Crossplane Provider works from a development perspective check [provider mechanics documentation page](https://kb.vshn.ch/app-catalog/explanations/crossplane_provider_mechanics.html).

### e2e testing with kuttl

Some scenarios are tested with the Kubernetes E2E testing tool [Kuttl](https://kuttl.dev/docs).
Kuttl is basically comparing the installed manifests (usually files named `##-install*.yaml`) with observed objects and compares the desired output (files named `##-assert*.yaml`).

To execute tests, run `make test-e2e` from the root dir.

If a test fails, kuttl leaves the resources in the kind-cluster intact, so you can inspect the resources and events if necessary.
Please note that Kubernetes Events from cluster-scoped resources appear in the `default` namespace only, but `kubectl describe ...` should show you the events.

If tests succeed, the relevant resources are deleted to not use up costs on the cloud providers.

### Cleaning up e2e tests

Usually `make clean` ensures that buckets and users are deleted before deleting the kind cluster, provided the operator is running in kind cluster.
Alternatively, `make .e2e-test-clean` also removes all `buckets` and `iamkeys`.

To cleanup manually on portal.exoscale.com, search for resources that begin with or contain `e2e` in the name.
