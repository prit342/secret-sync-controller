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

