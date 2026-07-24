package integration

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
	"github.com/example/platform-operator/internal/controller"
)

// testEnv is the shared envtest environment for all integration tests.
var testEnv *envtest.Environment

// k8sClient is a Kubernetes client connected to the envtest API server.
var k8sClient client.Client

// mgr is the controller manager running in a goroutine.
var mgr manager.Manager

// mgrCtx is the context for the manager goroutine (canceled on teardown).
var mgrCtx context.Context

// mgrCancel cancels the manager context.
var mgrCancel context.CancelFunc

// testNamespace is the namespace used for all integration tests.
const testNamespace = "integration-test"

// crdAvailable tracks whether the PlatformApplication CRD was successfully installed.
var crdAvailable bool

// TestMain sets up the envtest environment once for all tests.
// envtest downloads real Kubernetes API server binaries and starts
// a local API server. This gives us real API behavior (validation,
// managed fields, status subresource) without needing a full cluster.
func TestMain(m *testing.M) {
	testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing: false,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		panic("failed to start envtest: " + err.Error())
	}
	defer func() {
		if err := testEnv.Stop(); err != nil {
			panic("failed to stop envtest: " + err.Error())
		}
	}()

	// Build scheme with all required types.
	scheme := runtime.NewScheme()
	_ = platformv1alpha1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = autoscalingv2.AddToScheme(scheme)
	_ = networkingv1.AddToScheme(scheme)
	_ = policyv1.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)

	// Create a client.
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		panic("failed to create client: " + err.Error())
	}

	// Install the PlatformApplication CRD programmatically.
	crdAvailable = installCRD(context.Background(), k8sClient)
	if !crdAvailable {
		// If CRD installation fails, skip all tests.
		panic("failed to install PlatformApplication CRD — integration tests cannot run")
	}

	// Create the test namespace.
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
	if err := k8sClient.Create(context.Background(), ns); err != nil {
		panic("failed to create test namespace: " + err.Error())
	}

	// Start the controller manager in a goroutine.
	mgr, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		panic("failed to create manager: " + err.Error())
	}

	// Register the reconciler.
	reconciler := &controller.PlatformApplicationReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		Recorder:    record.NewFakeRecorder(100),
		Concurrency: 1,
	}
	if err := reconciler.SetupWithManager(mgr); err != nil {
		panic("failed to setup reconciler: " + err.Error())
	}

	mgrCtx, mgrCancel = context.WithCancel(context.Background())
	go func() {
		if err := mgr.Start(mgrCtx); err != nil {
			panic("manager exited with error: " + err.Error())
		}
	}()

	// Wait for the manager cache to sync.
	if !mgr.GetCache().WaitForCacheSync(mgrCtx) {
		panic("cache did not sync")
	}

	// Run tests.
	code := m.Run()

	// Teardown.
	mgrCancel()
	time.Sleep(500 * time.Millisecond)
	_ = k8sClient.Delete(context.Background(), ns)

	if code != 0 {
		panic("tests failed")
	}
}

// installCRD creates the PlatformApplication CRD in the envtest API server.
func installCRD(ctx context.Context, c client.Client) bool {
	preserveUnknownFields := true
	crd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "platformapplications.platform.example.io",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "platform.example.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind:       "PlatformApplication",
				ListKind:   "PlatformApplicationList",
				Plural:     "platformapplications",
				Singular:   "platformapplication",
				ShortNames: []string{"papp"},
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  true,
					Storage: true,
					Subresources: &apiextensionsv1.CustomResourceSubresources{
						Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
					},
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type:                   "object",
							XPreserveUnknownFields: &preserveUnknownFields,
						},
					},
					AdditionalPrinterColumns: []apiextensionsv1.CustomResourceColumnDefinition{
						{Name: "Ready", Type: "string", JSONPath: ".status.conditions[?(@.type==\"Ready\")].status"},
						{Name: "Age", Type: "date", JSONPath: ".metadata.creationTimestamp"},
					},
				},
			},
		},
	}

	if err := c.Create(ctx, crd); err != nil {
		return false
	}

	// Wait for the CRD to be established.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		existing := &apiextensionsv1.CustomResourceDefinition{}
		if err := c.Get(ctx, client.ObjectKeyFromObject(crd), existing); err == nil {
			for _, cond := range existing.Status.Conditions {
				if cond.Type == apiextensionsv1.Established && cond.Status == apiextensionsv1.ConditionTrue {
					return true
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// newTestApp creates a minimal PlatformApplication for integration tests.
func newTestApp(name string) *platformv1alpha1.PlatformApplication {
	return &platformv1alpha1.PlatformApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: platformv1alpha1.PlatformApplicationSpec{
			Image: platformv1alpha1.ImageSpec{
				Repository: "ghcr.io/example/test-app",
				Tag:        "1.0.0",
				PullPolicy: "IfNotPresent",
			},
			Replicas: platformv1alpha1.ReplicasSpec{
				Min: 2,
				Max: 5,
			},
			Service: platformv1alpha1.ServiceSpec{
				Port: 8080,
				Type: "ClusterIP",
			},
			Autoscaling: platformv1alpha1.AutoscalingSpec{
				Enabled:              true,
				TargetCPUUtilization: 70,
			},
			HealthChecks: platformv1alpha1.HealthChecksSpec{
				LivenessPath:  "/healthz",
				ReadinessPath: "/readyz",
			},
			Rollout: platformv1alpha1.RolloutSpec{
				Strategy: "RollingUpdate",
			},
		},
	}
}

// waitForResource polls until the specified resource exists in the cluster.
func waitForResource(ctx context.Context, c client.Client, key client.ObjectKey, obj client.Object, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := c.Get(ctx, key, obj); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return context.DeadlineExceeded
}

// gvk returns the GroupVersionKind for a given object using the scheme.
func gvk(obj runtime.Object) schema.GroupVersionKind {
	gvks, _, _ := mgr.GetScheme().ObjectKinds(obj)
	if len(gvks) > 0 {
		return gvks[0]
	}
	return schema.GroupVersionKind{}
}
