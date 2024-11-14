package secrets

import (
	"context"

	finopsDataTypes "github.com/krateoplatformops/finops-data-types/api/v1"
	client "github.com/krateoplatformops/finops-prometheus-scraper-generic/internal/helpers/kube/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

func Get(ctx context.Context, rc *rest.Config, sel *finopsDataTypes.SecretKeySelector) (*corev1.Secret, error) {
	cli, err := client.New(rc, schema.GroupVersion{Group: "", Version: "v1"})
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

func Create(ctx context.Context, rc *rest.Config, secret *corev1.Secret) error {
	cli, err := client.New(rc, schema.GroupVersion{Group: "", Version: "v1"})
	if err != nil {
		return err
	}

	return cli.Post().
		Namespace(secret.GetNamespace()).
		Resource("secrets").
		Body(secret).
		Do(ctx).
		Error()
}

func Update(ctx context.Context, rc *rest.Config, secret *corev1.Secret) error {
	cli, err := client.New(rc, schema.GroupVersion{Group: "", Version: "v1"})
	if err != nil {
		return err
	}
	return cli.Put().
		Namespace(secret.GetNamespace()).
		Resource("secrets").
		Name(secret.Name).
		Body(secret).
		Do(ctx).
		Error()
}

func Delete(ctx context.Context, rc *rest.Config, sel *finopsDataTypes.SecretKeySelector) error {
	cli, err := client.New(rc, schema.GroupVersion{Group: "", Version: "v1"})
	if err != nil {
		return err
	}

	return cli.Delete().
		Namespace(sel.Namespace).
		Resource("secrets").
		Name(sel.Name).
		Do(ctx).
		Error()
}
