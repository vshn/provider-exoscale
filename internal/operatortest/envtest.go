package operatortest

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var invalidNSNameCharacters = regexp.MustCompile("[^a-z0-9-]")

// Suite is the common test suite for integration tests using envtest.
// It's expected that concrete suites use this suite as the base.
type Suite struct {
	suite.Suite

	NS      string
	Client  client.Client
	Config  *rest.Config
	Env     *envtest.Environment
	Logger  logr.Logger
	Context context.Context
	cancel  context.CancelFunc
	Scheme  *runtime.Scheme
	Manager manager.Manager

	ExoClientMock *ClientWithResponsesInterface
}

// SetupSuite implements suite.SetupAllSuite.
// It is run before running all the tests in the suite and is used to start up a local Kubernetes API server.
func (ts *Suite) SetupSuite() {
	ts.Logger = zapr.NewLogger(zaptest.NewLogger(ts.T()))
	log.SetLogger(ts.Logger)

	ts.Context, ts.cancel = context.WithCancel(context.Background())

	envtestAssets, ok := os.LookupEnv("KUBEBUILDER_ASSETS")
	if !ok {
		ts.FailNow("The environment variable KUBEBUILDER_ASSETS is undefined. Configure your IDE to set this variable when running the integration test.")
	}
	crdDir, ok := os.LookupEnv("ENVTEST_CRD_DIR")
	if !ok {
		ts.FailNow("The environment variable ENVTEST_CRD_DIR is undefined. Configure your IDE to set this variable when running the integration test.")
	}

	info, err := os.Stat(envtestAssets)
	absEnvtestAssets, _ := filepath.Abs(envtestAssets)
	ts.Require().NoErrorf(err, "'%s' does not seem to exist. Check KUBEBUILDER_ASSETS and make sure you run `make test-integration` before you run this test in your IDE.", absEnvtestAssets)
	ts.Require().Truef(info.IsDir(), "'%s' does not seem to be a directory. Check KUBEBUILDER_ASSETS and make sure you run `make test-integration` before you run this test in your IDE.", absEnvtestAssets)

	absCrds, _ := filepath.Abs(crdDir)
	info, err = os.Stat(crdDir)
	ts.Require().NoErrorf(err, "'%s' does not seem to exist. Make sure to set the working directory to the project root.", absCrds)
	ts.Require().Truef(info.IsDir(), "'%s' does not seem to be a directory. Make sure to set the working directory to the project root.", absCrds)

	ts.Logger.Info("envtest directories", "crd", absCrds, "binary assets", absEnvtestAssets)

	testEnv := &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     []string{crdDir},
		BinaryAssetsDirectory: envtestAssets,
	}

	config, err := testEnv.Start()
	ts.Require().NoError(err)
	ts.Require().NotNil(config)

	registerCommonCRDs(ts)

	k8sClient, err := client.New(config, client.Options{
		Scheme: ts.Scheme,
	})
	ts.Require().NoError(err)
	ts.Require().NotNil(k8sClient)

	ts.Env = testEnv
	ts.Config = config
	ts.Client = k8sClient

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		MetricsBindAddress: "0",
		Scheme:             ts.Scheme,
		Port:               0,
	})
	ts.Require().NoError(err)
	ts.Require().NotNil(mgr)
	ts.Manager = mgr

	ts.ExoClientMock = &ClientWithResponsesInterface{}

	// start manager in background since the k8s client cache is not started otherwise.
	// error which appears without that:
	//   logger.go:130: 2022-11-08T10:26:16.618+0100	DEBUG	Cannot get managed resource	{"controller": "testredis", "request": "/test-create", "error": "the cache is not started, can not read objects"}
	go func() {
		err = mgr.Start(ts.Context)
		ts.Require().NoError(err)
	}()
}

func registerCommonCRDs(ts *Suite) {
	ts.Scheme = runtime.NewScheme()
	ts.RegisterScheme(exoscalev1.SchemeBuilder.AddToScheme)
	ts.RegisterScheme(corev1.SchemeBuilder.AddToScheme)

	// +kubebuilder:scaffold:scheme
}

