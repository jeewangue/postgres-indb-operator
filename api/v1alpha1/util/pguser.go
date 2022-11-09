package util

import (
	"context"
	"fmt"

	"github.com/jeewangue/postgres-indb-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PgUsers(c client.Client, namespace string) ([]v1alpha1.PgUser, error) {
	var users v1alpha1.PgUserList
	err := c.List(context.TODO(), &users, client.InNamespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("get users in namespace: %w", err)
	}
	return users.Items, nil
}

func PgUserByName(c client.Client, namespace string, name string) (*v1alpha1.PgUser, error) {
	user := &v1alpha1.PgUser{}
	err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, user)
	if err != nil {
		return nil, fmt.Errorf("get a user by name (%s) in namespace (%s): %w", name, namespace, err)
	}
	return user, nil
}
