//go:build generate

// Clean samples dir
//go:generate rm -rf ./samples/*

// Generate sample files
//go:generate go run generate_sample.go ./samples

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	exoscaleoapi "github.com/exoscale/egoscale/v2/oapi"
	"github.com/vshn/provider-exoscale/apis"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	providerv1 "github.com/vshn/provider-exoscale/apis/provider/v1"
	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authentication/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	serializerjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

var scheme = runtime.NewScheme()

func main() {
	failIfError(apis.AddToScheme(scheme))
	generateBucketSample()
	generateExoscaleIAMKeySample()
	generateProviderConfigSample()
	generateIAMKeyAdmissionRequest()
	generateMysqlSample()
	generatePostgresqlSample()
	generateRedisSample()
	generateKafkaSample()
	generateOpensearchSample()
}

func generatePostgresqlSample() {
	spec := newPostgresqlSample()
	serialize(spec, true)
}

func newPostgresqlSample() *exoscalev1.PostgreSQL {
	return &exoscalev1.PostgreSQL{
		TypeMeta: metav1.TypeMeta{
			APIVersion: exoscalev1.PostgreSQLGroupVersionKind.GroupVersion().String(),
			Kind:       exoscalev1.PostgreSQLKind,
		},
		ObjectMeta: metav1.ObjectMeta{Name: "postgresql-local-dev"},
		Spec: exoscalev1.PostgreSQLSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
				WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "postgresql-local-dev-details", Namespace: "default"},
			},
			ForProvider: exoscalev1.PostgreSQLParameters{
				Maintenance: exoscalev1.MaintenanceSpec{
					TimeOfDay: "12:00:00",
					DayOfWeek: exoscaleoapi.DbaasServiceMaintenanceDowMonday,
				},
				Backup: exoscalev1.BackupSpec{
					TimeOfDay: "13:00:00",
				},
				Zone: "ch-dk-2",
				DBaaSParameters: exoscalev1.DBaaSParameters{
					Size: exoscalev1.SizeSpec{
						Plan: "hobbyist-2",
					},
					IPFilter: exoscalev1.IPFilter{"0.0.0.0/0"},
				},
				Version:    "14",
				PGSettings: runtime.RawExtension{Raw: []byte(`{"timezone":"Europe/Zurich"}`)},
			},
		},
	}
}

func generateMysqlSample() {
	spec := newMysqlSample()
	serialize(spec, true)
}

func newMysqlSample() *exoscalev1.MySQL {
	return &exoscalev1.MySQL{
		TypeMeta: metav1.TypeMeta{
			APIVersion: exoscalev1.MySQLGroupVersionKind.GroupVersion().String(),
			Kind:       exoscalev1.MySQLKind,
		},
		ObjectMeta: metav1.ObjectMeta{Name: "mysql-local-dev"},
		Spec: exoscalev1.MySQLSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
				WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "mysql-local-dev-details", Namespace: "default"},
			},
			ForProvider: exoscalev1.MySQLParameters{
				Maintenance: exoscalev1.MaintenanceSpec{
					TimeOfDay: "12:00:00",
					DayOfWeek: exoscaleoapi.DbaasServiceMaintenanceDowMonday,
				},
				Backup: exoscalev1.BackupSpec{
					TimeOfDay: "13:00:00",
				},
				Zone: "ch-dk-2",
				DBaaSParameters: exoscalev1.DBaaSParameters{
					Size: exoscalev1.SizeSpec{
						Plan: "hobbyist-2",
					},
					IPFilter: exoscalev1.IPFilter{"0.0.0.0/0"},
				},
				Version:       "8",
				MySQLSettings: runtime.RawExtension{Raw: []byte(`{"default_time_zone":"+01:00"}`)},
			},
		},
	}
}

func generateBucketSample() {
	spec := newBucketSample()
	serialize(spec, true)
}

func generateExoscaleIAMKeySample() {
	spec := newIAMKeySample()
	serialize(spec, true)
}

func generateProviderConfigSample() {
	spec := newProviderConfigSample()
	serialize(spec, true)
}

func newBucketSample() *exoscalev1.Bucket {
	return &exoscalev1.Bucket{
		TypeMeta: metav1.TypeMeta{
			APIVersion: exoscalev1.BucketGroupVersionKind.GroupVersion().String(),
			Kind:       exoscalev1.BucketKind,
		},
		ObjectMeta: metav1.ObjectMeta{Name: "bucket-local-dev"},
		Spec: exoscalev1.BucketSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: &xpv1.Reference{Name: "provider-config"},
			},
			ForProvider: exoscalev1.BucketParameters{
				BucketName:           "bucket-local-dev",
				Zone:                 "ch-gva-2",
				BucketDeletionPolicy: exoscalev1.DeleteIfEmpty,
			},
		},
	}
}

