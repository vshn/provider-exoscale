crossplane_sentinel = $(kind_dir)/crossplane_sentinel
registry_sentinel = $(kind_dir)/registry_sentinel

setup_envtest_bin = $(kind_dir)/setup-envtest

# Prepare binary
# We need to set the Go arch since the binary is meant for the user's OS.
$(setup_envtest_bin): export GOOS = $(shell go env GOOS)
$(setup_envtest_bin): export GOARCH = $(shell go env GOARCH)
$(setup_envtest_bin):
	@mkdir -p $(kind_dir)
	cd test && go build -o $@ sigs.k8s.io/controller-runtime/tools/setup-envtest
	$@ $(ENVTEST_ADDITIONAL_FLAGS) use '$(ENVTEST_K8S_VERSION)!'
	chmod -R +w $(kind_dir)/k8s

.PHONY: local-install
local-install: export KUBECONFIG = $(KIND_KUBECONFIG)
# for ControllerConfig:
local-install: export INTERNAL_PACKAGE_IMG = registry.registry-system.svc.cluster.local:5000/$(PROJECT_OWNER)/$(PROJECT_NAME)/package:$(IMG_TAG)
# for package-push:
local-install: PACKAGE_IMG = localhost:5000/$(PROJECT_OWNER)/$(PROJECT_NAME)/package:$(IMG_TAG)
local-install: kind-load-image crossplane-setup registry-setup $(kind_dir)/.credentials.yaml package-push  ## Install Operator in local cluster
	yq e '.spec.metadata.annotations."local.dev/installed"="$(shell date)"' test/controllerconfig-exoscale.yaml | kubectl apply -f -
	yq e '.spec.package=strenv(INTERNAL_PACKAGE_IMG)' test/provider-exoscale.yaml | kubectl apply -f -
	kubectl wait --for condition=Healthy provider.pkg.crossplane.io/provider-exoscale --timeout 60s
	kubectl -n crossplane-system wait --for condition=Ready $$(kubectl -n crossplane-system get pods -o name -l pkg.crossplane.io/provider=provider-exoscale) --timeout 60s
	kubectl apply -n crossplane-system -f $(kind_dir)/.credentials.yaml

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

.PHONY: kind-run-operator
kind-run-operator: export KUBECONFIG = $(KIND_KUBECONFIG)
kind-run-operator: kind-setup webhook-cert ## Run in Operator mode against kind cluster (you may also need `install-crd`)
	go run . -v 1 operator --webhook-tls-cert-dir $(kind_dir)

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

.PHONY: test-integration
test-integration: export ENVTEST_CRD_DIR = $(shell realpath $(envtest_crd_dir))
test-integration: $(setup_envtest_bin) .envtest_crds ## Run integration tests against code
	export KUBEBUILDER_ASSETS="$$($(setup_envtest_bin) $(ENVTEST_ADDITIONAL_FLAGS) use -i -p path '$(ENVTEST_K8S_VERSION)!')" && \
	go test -tags=integration -coverprofile cover.out -covermode atomic ./...

envtest_crd_dir ?= $(kind_dir)/crds

.envtest_crd_dir:
	@mkdir -p $(envtest_crd_dir)
	@cp -r package/crds $(kind_dir)

.envtest_crds: .envtest_crd_dir

$(kind_dir)/.credentials.yaml:
	@if [ "$$EXOSCALE_API_KEY" = "" ]; then echo "Environment variable EXOSCALE_API_KEY not set"; exit 1; fi
	@if [ "$$EXOSCALE_API_SECRET" = "" ]; then echo "Environment variable EXOSCALE_API_SECRET not set"; exit 1; fi
	kubectl create secret generic --from-literal EXOSCALE_API_KEY=$(shell echo $$EXOSCALE_API_KEY) --from-literal EXOSCALE_API_SECRET=$(shell echo $$EXOSCALE_API_SECRET) -o yaml --dry-run=client api-secret > $@

###
### Generate webhook certificates.
### This is only relevant when running in IDE with debugger.
### When installed as a provider, Crossplane handles the certificate generation.
###

webhook_key = $(kind_dir)/tls.key
webhook_cert = $(kind_dir)/tls.crt
webhook_service_name = provider-exocale.crossplane-system.svc

.PHONY: webhook-cert
webhook-cert: $(webhook_cert) ## Generate webhook certificates for out-of-cluster debugging in an IDE

$(webhook_key):
	openssl req -x509 -newkey rsa:4096 -nodes -keyout $@ --noout -days 3650 -subj "/CN=$(webhook_service_name)" -addext "subjectAltName = DNS:$(webhook_service_name)"

$(webhook_cert): $(webhook_key)
	openssl req -x509 -key $(webhook_key) -nodes -out $@ -days 3650 -subj "/CN=$(webhook_service_name)" -addext "subjectAltName = DNS:$(webhook_service_name)"
