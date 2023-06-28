package iamkeycontroller

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	exoscalesdk "github.com/exoscale/egoscale/v2"
	exooapi "github.com/exoscale/egoscale/v2/oapi"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	corev1 "k8s.io/api/core/v1"

	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// KeyIDAnnotationKey is the annotation key where the IAMKey ID is stored.
	KeyIDAnnotationKey  = "exoscale.crossplane.io/key-id"
	RoleIDAnnotationKey = "exoscale.crossplane.io/role-id"
	// BucketResourceType is the resource type bucket to which the IAMKey has access to.
	BucketResourceType = "bucket"
	//SOSResourceDomain is the resource domain to which the IAMKey has access to.
	SOSResourceDomain = "sos"
)

// IAMKeyPipeline provisions IAMKeys on exoscale.com
type IAMKeyPipeline struct {
	kube           client.Client
	recorder       event.Recorder
	exoscaleClient *exoscalesdk.Client
}

type pipelineContext struct {
	context.Context
	iamKey            *exoscalev1.IAMKey
	iamExoscaleKey    *exoscalesdk.IAMAccessKey
	credentialsSecret *corev1.Secret
}

type IamRolesList struct {
	IamRoles []exooapi.IamRole `json:"iam-roles"`
}

type IamKeysList struct {
	IamKeys []exooapi.IamApiKey `json:"api-keys"`
}

// NewPipeline returns a new instance of IAMKeyPipeline.
func NewPipeline(client client.Client, recorder event.Recorder, exoscaleClient *exoscalesdk.Client) *IAMKeyPipeline {
	return &IAMKeyPipeline{
		kube:           client,
		recorder:       recorder,
		exoscaleClient: exoscaleClient,
	}
}

func toConnectionDetails(iamKey *exoscalesdk.IAMAccessKey) (managed.ConnectionDetails, error) {

	if iamKey.Key == nil {
		return nil, errors.New("iamKey key not found in connection details")
	}
	if iamKey.Secret == nil {
		return nil, errors.New("iamKey secret not found in connection details")
	}
	return map[string][]byte{
		exoscalev1.AccessKeyIDName:     []byte(*iamKey.Key),
		exoscalev1.SecretAccessKeyName: []byte(*iamKey.Secret),
	}, nil
}

func fromManaged(mg resource.Managed) *exoscalev1.IAMKey {
	return mg.(*exoscalev1.IAMKey)
}

// https://github.com/exoscale/egoscale/blob/master/v2/api/security.go
// copy-pasted and adjusted to our use case
// this way we will sign all of the requests with the same signature
func signRequest(req *http.Request, expiration time.Time, apiKey, apiSecret string) error {
	var (
		sigParts    []string
		headerParts []string
	)

	// Request method/URL path
	sigParts = append(sigParts, fmt.Sprintf("%s %s", req.Method, req.URL.EscapedPath()))
	headerParts = append(headerParts, "EXO2-HMAC-SHA256 credential="+apiKey)

	// Request body if present
	body := ""
	if req.Body != nil {
		data, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}
		err = req.Body.Close()
		if err != nil {
			return err
		}
		body = string(data)
		req.Body = io.NopCloser(bytes.NewReader(data))
	}
	sigParts = append(sigParts, body)

	// Request query string parameters
	// Important: this is order-sensitive, we have to have to sort parameters alphabetically to ensure signed
	// values match the names listed in the "signed-query-args=" signature pragma.
	signedParams, paramsValues := extractRequestParameters(req)
	sigParts = append(sigParts, paramsValues)
	if len(signedParams) > 0 {
		headerParts = append(headerParts, "signed-query-args="+strings.Join(signedParams, ";"))
	}

	// Request headers -- none at the moment
	// Note: the same order-sensitive caution for query string parameters applies to headers.
	sigParts = append(sigParts, "")

	// Request expiration date (UNIX timestamp, no line return)
	sigParts = append(sigParts, fmt.Sprint(expiration.Unix()))
	headerParts = append(headerParts, "expires="+fmt.Sprint(expiration.Unix()))

	h := hmac.New(sha256.New, []byte(apiSecret))
	if _, err := h.Write([]byte(strings.Join(sigParts, "\n"))); err != nil {
		return err
	}
	headerParts = append(headerParts, "signature="+base64.StdEncoding.EncodeToString(h.Sum(nil)))

	req.Header.Set("Authorization", strings.Join(headerParts, ","))

	return nil
}

