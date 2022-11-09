package util

import (
	"context"
	"errors"
	"fmt"

	"github.com/jeewangue/postgres-indb-operator/api/v1alpha1"
	ctlerrors "github.com/jeewangue/postgres-indb-operator/internal/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// ErrNoValue indicates that a resource could not be resolved to a value.
	ErrNoValue    = ctlerrors.NewInvalid(errors.New("no value"))
	ErrNotFound   = ctlerrors.NewTemporary(ctlerrors.NewInvalid(errors.New("not found")))
	ErrUnknownKey = ctlerrors.NewTemporary(ctlerrors.NewInvalid(errors.New("unknown key")))
)

// ResourceValue returns the value of a ResourceVar in a specific namespace.
func ResourceValue(client client.Client, resource v1alpha1.ResourceVar, namespace string) (string, error) {
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
			return "", ErrNotFound
		}
		return "", err
	}
	secretData, ok := secret.Data[key]
	if !ok {
		return "", ErrUnknownKey
	}
	return string(secretData), nil
}

func ConfigMapValue(client client.Client, namespacedName types.NamespacedName, key string) (string, error) {
	configMap := &corev1.ConfigMap{}
	err := client.Get(context.TODO(), namespacedName, configMap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	data, ok := configMap.Data[key]
	if !ok {
		return "", ErrUnknownKey
	}
	return string(data), nil
}
