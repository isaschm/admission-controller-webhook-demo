# Kubernetes Admission Controller Webhook Demo

This repository contains a small HTTP server that can be used as a Kubernetes
[MutatingAdmissionWebhook](https://kubernetes.io/docs/admin/admission-controllers/#mutatingadmissionwebhook-beta-in-19).

The logic of this demo webhook is fairly simple: it enforces more secure defaults for running
containers as non-root user. While it is still possible to run containers as root, the webhook
ensures that this is only possible if the setting `runAsNonRoot` is *explicitly* set to `false`
in the `securityContext` of the Pod. If no value is set for `runAsNonRoot`, a default of `true`
is applied, and the user ID defaults to `1234`.

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
    legalBasis: Article 2
    legitimateInterest: not present
    purposes: given
...
```

4. Deploy [a pod](examples/pod-with-override.yaml) that explicitly sets `runAsNonRoot` to `false`, allowing it to run as the
`root` user:
```
$ kubectl create -f examples/pod-with-override.yaml
$ kubectl get pod/pod-with-override -o yaml | grep securityContext -A 1
...
  securityContext:
    runAsNonRoot: false
...
$ kubectl logs pod-with-override
I am running as user 0
```

5. Attempt to deploy [a pod](examples/pod-with-conflict.yaml) that has a conflicting setting: `runAsNonRoot` set to `true`, but `runAsUser` set to 0 (root).
The admission controller should block the creation of that pod.
```
$ kubectl create -f examples/pod-with-conflict.yaml 
Error from server (InternalError): error when creating "examples/pod-with-conflict.yaml": Internal error
occurred: admission webhook "webhook-server.webhook-demo.svc" denied the request: runAsNonRoot specified,
but runAsUser set to 0 (the root user)
```

## Build the Image from Sources (optional)

An image can be built by running `make`.
If you want to modify the webhook server for testing purposes, be sure to set and export
the shell environment variable `IMAGE` to an image tag for which you have push access. You can then
build and push the image by running `make push-image`. Also make sure to change the image tag
in `deployment/deployment.yaml.template`, and if necessary, add image pull secrets.

