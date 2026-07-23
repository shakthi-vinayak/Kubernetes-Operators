package subreconcilers

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
)

// ReconcileHTTPRoute computes the desired HTTPRoute and applies it,
// or deletes it if gateway is disabled.
func ReconcileHTTPRoute(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	app *platformv1alpha1.PlatformApplication,
) (ApplyResult, error) {
	if !app.Spec.Gateway.Enabled {
		return deleteHTTPRouteIfExists(ctx, c, app)
	}

	desired := buildDesiredHTTPRoute(app)
	return Apply(ctx, c, scheme, app, desired)
}

func buildDesiredHTTPRoute(app *platformv1alpha1.PlatformApplication) *gatewayv1.HTTPRoute {
	labels := CommonLabels(app.Name)

	pathPrefix := app.Spec.Gateway.PathPrefix
	if pathPrefix == "" {
		pathPrefix = "/"
	}

	pathMatchType := gatewayv1.PathMatchPathPrefix

	route := &gatewayv1.HTTPRoute{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "gateway.networking.k8s.io/v1",
			Kind:       "HTTPRoute",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{
				{
					Matches: []gatewayv1.HTTPRouteMatch{
						{
							Path: &gatewayv1.HTTPPathMatch{
								Type:  &pathMatchType,
								Value: &pathPrefix,
							},
						},
					},
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Group: (*gatewayv1.Group)(strPtr("")),
									Kind:  (*gatewayv1.Kind)(strPtr("Service")),
									Name:  gatewayv1.ObjectName(app.Name),
									Port:  (*gatewayv1.PortNumber)(portPtr(app.Spec.Service.Port)),
								},
							},
						},
					},
				},
			},
		},
	}

	// Set hostname if configured.
	if app.Spec.Gateway.Host != "" {
		hostname := gatewayv1.Hostname(app.Spec.Gateway.Host)
		route.Spec.Hostnames = []gatewayv1.Hostname{hostname}
	}

	// Set parent gateway reference if configured.
	if app.Spec.Gateway.GatewayRef != "" {
		parts := strings.SplitN(app.Spec.Gateway.GatewayRef, "/", 2)
		if len(parts) == 2 {
			ns := gatewayv1.Namespace(parts[0])
			gwName := gatewayv1.ObjectName(parts[1])
			gwGroup := gatewayv1.Group("gateway.networking.k8s.io")
			gwKind := gatewayv1.Kind("Gateway")
			route.Spec.ParentRefs = []gatewayv1.ParentReference{
				{
					Group:     &gwGroup,
					Kind:      &gwKind,
					Namespace: &ns,
					Name:      gwName,
				},
			}
		}
	}

	return route
}

func deleteHTTPRouteIfExists(ctx context.Context, c client.Client, app *platformv1alpha1.PlatformApplication) (ApplyResult, error) {
	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
	}
	if err := DeleteIfExists(ctx, c, route); err != nil {
		return "", err
	}
	return ApplyResultUnchanged, nil
}

// ReconcileNetworkPolicy computes the desired NetworkPolicy and applies it,
// or deletes it if security.networkPolicy is disabled.
func ReconcileNetworkPolicy(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	app *platformv1alpha1.PlatformApplication,
) (ApplyResult, error) {
	if !app.Spec.Security.NetworkPolicy {
		return deleteNetworkPolicyIfExists(ctx, c, app)
	}

	desired := buildDesiredNetworkPolicy(app)
	return Apply(ctx, c, scheme, app, desired)
}

func buildDesiredNetworkPolicy(app *platformv1alpha1.PlatformApplication) *networkingv1.NetworkPolicy {
	labels := CommonLabels(app.Name)
	protocol := gatewayv1.ProtocolType("TCP")

	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "NetworkPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-netpol", app.Name),
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":     app.Name,
					"app.kubernetes.io/instance": app.Name,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: protocolType(&protocol),
							Port:     portIntStr(app.Spec.Service.Port),
						},
					},
				},
			},
			// Allow all egress (DNS, external services, etc.)
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{},
			},
		},
	}
}

func deleteNetworkPolicyIfExists(ctx context.Context, c client.Client, app *platformv1alpha1.PlatformApplication) (ApplyResult, error) {
	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-netpol", app.Name),
			Namespace: app.Namespace,
		},
	}
	if err := DeleteIfExists(ctx, c, np); err != nil {
		return "", err
	}
	return ApplyResultUnchanged, nil
}

func strPtr(s string) *string {
	return &s
}

func portPtr(p int32) *int32 {
	return &p
}

func protocolType(p *gatewayv1.ProtocolType) *corev1.Protocol {
	if p == nil {
		return nil
	}
	proto := corev1.Protocol(string(*p))
	return &proto
}

func portIntStr(port int32) *intstr.IntOrString {
	v := intstr.FromInt32(port)
	return &v
}
