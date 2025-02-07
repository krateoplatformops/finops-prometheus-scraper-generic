package secrets

import (
	"context"

	helperclient "github.com/krateoplatformops/finops-prometheus-scraper-generic/internal/helpers/kube/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"

	finopsdatatypes "github.com/krateoplatformops/finops-data-types/api/v1"
)

const (
	resourceName = "secrets"
)

func NewClient(rc *rest.Config) (*Client, error) {
	gv := schema.GroupVersion{
		Group:   "",
		Version: "v1",
	}

	sb := runtime.NewSchemeBuilder(
		func(reg *runtime.Scheme) error {
			reg.AddKnownTypes(
				gv,
				&corev1.Secret{},
				&corev1.SecretList{},
				&metav1.ListOptions{},
				&metav1.GetOptions{},
				&metav1.DeleteOptions{},
				&metav1.CreateOptions{},
				&metav1.UpdateOptions{},
				&metav1.PatchOptions{},
				&metav1.Status{},
			)
			return nil
		})

	s := runtime.NewScheme()
	sb.AddToScheme(s)

	config := *rc
	config.APIPath = "/api"
	config.GroupVersion = &gv
	config.NegotiatedSerializer = serializer.NewCodecFactory(s).
		WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	cli, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	pc := runtime.NewParameterCodec(s)

	return &Client{rc: cli, pc: pc}, nil
}

type Client struct {
	rc rest.Interface
	pc runtime.ParameterCodec
	ns string
}

func (c *Client) Namespace(ns string) *Client {
	c.ns = ns
	return c
}

func (c *Client) Get(ctx context.Context, name string, options metav1.GetOptions) (result *corev1.Secret, err error) {
	result = &corev1.Secret{}
	err = c.rc.Get().
		Namespace(c.ns).
		Resource(resourceName).
		Name(name).
		VersionedParams(&options, c.pc).
		Do(ctx).
		Into(result)
	return
}

func Get(ctx context.Context, rc *rest.Config, sel *finopsdatatypes.SecretKeySelector) (*corev1.Secret, error) {
	cli, err := helperclient.New(rc, schema.GroupVersion{Group: "", Version: "v1"})
	if err != nil {
		return nil, err
	}

	res := &corev1.Secret{}
	err = cli.Get().
		Resource("secrets").
		Namespace(sel.Namespace).Name(sel.Name).
		Do(ctx).
		Into(res)

	return res, err
}