func newIAMKeySample() *exoscalev1.IAMKey {
	return &exoscalev1.IAMKey{
		TypeMeta: metav1.TypeMeta{
			APIVersion: exoscalev1.IAMKeyGroupVersionKind.GroupVersion().String(),
			Kind:       exoscalev1.IAMKeyKind,
		},
		ObjectMeta: metav1.ObjectMeta{Name: "iam-key-local-dev"},
		Spec: exoscalev1.IAMKeySpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: &xpv1.Reference{Name: "provider-config"},
				WriteConnectionSecretToReference: &xpv1.SecretReference{
					Name:      "my-exoscale-user-credentials",
					Namespace: "default",
				},
			},
			ForProvider: exoscalev1.IAMKeyParameters{
				KeyName: "iam-key-local-dev",
				Zone:    "CH-DK-2",
				Services: exoscalev1.ServicesSpec{
					SOS: exoscalev1.SOSSpec{
						Buckets: []string{"bucket-local-dev"},
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

func generateRedisSample() {
	spec := newRedisSample()
	serialize(spec, true)
}

func newRedisSample() *exoscalev1.Redis {
	return &exoscalev1.Redis{
		TypeMeta: metav1.TypeMeta{
			APIVersion: exoscalev1.RedisGroupVersionKind.GroupVersion().String(),
			Kind:       exoscalev1.RedisKind,
		},
		ObjectMeta: metav1.ObjectMeta{Name: "redis-local-dev"},
		Spec: exoscalev1.RedisSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
				WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "redis-local-dev-details", Namespace: "default"},
			},
			ForProvider: exoscalev1.RedisParameters{
				Maintenance: exoscalev1.MaintenanceSpec{
					TimeOfDay: "12:00:00",
					DayOfWeek: exoscaleoapi.DbaasServiceMaintenanceDowMonday,
				},
				Zone: "ch-dk-2",
				DBaaSParameters: exoscalev1.DBaaSParameters{
					Size: exoscalev1.SizeSpec{
						Plan: "hobbyist-2",
					},
					IPFilter: exoscalev1.IPFilter{"0.0.0.0/0"},
				},
				RedisSettings: runtime.RawExtension{Raw: []byte(`{"maxmemory_policy":"noeviction"}`)},
			},
		},
	}
}
func generateKafkaSample() {
	spec := newKafkaSample()
	serialize(spec, true)
}

func newKafkaSample() *exoscalev1.Kafka {
	return &exoscalev1.Kafka{
		TypeMeta: metav1.TypeMeta{
			APIVersion: exoscalev1.KafkaGroupVersionKind.GroupVersion().String(),
			Kind:       exoscalev1.KafkaKind,
		},
		ObjectMeta: metav1.ObjectMeta{Name: "kafka-local-dev"},
		Spec: exoscalev1.KafkaSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
				WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "kafka-local-dev-details", Namespace: "default"},
			},
			ForProvider: exoscalev1.KafkaParameters{
				Maintenance: exoscalev1.MaintenanceSpec{
					TimeOfDay: "12:00:00",
					DayOfWeek: exoscaleoapi.DbaasServiceMaintenanceDowMonday,
				},
				Zone: "ch-dk-2",
				DBaaSParameters: exoscalev1.DBaaSParameters{
					Size: exoscalev1.SizeSpec{
						Plan: "startup-2",
					},
					IPFilter: exoscalev1.IPFilter{"0.0.0.0/0"},
				},
				KafkaSettings: runtime.RawExtension{Raw: []byte(`{"connections_max_idle_ms": 60000}`)},
			},
		},
	}
}

func generateOpensearchSample() {
	spec := newOpensearchSample()
	serialize(spec, true)
}

func newOpensearchSample() *exoscalev1.OpenSearch {
	return &exoscalev1.OpenSearch{
		TypeMeta: metav1.TypeMeta{
			Kind:       exoscalev1.OpenSearchKind,
			APIVersion: exoscalev1.OpenSearchGroupVersionKind.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "opensearch-local-dev",
		},
		Spec: exoscalev1.OpenSearchSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
				WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "opensearch-local-dev-details", Namespace: "default"},
			},
			ForProvider: exoscalev1.OpenSearchParameters{
				Maintenance: exoscalev1.MaintenanceSpec{
					DayOfWeek: exoscaleoapi.DbaasServiceMaintenanceDowMonday,
					TimeOfDay: "12:01:55",
				},
				DBaaSParameters: exoscalev1.DBaaSParameters{
					TerminationProtection: false,
					Size: exoscalev1.SizeSpec{
						Plan: "hobbyist-2",
					},
					IPFilter: exoscalev1.IPFilter{"0.0.0.0/0"},
				},
				Zone:               "ch-dk-2",
				MajorVersion:       "2",
				OpenSearchSettings: runtime.RawExtension{},
			},
		},
	}
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