// RegisterScheme passes the current scheme to the given SchemeBuilder func.
func (ts *Suite) RegisterScheme(addToScheme func(s *runtime.Scheme) error) {
	ts.Require().NoError(addToScheme(ts.Scheme))
}

// TearDownSuite implements suite.TearDownAllSuite.
// It is used to shut down the local envtest environment.
func (ts *Suite) TearDownSuite() {
	ts.Logger.Info("starting tear down")
	ts.cancel()
	err := ts.Env.Stop()
	ts.Require().NoErrorf(err, "error while stopping test environment")
	ts.Logger.Info("test environment stopped")
}

// NewNS returns a new Namespace object with the given name.
// Note: The namespace is not actually created, use EnsureNS for this.
func (ts *Suite) NewNS(nsName string) *corev1.Namespace {
	ts.Assert().Emptyf(validation.IsDNS1123Label(nsName), "'%s' does not appear to be a valid name for a namespace", nsName)

	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}
}

// EnsureNS creates a new Namespace object using Suite.Client.
func (ts *Suite) EnsureNS(nsName string) {
	ns := ts.NewNS(nsName)
	ts.T().Logf("creating namespace '%s'", nsName)
	err := ts.Client.Create(ts.Context, ns)
	if err != nil && apierrors.IsAlreadyExists(err) {
		return
	}
	ts.Require().NoError(err)
}

// EnsureResources ensures that the given resources are existing in the suite. Each error will fail the test.
func (ts *Suite) EnsureResources(resources ...client.Object) {
	for _, resource := range resources {
		ts.T().Logf("creating resource '%s/%s'", resource.GetNamespace(), resource.GetName())
		ts.Require().NoError(ts.Client.Create(ts.Context, resource))
	}
}

// UpdateResources ensures that the given resources are updated in the suite. Each error will fail the test.
func (ts *Suite) UpdateResources(resources ...client.Object) {
	for _, resource := range resources {
		ts.T().Logf("updating resource '%s/%s'", resource.GetNamespace(), resource.GetName())
		ts.Require().NoError(ts.Client.Update(ts.Context, resource))
	}
}

// UpdateStatus ensures that the Status property of the given resources are updated in the suite. Each error will fail the test.
func (ts *Suite) UpdateStatus(resources ...client.Object) {
	for _, resource := range resources {
		ts.T().Logf("updating status '%s/%s'", resource.GetNamespace(), resource.GetName())
		ts.Require().NoError(ts.Client.Status().Update(ts.Context, resource))
	}
}

// DeleteResources deletes the given resources are updated from the suite. Each error will fail the test.
func (ts *Suite) DeleteResources(resources ...client.Object) {
	for _, resource := range resources {
		ts.T().Logf("deleting '%s/%s'", resource.GetNamespace(), resource.GetName())
		ts.Require().NoError(ts.Client.Delete(ts.Context, resource))
	}
}

// FetchResource fetches the given object name and stores the result in the given object.
// Test fails on errors.
func (ts *Suite) FetchResource(name types.NamespacedName, object client.Object) {
	ts.Require().NoError(ts.Client.Get(ts.Context, name, object))
}

// FetchResources fetches resources and puts the items into the given list with the given list options.
// Test fails on errors.
func (ts *Suite) FetchResources(objectList client.ObjectList, opts ...client.ListOption) {
	ts.Require().NoError(ts.Client.List(ts.Context, objectList, opts...))
}

// MapToRequest maps the given object into a reconcile Request.
func (ts *Suite) MapToRequest(object metav1.Object) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      object.GetName(),
			Namespace: object.GetNamespace(),
		},
	}
}

// SanitizeNameForNS first converts the given name to lowercase using strings.ToLower
// and then remove all characters but `a-z` (only lower case), `0-9` and the `-` (dash).
func (ts *Suite) SanitizeNameForNS(name string) string {
	return invalidNSNameCharacters.ReplaceAllString(strings.ToLower(name), "")
}
