crossplane_sentinel = $(kind_dir)/crossplane_sentinel
registry_sentinel = $(kind_dir)/registry_sentinel

.PHONY: local-install
local-install: export KUBECONFIG = $(KIND_KUBECONFIG)
# for ControllerConfig:
local-install: export INTERNAL_PACKAGE_IMG = registry.registry-system.svc.cluster.local:5000/$(PROJECT_OWNER)/$(PROJECT_NAME)/package:$(IMG_TAG)
# for package-push:
local-install: PACKAGE_IMG = localhost:5000/$(PROJECT_OWNER)/$(PROJECT_NAME)/package:$(IMG_TAG)
local-install: kind-load-image crossplane-setup registry-setup package-push  ## Install Operator in local cluster
	yq e '.spec.metadata.annotations."local.dev/installed"="$(shell date)"' test/controllerconfig-exoscale.yaml | kubectl apply -f -
	yq e '.spec.package=strenv(INTERNAL_PACKAGE_IMG)' test/provider-exoscale.yaml | kubectl apply -f -
	kubectl wait --for condition=Healthy provider.pkg.crossplane.io/provider-exoscale --timeout 60s
	kubectl -n crossplane-system wait --for condition=Ready $$(kubectl -n crossplane-system get pods -o name -l pkg.crossplane.io/provider=provider-exoscale) --timeout 60s

.PHONY: crossplane-setup
crossplane-setup: $(crossplane_sentinel) ## Installs Crossplane in kind cluster.

$(crossplane_sentinel): export KUBECONFIG = $(KIND_KUBECONFIG)
$(crossplane_sentinel): $(KIND_KUBECONFIG)
	helm repo add crossplane https://charts.crossplane.io/stable
	helm upgrade --install crossplane crossplane/crossplane \
		--create-namespace \
		--namespace crossplane-system \
		--set "args[0]='--debug'" \
		--set "args[1]='--enable-composition-revisions'" \
		--set webhooks.enabled=true \
		--wait
	@touch $@


$(kind_dir)/.credentials.yaml:
	@if [ "$$EXOSCALE_API_KEY" = "" ]; then echo "Environment variable EXOSCALE_API_KEY not set"; exit 1; fi
	@if [ "$$EXOSCALE_API_SECRET" = "" ]; then echo "Environment variable EXOSCALE_API_SECRET not set"; exit 1; fi
	kubectl create secret generic --from-literal EXOSCALE_API_KEY=$$EXOSCALE_API_KEY --from-literal EXOSCALE_API_SECRET=$$EXOSCALE_API_SECRET -o yaml --dry-run=client api-secret > $@

.PHONY: provider-config
provider-config: export KUBECONFIG = $(KIND_KUBECONFIG)
provider-config: $(KIND_KUBECONFIG) $(kind_dir)/.credentials.yaml
	kubectl apply -n crossplane-system -f $(kind_dir)/.credentials.yaml -f samples/exoscale.crossplane.io_providerconfig.yaml

.PHONY: registry-setup
registry-setup: $(registry_sentinel) ## Installs an image registry required for the package image in kind cluster.

$(registry_sentinel): export KUBECONFIG = $(KIND_KUBECONFIG)
$(registry_sentinel): $(KIND_KUBECONFIG)
	helm repo add twuni https://helm.twun.io
	helm upgrade --install registry twuni/docker-registry \
		--create-namespace \
		--namespace registry-system \
		--set service.type=NodePort \
		--set service.nodePort=30500 \
		--set fullnameOverride=registry \
		--wait
	@touch $@

###
### Integration Tests
###

setup_envtest_bin = $(go_bin)/setup-envtest
envtest_crd_dir ?= $(kind_dir)/crds

# Prepare binary
$(setup_envtest_bin): export GOBIN = $(go_bin)
$(setup_envtest_bin): | $(go_bin)
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: test-integration
test-integration: export ENVTEST_CRD_DIR = $(envtest_crd_dir)
test-integration: $(setup_envtest_bin) .envtest_crds ## Run integration tests against code
	$(setup_envtest_bin) $(ENVTEST_ADDITIONAL_FLAGS) use '$(ENVTEST_K8S_VERSION)!'
	chmod -R +w $(kind_dir)/k8s
	export KUBEBUILDER_ASSETS="$$($(setup_envtest_bin) $(ENVTEST_ADDITIONAL_FLAGS) use -i -p path '$(ENVTEST_K8S_VERSION)!')" && \
	go test -tags=integration -coverprofile cover.out -covermode atomic ./...

.envtest_crd_dir:
	@mkdir -p $(envtest_crd_dir)
	@cp -r package/crds $(kind_dir)

.envtest_crds: .envtest_crd_dir

.PHONY: .envtest-clean
.envtest-clean:
	rm -f $(setup_envtest_bin)

###
### Local debugging
###

.PHONY: kind-run-operator
kind-run-operator: export KUBECONFIG = $(KIND_KUBECONFIG)
kind-run-operator: kind-setup webhook-cert ## Run in Operator mode against kind cluster
	go run . -v 1 operator --webhook-tls-cert-dir $(kind_dir)

webhook_key = $(kind_dir)/tls.key
webhook_cert = $(kind_dir)/tls.crt
webhook_service_name = provider-exocale.crossplane-system.svc

# Generate webhook certificates.
# This is only relevant when running in IDE with debugger.
# When installed as a provider, Crossplane handles the certificate generation.
.PHONY: webhook-cert
webhook-cert: $(webhook_cert) ## Generate webhook certificates for out-of-cluster debugging in an IDE

$(webhook_key):
	openssl req -x509 -newkey rsa:4096 -nodes -keyout $@ --noout -days 3650 -subj "/CN=$(webhook_service_name)" -addext "subjectAltName = DNS:$(webhook_service_name)"

$(webhook_cert): $(webhook_key)
	openssl req -x509 -key $(webhook_key) -nodes -out $@ -days 3650 -subj "/CN=$(webhook_service_name)" -addext "subjectAltName = DNS:$(webhook_service_name)"

###
### E2E Tests
### with KUTTL (https://kuttl.dev)
###

kuttl_bin = $(go_bin)/kubectl-kuttl
$(kuttl_bin): export GOBIN = $(go_bin)
$(kuttl_bin): | $(go_bin)
	go install github.com/kudobuilder/kuttl/cmd/kubectl-kuttl@latest

mc_bin = $(go_bin)/mc
$(mc_bin): export GOBIN = $(go_bin)
$(mc_bin): | $(go_bin)
	go insta\ll github.com/minio/mc@latest

test-e2e: export KUBECONFIG = $(KIND_KUBECONFIG)
test-e2e: $(kuttl_bin) $(mc_bin) local-install provider-config ## E2E tests
	GOBIN=$(go_bin) $(kuttl_bin) test ./test/e2e --config ./test/e2e/kuttl-test.yaml
	@rm kubeconfig
# kuttle leaves kubeconfig garbage: https://github.com/kudobuilder/kuttl/issues/297

.PHONY: .e2e-test-clean
.e2e-test-clean:
	rm -f $(kuttl_bin) $(mc_bin)
