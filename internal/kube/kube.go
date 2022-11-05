package kube

import (
	"context"
	"errors"
	"fmt"

	api "github.com/jeewangue/postgres-indb-operator/api/v1alpha1"
	ctlerrors "github.com/jeewangue/postgres-indb-operator/internal/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// ErrNoValue indicates that a resource could not be resolved to a value.
	ErrNoValue    = ctlerrors.NewInvalid(errors.New("no value"))
	errNotFound   = ctlerrors.NewTemporary(ctlerrors.NewInvalid(errors.New("not found")))
	errUnknownKey = ctlerrors.NewTemporary(ctlerrors.NewInvalid(errors.New("unknown key")))
)

// ResourceValue returns the value of a ResourceVar in a specific namespace.
func ResourceValue(client client.Client, resource api.ResourceVar, namespace string) (string, error) {
	if resource.Value != "" {
		return resource.Value, nil
	}

	if resource.ValueFrom != nil && resource.ValueFrom.SecretKeyRef != nil && resource.ValueFrom.SecretKeyRef.Key != "" {
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      resource.ValueFrom.SecretKeyRef.Name,
		}
		key := resource.ValueFrom.SecretKeyRef.Key
		v, err := SecretValue(client, namespacedName, key)
		if err != nil {
			return "", fmt.Errorf("secret %s key %s: %w", namespacedName, key, err)
		}
		return v, nil
	}

	if resource.ValueFrom != nil && resource.ValueFrom.ConfigMapKeyRef != nil && resource.ValueFrom.ConfigMapKeyRef.Key != "" {
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      resource.ValueFrom.ConfigMapKeyRef.Name,
		}
		key := resource.ValueFrom.ConfigMapKeyRef.Key
		v, err := ConfigMapValue(client, namespacedName, key)
		if err != nil {
			return "", fmt.Errorf("config map %s key %s: %w", namespacedName, key, err)
		}
		return v, nil
	}

	return "", ErrNoValue
}

func SecretValue(client client.Client, namespacedName types.NamespacedName, key string) (string, error) {
	secret := &corev1.Secret{}
	err := client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", errNotFound
		}
		return "", err
	}
	secretData, ok := secret.Data[key]
	if !ok {
		return "", errUnknownKey
	}
	return string(secretData), nil
}

func ConfigMapValue(client client.Client, namespacedName types.NamespacedName, key string) (string, error) {
	configMap := &corev1.ConfigMap{}
	err := client.Get(context.TODO(), namespacedName, configMap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", errNotFound
		}
		return "", err
	}
	data, ok := configMap.Data[key]
	if !ok {
		return "", errUnknownKey
	}
	return string(data), nil
}

func PgHostCredentials(c client.Client, namespace string) ([]api.PgHostCredential, error) {
	var creds api.PgHostCredentialList
	err := c.List(context.TODO(), &creds, client.InNamespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("get host credentials in namespace: %w", err)
	}
	return creds.Items, nil
}

func PgHostCredentialByName(c client.Client, namespace string, name string) (*api.PgHostCredential, error) {
	cred := &api.PgHostCredential{}
	err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, cred)
	if err != nil {
		return nil, fmt.Errorf("get a host credential by name (%s) in namespace (%s): %w", name, namespace, err)
	}
	return cred, nil
}

func PgDatabases(c client.Client, namespace string) ([]api.PgDatabase, error) {
	var databases api.PgDatabaseList
	err := c.List(context.TODO(), &databases, client.InNamespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("get databases in namespace: %w", err)
	}
	return databases.Items, nil
}

func PgDatabaseByName(c client.Client, namespace string, name string) (*api.PgDatabase, error) {
	database := &api.PgDatabase{}
	err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, database)
	if err != nil {
		return nil, fmt.Errorf("get a database by name (%s) in namespace (%s): %w", name, namespace, err)
	}
	return database, nil
}

func PgUsers(c client.Client, namespace string) ([]api.PgUser, error) {
	var users api.PgUserList
	err := c.List(context.TODO(), &users, client.InNamespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("get users in namespace: %w", err)
	}
	return users.Items, nil
}

func PgUserByName(c client.Client, namespace string, name string) (*api.PgUser, error) {
	user := &api.PgUser{}
	err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, user)
	if err != nil {
		return nil, fmt.Errorf("get a user by name (%s) in namespace (%s): %w", name, namespace, err)
	}
	return user, nil
}
