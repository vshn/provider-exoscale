//go:build generate

// Clean samples dir
//go:generate rm -rf ./samples/*

// Generate sample files
//go:generate go run generate_sample.go ./samples

package main

import (
	"fmt"
	"github.com/vshn/provider-exoscale/apis"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	providerv1 "github.com/vshn/provider-exoscale/apis/provider/v1"
	"io"
	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authentication/v1"
	"log"
	"os"
	"path/filepath"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	serializerjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

var scheme = runtime.NewScheme()

func main() {
	failIfError(apis.AddToScheme(scheme))
	generateExoscaleIAMKeySample()
	generateProviderConfigSample()
	generateIAMKeyAdmissionRequest()
}

func generateExoscaleIAMKeySample() {
	spec := newIAMKeySample()
	serialize(spec, true)
}

func generateProviderConfigSample() {
	spec := newProviderConfigSample()
	serialize(spec, true)
}

func newIAMKeySample() *exoscalev1.IAMKey {
	return &exoscalev1.IAMKey{
		TypeMeta: metav1.TypeMeta{
			APIVersion: exoscalev1.IAMKeyGroupVersionKind.GroupVersion().String(),
			Kind:       exoscalev1.IAMKeyKind,
		},
		ObjectMeta: metav1.ObjectMeta{Name: "iam-key"},
		Spec: exoscalev1.IAMKeySpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: &xpv1.Reference{Name: "provider-config"},
				WriteConnectionSecretToReference: &xpv1.SecretReference{
					Name:      "my-exoscale-user-credentials",
					Namespace: "default",
				},
			},
			ForProvider: exoscalev1.IAMKeyParameters{
				KeyName: "iam-key",
				Zone:    "CH-DK-2",
				Services: exoscalev1.ServicesSpec{
					exoscalev1.SOSSpec{
						Buckets: []string{"bucket.test.1"},
					},
				},
			},
		},
		Status: exoscalev1.IAMKeyStatus{},
	}
}

func newProviderConfigSample() *providerv1.ProviderConfig {
	return &providerv1.ProviderConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: providerv1.ProviderConfigGroupVersionKind.GroupVersion().String(),
			Kind:       providerv1.ProviderConfigKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "provider-config"},
		Spec: providerv1.ProviderConfigSpec{
			Credentials: providerv1.ProviderCredentials{
				Source: xpv1.CredentialsSourceInjectedIdentity,
				APISecretRef: corev1.SecretReference{
					Name:      "api-secret",
					Namespace: "crossplane-system",
				},
			},
		},
	}
}

// generateIAMKeyAdmissionRequest generates an update request that will fail.
func generateIAMKeyAdmissionRequest() {
	oldSpec := newIAMKeySample()
	newSpec := newIAMKeySample()
	newSpec.Spec.ForProvider.KeyName = "another"
	oldSpec.Status.AtProvider.KeyName = oldSpec.Spec.ForProvider.KeyName

	gvk := metav1.GroupVersionKind{Group: exoscalev1.Group, Version: exoscalev1.Version, Kind: exoscalev1.IAMKeyKind}
	gvr := metav1.GroupVersionResource{Group: exoscalev1.Group, Version: exoscalev1.Version, Resource: exoscalev1.IAMKeyKind}
	admission := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"},
		Request: &admissionv1.AdmissionRequest{
			Object:          runtime.RawExtension{Object: newSpec},
			OldObject:       runtime.RawExtension{Object: oldSpec},
			Kind:            gvk,
			Resource:        gvr,
			RequestKind:     &gvk,
			RequestResource: &gvr,
			Name:            oldSpec.Name,
			Operation:       admissionv1.Update,
			UserInfo: authv1.UserInfo{
				Username: "admin",
				Groups:   []string{"system:authenticated"},
			},
		},
	}
	serialize(admission, false)
}

func failIfError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func serialize(object runtime.Object, useYaml bool) {
	gvk := object.GetObjectKind().GroupVersionKind()
	fileExt := "json"
	if useYaml {
		fileExt = "yaml"
	}
	fileName := fmt.Sprintf("%s_%s.%s", strings.ToLower(gvk.Group), strings.ToLower(gvk.Kind), fileExt)
	f := prepareFile(fileName)
	err := serializerjson.NewSerializerWithOptions(serializerjson.DefaultMetaFactory, scheme, scheme, serializerjson.SerializerOptions{Yaml: useYaml, Pretty: true}).Encode(object, f)
	failIfError(err)
}

func prepareFile(file string) io.Writer {
	dir := os.Args[1]
	err := os.MkdirAll(os.Args[1], 0775)
	failIfError(err)
	f, err := os.Create(filepath.Join(dir, file))
	failIfError(err)
	return f
}
