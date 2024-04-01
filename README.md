# Jokati

![GitHub go.mod Go version (subdirectory of monorepo)](https://img.shields.io/github/go-mod/go-version/freepik-company/tekton-exporter)
![GitHub](https://img.shields.io/github/license/freepik-company/tekton-exporter)

![YouTube Channel Subscribers](https://img.shields.io/youtube/channel/subscribers/UCeSb3yfsPNNVr13YsYNvCAw?label=achetronic&link=http%3A%2F%2Fyoutube.com%2Fachetronic)
![GitHub followers](https://img.shields.io/github/followers/achetronic?label=achetronic&link=http%3A%2F%2Fgithub.com%2Fachetronic)
![X (formerly Twitter) Follow](https://img.shields.io/twitter/follow/achetronic?style=flat&logo=twitter&link=https%3A%2F%2Ftwitter.com%2Fachetronic)

Kubernetes operator to watch groups of resources and send notifications if conditions are met (realtime)

## Motivation

As you probably know, on Prometheus centered monitoring systems,
the alerts are commonly managed using PrometheusRule resources as they are straightforward 
to understand and configure.

These alerts are triggered based on PromQL queries results, but PromQL is quite limited to 
what it can evaluate or compare, and in some scenarios, something more advanced is needed.
For example, PromQL cannot compare labels as they were values.

What if it is possible to watch a group of resources directly from Kubernetes and
send notifications when something happens?

What if this 'something' could be templated with all the functionalities you already know from Helm
to craft really complex or powerful conditions?

This is exactly our proposal

## Prerequisites

- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### Development purposes

- go version v1.21.0+
- docker version 17.03+.
- kubebuilder version v3.14.0 (Only for development)

## Flags

Some configuration parameters can be defined by flags that can be passed to the controller.
They are described in the following table:

| Name                          | Description                                                        | Default Example |                                       |
|:------------------------------|:-------------------------------------------------------------------|:---------------:|---------------------------------------|
| `--config`                    | The path to configuration file                                     |  `jokati.yaml`  | `--config "./jokati.yaml"`            |
| `--kubeconfig`                | Path to kubeconfig                                                 |       `-`       | `--kubeconfig="~/.kube/config"`       |   
| `--enable-http2`              | If set, HTTP/2 will be enabled for the metrics and webhook servers |     `false`     | `--enable-http2 true`                 |
| `--metrics-secure`            | If set the metrics endpoint is served securely                     |     `false`     | `--metrics-secure true`               |
| `--leader-elect`              | Enable leader election for controller manager                      |     `false`     | `--leader-elect true`                 |
| `--health-probe-bind-address` | The address the probe endpoint binds to                            |       `-`       | `--health-probe-bind-address ":8091"` |
| `--metrics-bind-address`      | The address the metric endpoint binds to                           |     `:8080`     | `--metrics-bind-address ":8090"`      |

## Config

There are some parameters that must be configured through the config file. 
For instance, those related to integrations, that is how the messages are sent to your monitoring systems.

```yaml
integrations:

  # (Optional) Configuration parameters to be able to connect with generic webhooks
  webhook:
    url: "https://${WEBHOOK_TEST_USERNAME}:${WEBHOOK_TEST_PASSWORD}@webhook.site/98f1c771-bfaf-4c4f-81f6-f11c76684fcf"
    headers:
      X-Scope-OrgID: your-company

  # (Optional) Configuration parameters to be able to connect with Alertmanager
  alertmanager:
    url: "https://${TEST_USERNAME}:${TEST_PASSWORD}@webhook.site/d0fd5417-8931-476c-ae6e-c41eda3682af"
    headers:
      X-Scope-OrgID: freepik-company
```

As you can see in the previous example, we expand environment variables passed to the controller for configuration file. 
This way, you can manage credentials in your desired way (this applies to everything, headers included)

## RBAC

We designed the operator to be able to watch any kind of resource in a Kubernetes cluster, but by design, Kubernetes
permissions are always only additive.

This means that we had to grant only some resources to be watched by default, such as Secrets and ConfigMaps.
But you can watch other kinds of resources just granting some permissions to the
ServiceAccount of the controller as follows:

```yaml
# clusterRole-jokati-custom-resources.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: jokati-custom-resources
rules:
  - apiGroups:
      - "*"
    resources:
      - "*"
    verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
---
# clusterRoleBinding-jokati-custom-resources.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: jokati-custom-resources
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: jokati-custom-resources
subjects:
  - kind: ServiceAccount
    name: jokati-controller-manager
    namespace: default
---
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: jokati

resources:
  - https://github.com/freepik-company/jokati//deploy/?ref=main

  # Add your custom resources
  - clusterRole-jokati-custom-resources.yaml
  - clusterRoleBinding-jokati-custom-resources.yaml
```

## Example

To watch resources using this operator, you will need to create a CR of kind Notification. 
You can find the spec samples for all the versions of the resource in the [examples directory](./config/samples)

You may prefer to learn directly from an example, so let's explain it watching a ConfigMap:

```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: testing
   data:
     TEST_VAR: "placeholder"
```

Now use a Notification CR to watch ConfigMap looking for changes. When the conditions are met, we want to 
send the notification to a webhook:

```yaml
apiVersion: jokati.freepik.com/v1alpha1
kind: Notification
metadata:
  name: notification-sample-simple
spec:

  # Resource type to be watched
  watch:
    group: ""
    version: v1
    resource: configmaps
    
    # Optional: It's possible to watch specific resources
    # name: testing
    # namespace: default

  conditions:
    - name: check-configmap-name
      # The 'key' field admits vitamin Golang templating (well known from Helm)
      # The result of this field will be compared with 'value' for equality
      key: |
        {{- $source := . -}}
        {{- printf "%s" $source.metadata.name -}}
      value: testing

  message:
    reason: "NameMatchedAlert"
    data: |
      {{- $source := . -}}
      {{- printf "Hi, I'm on fire: %s/%s" $source.metadata.namespace $source.metadata.name -}}
```

## Templating engine

### What you can use

We recommend keeping the scope of the Notification resources as small as possible.
At the same time, we wanted to craft a powerful engine to do whatever we needed. 
So we mixed several gears, from here and there, and got all the power of a wonderful toy.

At the end of this madness you are reading about, what you will notice is that you can use everything you
already know from [Helm Template](https://helm.sh/docs/chart_template_guide/functions_and_pipelines/)

### How to use collected data

When a watched resource triggers an event, we pass the whole manifest to all the conditions (and even to
the alert data). 

This means the manifest (as Go object) is available in the main scope `.`. You can retrieve it as follows:

This means that the objects can be accessed or stored in variables in the following way:
```yaml
apiVersion: jokati.freepik.com/v1alpha1
kind: Notification
metadata:
  name: notification-sample-simple
spec:
  .
  .
  .
  conditions:
  - name: check-configmap-name
    # The 'key' field admits vitamin Golang templating (well known from Helm)
    # The result of this field will be compared with 'value' for equality
    key: |
      {{- $source := . -}}
      {{- printf "%s" $source.metadata.name -}}
    value: testing
```

> Remember: with a big power comes big responsibility
> ```gotemplate
> {{- $source := . -}}
> ```

## How to develop

> We recommend you to use a development tool like [Kind](https://kind.sigs.k8s.io/) 
> or [Minikube](https://minikube.sigs.k8s.io/docs/start/)
> to launch a lightweight Kubernetes on your local machine for development purposes

For learning purposes, we will suppose you are going to use Kind. 
So the first step is to create a Kubernetes cluster on your local machine executing the following command:

```console
kind create cluster
```

Once you have launched a safe play place, execute the following command. 
It will install the custom resource definitions (CRDs) in the cluster configured 
in your ~/.kube/config file and run the Operator locally against the cluster:

```console
make install run
```

> Remember that your `kubectl` is pointing to your Kind cluster. 
> However, you should always review the context your kubectl CLI is pointing to

If you desire to run against your cluster using a custom configuration, run the controller this way
from the root directory of this project:

```console
 go run ./cmd/main.go --config ./your-path-to-jokati/config/samples/config/jokati.yaml
```

## How releases are created

Each release of this operator is done following several steps carefully in order not to break the things for anyone.
Reliability is important to us, so we automated all the process of launching a release.

For a better understanding of the process, the steps are described in the following recipe:

1. Test the changes on the code:

    ```console
    make test
    ```

   > A release is not done if this stage fails


2. Define the package information

    ```console
    export VERSION="0.0.1"
    export IMG="freepik-company/jokati:v$VERSION"
    ```

3. Generate and push the Docker image (published on Docker Hub).

    ```console
    make docker-build docker-push
    ```

4. Generate the manifests for deployments using Kustomize.

   > NOTE: The makefile target mentioned above generates an 'install.yaml'
   > file in the dist directory. This file contains all the resources built
   > with Kustomize, which are necessary to install this project without
   > its dependencies.

   ```console
    make build-installer
    ```

## How to deploy

1. Using the installer

   The operator can be installed just running kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

   ```sh
   kubectl apply -f https://raw.githubusercontent.com/freepik-company/jokati/<tag or branch>/dist/install.yaml
   ```

2. Using Helm

   > TBD in a future release


## How to collaborate

This project is done on top of [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder), so read 
about that project before collaborating. 

Of course, we are open to external collaborations for this project. 
For doing it, you must fork the repository, make your changes to the code and open a PR. 
The code will be reviewed and tested (always)

> We are developers and hate bad code. For that reason, we ask you the highest quality on each line of code to improve
> this project on each iteration.

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Special mention

This project was done using IDEs from JetBrains. They helped us to develop faster, so we recommend them a lot! 🤓

<img src="https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.png" alt="JetBrains Logo (Main) logo." width="150">
