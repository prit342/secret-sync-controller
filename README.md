# secret-sync-controller
The **SecretSync Controller** is a Kubernetes controller that allows you to sync a `Secret` from a single source namespace into one or more target namespaces.
This is useful in multi-tenant or namespace-isolated environments where shared credentials need to be replicated automatically and safely.



## Custom Resource Definition (CRD)

A `SecretSync` custom resource looks like this:

```yaml
apiVersion: sync.example.com/v1alpha1
kind: SecretSync
metadata:
  name: sync-my-secret 
  namespace: default
spec:
  sourceName: my-secret ## the source secret that will be copied
  sourceNamespace: default ## the namespace where source secreted can be found
  targetNamespaces:
    - team-a # the target namespace where the source secret will be copied
    - team-b # the target namespace where the source secret will be copied
```
- In the above example, the `SecretSync` controller will copy the `my-secret` from the `default` namespace into the `team-a` and `team-b` namespaces.
- The controller will ensure that the `Secret` in the target namespaces is always in sync with the source `Secret`. If the source `Secret` is updated, the controller will automatically update the target `Secrets` as well.
- The name of the secret that will be created in the target namespaces will be the same as the source secret, i.e., `my-secret` in this case.


## Features
- One-to-many secret replication: Sync a single secret to multiple namespaces.

- Ownership checks: Ensures existing synced secrets are not overwritten unless they are managed by the same CR resource and controller

- Status reporting: Updates the CR’s .status with success or error messages and the last sync time.

- Finalizer-based cleanup: Automatically deletes target secrets on CR deletion.

- Event-driven updates: Reconciles when source secret is created, deleted or updated.

## Behavior by Scenario

### SecretSync is Created and Source Secret Exists
- The controller copies the secret from the source namespace to each target namespace.
- The copied secrets are labeled and annotated for ownership tracking.

### SecretSync is Created but Source Secret Does Not Exist
- The controller logs an error and updates the `.status` field.
- Once the source secret is created, the controller automatically retries and syncs it to the targets.

### Source Secret is Updated
- The controller detects the change and reconciles the SecretSync CR
- The updated data is copied to all target secrets (if they are managed by this CR).

### SecretSync CR is Deleted
- A finalizer ensures that all secrets synced by this CR are deleted from the target namespaces.
- After cleanup, the finalizer is removed, allowing Kubernetes to complete deletion.

### Secret Already Exists in Target Namespace
If the secret in the target namespace is already present:
- If managed by this CR (based on annotations), it is updated.
- If not managed by this CR, it is skipped and a warning is logged into the status field of the CR.

## Getting Started

### Prerequisites
- go version v1.24.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### Quick testing
- Create a cluster using `kind` or `minikube`. For example using `kind`:

```sh
kind create cluster --name secret-sync-cluster
```

- Install the controller using the provided `Makefile` commands. You can also run it locally using `make run`. Make sure you run this in a different terminal window.
- In a different terminal window, apply the sample `SecretSync` CRD provided in the `config/samples/` directory to test the functionality.
```bash
❯ kubectl apply -f config/crd/bases/sync.example.com_secretsyncs.yaml
customresourcedefinition.apiextensions.k8s.io/secretsyncs.sync.example.com created
```

- Create the source secret in the `default` namespace:
```
kubectl apply -f ./test-manifest/source-secret.yaml
```
- Create a custom CR to test the controller.
```
❯ kubectl apply -f ./test-manifest/cr.yaml
namespace/test1 created
namespace/test2 created
secretsync.sync.example.com/sync-my-secret created

❯ kubectl get secretsyncs -A
NAMESPACE   NAME             SYNCED   MESSAGE                                                           LASTTRANSITION
default     sync-my-secret   True     successfully synced secret my-secret to namespaces: test1,test2   2025-07-16T17:29:44Z
```
- To check what happens when you update the source secret, you can edit the `source-secret.yaml` file and change the data. For example, change the value of `key1` to `new-value1`, and then apply the changes:
```bash
kubectl patch secret my-secret -n default --type='merge' -p '{"data":{"key1":"bmV3LXZhbHVlMQo="}}'
```
- The controller will automatically detect the change and update the target secrets in `test1` and `test2` namespaces. You can verify this by checking the secrets in those namespaces:

- You can also check what happens when you try to create duplicate secrets by specifying the same source secret in multiple `SecretSync` CRs. The controller will log a warning and skip syncing those secrets if they are not managed by the same CR.

```bash
❯ kubectl apply -f test-manifest/duplicate-cr.yaml

❯ kubectl get secretsyncs -A
NAMESPACE   NAME                       SYNCED   MESSAGE                                                                                                                                       LASTTRANSITION
default     sync-my-secret             True     successfully synced secret my-secret to namespaces: test1,test2                                                                               2025-07-16T17:31:43Z
default     sync-my-secret-duplicate   False    failed to sync object: the secret my-secret already exists in namespace test1 and is not owned by this instance sync-my-secret-duplicate...   2025-07-16T17:32:50Z
```

- If you delete the CRs, the controller will automatically delete the target secrets in the specified namespaces:

```bash
❯ kubectl delete secretsyncs.sync.example.com sync-my-secret
secretsync.sync.example.com "sync-my-secret" deleted
❯ kubectl get secrets -n test1,test2
No resources found in test1,test2 namespace.
```
- If you delete the source secret, the controller will log an error and update the `.status` field of the `SecretSync` CR. Once the source secret is recreated, the controller will automatically sync it to the target namespaces again.

```bash
❯ kubectl apply -f ./test-manifest/
❯ kubectl delete secret my-secret
secret "my-secret" deleted

❯ kubectl get secretsyncs.sync.example.com
NAME                       SYNCED   MESSAGE                                                                                    LASTTRANSITION
sync-my-secret-duplicate   False    error reading source secret my-secret in namespace default: Secret "my-secret" not found   2025-07-16T17:38:34Z
```

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/secret-sync-controller:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/secret-sync-controller:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/secret-sync-controller:tag
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/secret-sync-controller/<tag or branch>/dist/install.yaml
```

### By providing a Helm Chart

1. Build the chart using the optional helm plugin

```sh
kubebuilder edit --plugins=helm/v1-alpha
```

2. See that a chart was generated under 'dist/chart', and users
can obtain this solution from there.

**NOTE:** If you change the project, you need to update the Helm Chart
using the same command above to sync the latest changes. Furthermore,
if you create webhooks, you need to use the above command with
the '--force' flag and manually ensure that any custom configuration
previously added to 'dist/chart/values.yaml' or 'dist/chart/manager/manager.yaml'
is manually re-applied afterwards.

.

