// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package propagator

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	"open-cluster-management.io/governance-policy-propagator/controllers/common"
)

// policyMapper looks at object and returns a slice of reconcile.Request to reconcile
// owners of object from label: policy.open-cluster-management.io/root-policy
func policyMapper(c client.Client) handler.MapFunc {
	return func(object client.Object) []reconcile.Request {
		// no work if policy has policy.open-cluster-management.io/experimental-controller-disable: "true" annotation
		if value, ok := object.GetAnnotations()[policiesv1.PolicyDisableAnnotationkey]; ok && value == "true" {
			log.V(2).Info("found a policy disable annotation in policy, skipping it", "policy", object.GetName())
			return nil
		}

		log := log.WithValues("name", object.GetName(), "namespace", object.GetNamespace())

		log.V(2).Info("Reconcile request for a policy")

		rootPlcName := object.GetLabels()[common.RootPolicyLabel]
		var name string
		var namespace string

		if rootPlcName != "" {
			// policy.open-cluster-management.io/root-policy exists, should be a replicated policy
			log.V(2).Info("Found reconciliation request from replicated policy")

			name = strings.Split(rootPlcName, ".")[1]
			namespace = strings.Split(rootPlcName, ".")[0]
			clusterList := &clusterv1.ManagedClusterList{}

			err := c.List(context.TODO(), clusterList, &client.ListOptions{})
			if err != nil {
				log.Error(err, "failed to list ManagedCluster objects")

				return nil
			}
			// do not handle a replicated policy which does not belong to the current cluster
			if !common.IsInClusterNamespace(object.GetNamespace(), clusterList.Items) {
				log.V(2).Info("Found a replicated policy in non-cluster namespace, skipping it")

				return nil
			}
		} else {
			// policy.open-cluster-management.io/root-policy doesn't exist, should be a root policy
			log.V(2).Info("Found reconciliation request from root policy")

			name = object.GetName()
			namespace = object.GetNamespace()
		}

		request := reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}}

		return []reconcile.Request{request}
	}
}
