package util

import (
	"context"
	"fmt"

	"github.com/jeewangue/postgres-indb-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PgDatabases(c client.Client, namespace string) ([]v1alpha1.PgDatabase, error) {
	var databases v1alpha1.PgDatabaseList
	err := c.List(context.TODO(), &databases, client.InNamespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("get databases in namespace: %w", err)
	}
	return databases.Items, nil
}

func PgDatabaseByName(c client.Client, namespace string, name string) (*v1alpha1.PgDatabase, error) {
	database := &v1alpha1.PgDatabase{}
	err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, database)
	if err != nil {
		return nil, fmt.Errorf("get a database by name (%s) in namespace (%s): %w", name, namespace, err)
	}
	return database, nil
}
