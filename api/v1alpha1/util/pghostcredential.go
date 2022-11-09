package util

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jeewangue/postgres-indb-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetConnectionStringWithDatabase(h *v1alpha1.PgHostCredential, c client.Client, database string) (string, error) {
	host, err := ResourceValue(c, h.Spec.Host, h.Namespace)
	if err != nil {
		return "", err
	}

	user, err := ResourceValue(c, h.Spec.User, h.Namespace)
	if err != nil {
		return "", err
	}

	password, err := ResourceValue(c, h.Spec.Password, h.Namespace)
	if err != nil {
		return "", err
	}

	connStr := fmt.Sprintf("postgresql://%s:%s@%s/%s", user, url.QueryEscape(password), host, database)
	if h.Spec.Params != "" {
		connStr += fmt.Sprintf("?%s", h.Spec.Params)
	} else {
		// backwards compatibility
		connStr += "?sslmode=disable"
	}
	return connStr, nil
}

func GetConnectionString(h *v1alpha1.PgHostCredential, c client.Client) (string, error) {
	return GetConnectionStringWithDatabase(h, c, "template1")
}

func PgHostCredentials(c client.Client, namespace string) ([]v1alpha1.PgHostCredential, error) {
	var creds v1alpha1.PgHostCredentialList
	err := c.List(context.TODO(), &creds, client.InNamespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("get host credentials in namespace: %w", err)
	}
	return creds.Items, nil
}

func PgHostCredentialByName(c client.Client, namespace string, name string) (*v1alpha1.PgHostCredential, error) {
	cred := &v1alpha1.PgHostCredential{}
	err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, cred)
	if err != nil {
		return nil, fmt.Errorf("get a host credential by name (%s) in namespace (%s): %w", name, namespace, err)
	}
	return cred, nil
}