func ExecuteRequest(ctx context.Context, method, host, path string, unMarshalledBody interface{}) (*http.Response, error) {
	log := controllerruntime.LoggerFrom(ctx)
	req := &http.Request{
		Method: method,
		URL: &url.URL{
			Scheme: "https",
			Host:   fmt.Sprintf("api-%s.exoscale.com", host),
			Path:   path,
		},
		Header: http.Header{
			"Authorization": []string{""},
		},
	}

	if unMarshalledBody != nil {
		jsonbt, err := json.Marshal(unMarshalledBody)
		if err != nil {
			log.Error(err, "Cannot unmarshal body")
			return nil, err
		}

		req.Body = io.NopCloser(bytes.NewReader(jsonbt))
	}

	// config, err := rest.InClusterConfig()
	// if err != nil {
	// 	log.Error(err, "Cannot get in cluster config", "path: ", path, "host: ", host, "method: ", method, "body: ", jsoned)
	// 	return nil, err
	// }
	// // Create a Kubernetes clientset
	// clientset, err := kubernetes.NewForConfig(config)
	// if err != nil {
	// 	log.Error(err, "Cannot create kubernetes clientset")
	// 	return nil, err
	// }

	// // Specify the namespace and secret name
	// namespace := "syn-crossplane"
	// secretName := "exoscale-api-access"

	// // Retrieve the secret
	// secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	// if err != nil {
	// 	log.Error(err, "Cannot get secret", "namespace: ", namespace, "secretName: ", secretName)
	// 	return nil, err
	// }

	//fmt.Println(string(secret.Data["EXOSCALE_API_KEY"]), string(secret.Data["EXOSCALE_API_SECRET"]), req.URL.String())

	if req.Method == "POST" {
		req.Header.Set("Content-Type", "application/json")
	}

	// sign request
	//err := signRequest(req, time.Now().Add(5*time.Minute), string(secret.Data["EXOSCALE_API_KEY"]), string(secret.Data["EXOSCALE_API_SECRET"]))
	err := signRequest(req, time.Now().Add(5*time.Minute), "xx", "xx")
	if err != nil {
		log.Error(err, "Cannot sign request")
		return nil, err
	}

	// send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(err, "Cannot send request", "path: ", path, "host: ", host, "method: ", method)
		return nil, err
	}
	if resp.StatusCode != 200 {
		log.Error(err, "Request returned non 200 status code", "path: ", path, "host: ", host, "method: ", method, "status code: ", resp.StatusCode)
		resp1, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error(err, "Cannot read response body")
			return nil, err
		}
		log.Error(err, "Response body", "body: ", string(resp1))
		return nil, errors.New("request returned non 200 status code")

	}

	return resp, nil
}

// extractRequestParameters returns the list of request URL parameters names
// and a strings concatenating the values of the parameters.
// this function is copy pasted from https://github.com/exoscale/egoscale/blob/master/v2/api/security.go
func extractRequestParameters(req *http.Request) ([]string, string) {
	var (
		names  []string
		values string
	)

	for param, values := range req.URL.Query() {
		// Keep only parameters that hold exactly 1 value (i.e. no empty or multi-valued parameters)
		if len(values) == 1 {
			names = append(names, param)
		}
	}
	sort.Strings(names)

	for _, param := range names {
		values += req.URL.Query().Get(param)
	}

	return names, values
}
