# Privacy-aware Kubernetes Admission Controller Webhook

This repository contains a small HTTP server that can be used as a Kubernetes
[MutatingAdmissionWebhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/).

## Prerequisites

A cluster on which this example can be tested must be running Kubernetes 1.24.0 or above,
with the `admissionregistration.k8s.io/v1` API enabled. You can verify that by observing that the
following command produces a non-empty output:
```
kubectl api-versions | grep admissionregistration.k8s.io/v1
```
In addition, the `MutatingAdmissionWebhook` admission controller should be added and listed in the admission-control
flag of `kube-apiserver`.

For building the image, [GNU make](https://www.gnu.org/software/make/) and [Go](https://golang.org) are required.

To issue and sign certificates, [cert-manager](https://cert-manager.io/) must be deployed to the cluster before the webhook. To deploy cert-manager, these steps can be followed:
```
$ kubectl create namespace cert-manager # cert-manager is the default namespace
$ kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.11.0/cert-manager.yaml
```
To verify the cert-manager api, run ```$cmctl check api```.

## Deploying the Webhook Server

1. Bring up a Kubernetes cluster satisfying the above prerequisites, and make
sure it is active (i.e., either via the configuration in the default location, or by setting
the `KUBECONFIG` environment variable).
2. Create the webhook namespace:

```
$ kubectl create namespace webhook-demo
```

3. Deploy all webhook resources:

```
$ kubectl apply -n webhook-demo -f deployment/deployment.yaml.template
```

## Verify

1. The `webhook-server` pod in the `webhook-demo` namespace should be running:
```
$ kubectl -n webhook-demo get pods
NAME                             READY     STATUS    RESTARTS   AGE
webhook-server-6f976f7bf-hssc9   1/1       Running   0          35m
```

2. A `MutatingWebhookConfiguration` named `demo-webhook` should exist:
```
$ kubectl get mutatingwebhookconfigurations | grep webhook-server
NAME           AGE
demo-webhook   36m
```

3. Deploy [a pod](examples/pod-with-information.yaml) that has all necessary transparency information:
```
$ kubectl create -f examples/pod-with-information.yaml
```
Verify that the pod has the transparency information in its annotations:
```
$ kubectl get pod/pod-with-information -o yaml | grep annotations -A 3
...
  annotations:
    purposes: given
    dataCategories: given
...
```

4. Deploy [a pod](examples/pod-with-override.yaml) that has some the transparency tags but is missing `dataCategories`:
```
$ kubectl create -f examples/pod-with-override.yaml
$ kubectl get pod/pod-with-override -o yaml | grep annotations -A 3
...
  annotations:
    purposes: given
    dataCategories: unspecified
...
```

5. Deploy [a pod](examples/pod-with-conflict.yaml) which disallows deployment outside of the Eu.
This only applies to GKE clusters.
```
$ kubectl create -f examples/pod-with-conflict.yaml
```
If the cluster is running in zones outside of the EU, the deployment will be rejected
and return an error. If not, the deployment will pass and can be verified with:
```
$ kubectl get pod/pod-with-conflict -o yaml | grep annotations -A 3
...
  annotations:
    purposes: given
    dataCategories: unspecified
...
```

## Build the Image from Sources (optional)

An image can be built by running `make`.
If you want to modify the webhook server for testing purposes, be sure to set and export
the shell environment variable `IMAGE` to an image tag for which you have push access. You can then
build and push the image by running `make push-image`. Also make sure to change the image tag
in `deployment/deployment.yaml.template`, and if necessary, add image pull secrets.

