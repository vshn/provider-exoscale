//go:build generate

// Generate manifests
//go:generate go run -tags generate sigs.k8s.io/controller-tools/cmd/controller-gen rbac:roleName=manager-role paths=./... output:artifacts:config=../.kind/rbac

package operator
