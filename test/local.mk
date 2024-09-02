crossplane_sentinel = $(kind_dir)/crossplane_sentinel
registry_sentinel = $(kind_dir)/registry_sentinel

.PHONY: local-install
local-install: export KUBECONFIG = $(KIND_KUBECONFIG)
# for ControllerConfig:
local-install: export INTERNAL_PACKAGE_IMG = registry.registry-system.svc.cluster.local:5000/$(ORG)/$(APP_NAME):$(IMG_TAG)
local-install: local-debug ## Install Operator in local cluster
	yq e '.spec.metadata.annotations."local.dev/installed"="$(shell date)"' test/controllerconfig-exoscale.yaml | kubectl apply -f -
	yq e '.spec.package=strenv(INTERNAL_PACKAGE_IMG)' test/provider-exoscale.yaml | kubectl apply -f -
	kubectl wait --for condition=Healthy provider.pkg.crossplane.io/provider-exoscale --timeout 60s
	kubectl -n crossplane-system wait --for condition=Ready $$(kubectl -n crossplane-system get pods -o name -l pkg.crossplane.io/provider=provider-exoscale) --timeout 60s

.PHONY: local-debug
local-debug: export KUBECONFIG = $(KIND_KUBECONFIG)
local-debug: kind-load-image crossplane-setup registry-setup mirror-setup package-push-local  ## Install Operator in local cluster

.PHONY: crossplane-setup
crossplane-setup: $(crossplane_sentinel) ## Installs Crossplane in kind cluster.

$(crossplane_sentinel): export KUBECONFIG = $(KIND_KUBECONFIG)
$(crossplane_sentinel): $(KIND_KUBECONFIG)
	helm repo add --force-update crossplane https://charts.crossplane.io/stable
	helm repo update
	helm upgrade --install crossplane crossplane/crossplane \
		--create-namespace \
		--namespace crossplane-system \
		--set "args[0]='--debug'" \
		--set "args[1]='--enable-composition-revisions'" \
		--set webhooks.enabled=true \
		--wait
	@touch $@

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

$(kind_dir)/.credentials.yaml:
	@if [ "$$EXOSCALE_API_KEY" = "" ]; then echo "Environment variable EXOSCALE_API_KEY not set"; exit 1; fi
	@if [ "$$EXOSCALE_API_SECRET" = "" ]; then echo "Environment variable EXOSCALE_API_SECRET not set"; exit 1; fi
	kubectl create secret generic --from-literal EXOSCALE_API_KEY=$$EXOSCALE_API_KEY --from-literal EXOSCALE_API_SECRET=$$EXOSCALE_API_SECRET -o yaml --dry-run=client api-secret > $@

.PHONY: provider-config
provider-config: export KUBECONFIG = $(KIND_KUBECONFIG)
provider-config: $(KIND_KUBECONFIG) $(kind_dir)/.credentials.yaml
	kubectl apply -n crossplane-system -f $(kind_dir)/.credentials.yaml -f samples/exoscale.crossplane.io_providerconfig.yaml

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
	go test -tags=integration ./...

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
	go install github.com/minio/mc@latest

test-e2e: export KUBECONFIG = $(KIND_KUBECONFIG)
test-e2e: $(kuttl_bin) $(mc_bin) local-install provider-config ## E2E tests
	GOBIN=$(go_bin) $(kuttl_bin) test ./test/e2e --config ./test/e2e/kuttl-test.yaml --suppress-log=Events
	@rm -f kubeconfig
# kuttl leaves kubeconfig garbage: https://github.com/kudobuilder/kuttl/issues/297

run-single-e2e: export KUBECONFIG = $(KIND_KUBECONFIG)
run-single-e2e: $(kuttl_bin) $(mc_bin) local-install provider-config ## Run specific e2e test with `run-single-e2e test=$name`
	GOBIN=$(go_bin) $(kuttl_bin) test ./test/e2e --config ./test/e2e/kuttl-test.yaml --suppress-log=Events --test $(test)
	@rm -f kubeconfig

.PHONY: .e2e-test-clean
.e2e-test-clean: export KUBECONFIG = $(KIND_KUBECONFIG)
.e2e-test-clean:
	@if [ -f $(KIND_KUBECONFIG) ]; then \
		kubectl delete buckets --all || true; \
		kubectl delete iamkeys --all || true; \
		kubectl delete postgresql --all || true; \
		kubectl delete mysql --all || true; \
		kubectl delete redis --all || true; \
		kubectl delete kafka --all || true; \
		kubectl delete opensearch --all || true; \
	else \
		echo "no kubeconfig found"; \
	fi
	rm -f $(kuttl_bin) $(mc_bin)

.PHONY: webhook-debug
webhook_service_name = host.docker.internal

webhook-debug: export KUBECONFIG = $(KIND_KUBECONFIG)
webhook-debug: $(webhook_cert) ## Creates certificates, patches the webhook registrations and applies everything to the given kube cluster
webhook-debug:
	kubectl apply -f package/crds/
	kubectl apply -f package/webhook/manifests.yaml
	kubectl apply -f samples/exoscale.crossplane.io_providerconfig.yaml
	kubectl apply -f samples/lab-provider-keys-exo.yaml
	cabundle=$$(cat $(kind_dir)/tls.crt | base64) && \
	kubectl annotate validatingwebhookconfigurations.admissionregistration.k8s.io validating-webhook-configuration kubectl.kubernetes.io/last-applied-configuration- && \
	kubectl annotate validatingwebhookconfigurations.admissionregistration.k8s.io validating-webhook-configuration cert-manager.io/inject-ca-from- && \
	kubectl get validatingwebhookconfigurations.admissionregistration.k8s.io validating-webhook-configuration -oyaml | \
	yq e "with(.webhooks[]; .clientConfig.caBundle = \"$$cabundle\") | with(.webhooks[]; .clientConfig.url = \"https://$(webhook_service_name):9443\" + .clientConfig.service.path) | with(.webhooks[]; del(.clientConfig.service))"  | \
	kubectl replace -f - && \
	kubectl annotate validatingwebhookconfigurations.admissionregistration.k8s.io validating-webhook-configuration kubectl.kubernetes.io/last-applied-configuration-

# Generate webhook certificates.
# This is only relevant when debugging.
# Component-appcat installs a proper certificate for this.
.PHONY: webhook-cert
webhook-cert: $(webhook_cert) ## Generate webhook certificates for out-of-cluster debugging in an IDE

$(webhook_key):
	ipsan="" && \
	if [[ $(webhook_service_name) =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$$ ]]; then \
		ipsan=", IP:$(webhook_service_name)"; \
	fi; \
	openssl req -x509 -newkey rsa:4096 -nodes -keyout $@ --noout -days 3650 -subj "/CN=$(webhook_service_name)" -addext "subjectAltName = DNS:$(webhook_service_name)$$ipsan"

$(webhook_cert): $(webhook_key)
	ipsan="" && \
	if [[ $(webhook_service_name) =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$$ ]]; then \
		ipsan=", IP:$(webhook_service_name)"; \
	fi; \
	openssl req -x509 -key $(webhook_key) -nodes -out $@ -days 3650 -subj "/CN=$(webhook_service_name)" -addext "subjectAltName = DNS:$(webhook_service_name)$$ipsan"
