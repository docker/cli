# Compose on Kubernetes

[![CircleCI](https://circleci.com/gh/docker/compose-on-kubernetes/tree/master.svg?style=svg)](https://circleci.com/gh/docker/compose-on-kubernetes/tree/master)

Compose on Kubernetes allows you to deploy Docker Compose files onto a
Kubernetes cluster.

# Table of contents

- [Get started](#get-started)
- [Developing Compose on Kubernetes](#developing-compose-on-kubernetes)

More documentation can be found in the [docs/](./docs) directory. This includes:
- [Architecture](./docs/architecture.md)
- [Mapping of stack to Kubernetes objects](./docs/mapping.md)
- [Compatibility matrix](./docs/compatibility.md)

# Get started

Compose on Kubernetes comes installed on
[Docker Desktop](https://www.docker.com/products/docker-desktop) and
[Docker Enterprise](https://www.docker.com/products/docker-enterprise).

On Docker Desktop you will need to activate Kubernetes in the settings to use
Compose on Kubernetes.

## Check that Compose on Kubernetes is installed

You can check that Compose on Kubernetes is installed by checking for the
availability of the API using the command:

```console
$ kubectl api-versions | grep compose
compose.docker.com/v1beta1
compose.docker.com/v1beta2
```

## Deploy a stack

To deploy a stack, you can use the Docker CLI:

```console
$ cat docker-compose.yml
version: '3.3'

services:

  db:
    build: db
    image: dockersamples/k8s-wordsmith-db

  words:
    build: words
    image: dockersamples/k8s-wordsmith-api
    deploy:
      replicas: 5

  web:
    build: web
    image: dockersamples/k8s-wordsmith-web
    ports:
     - "33000:80"

$ docker stack deploy --orchestrator=kubernetes -c docker-compose.yml hellokube
```

# Developing Compose on Kubernetes

See the [contributing](./CONTRIBUTING.md) guides for how to contribute code.

## Pre-requisites

- `make`
- [Docker Desktop](https://www.docker.com/products/docker-desktop) (Mac or Windows) with engine version 18.09 or later
- Enable Buildkit by setting `DOCKER_BUILDKIT=1` in your environment
- Enable Kubernetes in Docker Desktop settings

### For live debugging

- Debugger capable of remote debugging with Delve API version 2
  - Goland run-configs are pre-configured

## Debug quick start

### Debug install

To build and install a debug version of Compose on Kubernetes onto Docker
Desktop, you can use the following command:

```console
$ make -f debug.Makefile install-debug-images
```

This command:
- Builds the images with debug symbols
- Runs the debug installer:
  - Installs debug versions of API server and Compose controller in the `docker` namespace
  - Creates two debugging _LoadBalancer_ services (unused in this mode)

You can verify that Compose on Kubernetes is running with `kubectl` as follows:

```console
$ kubectl get all -n docker
NAME                               READY   STATUS    RESTARTS   AGE
pod/compose-7c4dfcff76-jgwst       1/1     Running   0          59s
pod/compose-api-759f8dbb4b-2z5n2   2/2     Running   0          59s

NAME                                      TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)           AGE
service/compose-api                       ClusterIP      10.98.42.151     <none>        443/TCP           59s
service/compose-api-server-remote-debug   LoadBalancer   10.101.198.179   localhost     40001:31693/TCP   59s
service/compose-controller-remote-debug   LoadBalancer   10.101.158.160   localhost     40000:31167/TCP   59s

NAME                          DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/compose       1         1         1            1           59s
deployment.apps/compose-api   1         1         1            1           59s

NAME                                     DESIRED   CURRENT   READY   AGE
replicaset.apps/compose-7c4dfcff76       1         1         1       59s
replicaset.apps/compose-api-759f8dbb4b   1         1         1       59s
```

If you describe one of the deployments, you should see `*-debug:latest` in the
image name.

### Live debugging install

To build and install a live debugging version of Compose on Kubernetes onto
Docker Desktop, you can use the following command:

```console
$ make -f debug.Makefile install-live-debug-images
```

This command:
- Builds the images with debug symbols
- Sets the image entrypoint to run a [Delve server](https://github.com/derekparker/delve)
- Runs the debug installer
  - Installs debug version of API server and Compose controller in the `docker` namespace
  - Creates two debugging _LoadBalancer_ services
    - `localhost:40000`: Compose controller
    - `localhost:40001`: API server
- The API server and Compose controller only start once a debugger is attached

To attach a debugger you have multiple options:
- Use [GoLand](https://www.jetbrains.com/go/): configuration can be found in `.idea` of the repository
  - Select the `Debug all` config, setup breakpoints and start the debugger
- Set your Delve compatible debugger to point to use `locahost:40000` and `localhost:40001`
  - Using a terminal: `dlv connect localhost:40000` then type `continue` and hit enter

To verify that the components are installed, you can use the following command:

```console
$ kubectl get all -n docker
```

To verify that the API server has started, ensure that it has started logging:
```console
$ kubectl logs -f -n docker deployment.apps/compose-api compose
API server listening at: [::]:40000
ERROR: logging before flag.Parse: I1207 15:25:13.760739      11 plugins.go:158] Loaded 2 mutating admission controller(s) successfully in the following order: NamespaceLifecycle,MutatingAdmissionWebhook.
ERROR: logging before flag.Parse: I1207 15:25:13.763211      11 plugins.go:161] Loaded 1 validating admission controller(s) successfully in the following order: ValidatingAdmissionWebhook.
ERROR: logging before flag.Parse: W1207 15:25:13.767429      11 client_config.go:552] Neither --kubeconfig nor --master was specified.  Using the inClusterConfig.  This might not work.
ERROR: logging before flag.Parse: W1207 15:25:13.851500      11 genericapiserver.go:319] Skipping API compose.docker.com/storage because it has no resources.
ERROR: logging before flag.Parse: I1207 15:25:13.998154      11 serve.go:116] Serving securely on [::]:9443
```

To verify that the Compose controller has started, ensure that it is logging:
```console
kubectl logs -f -n docker deployment.apps/compose
API server listening at: [::]:40000
Version:    v0.4.16-dirty
Git commit: b2e3a6b-dirty
OS/Arch:    linux/amd64
Built:      Fri Dec  7 15:18:13 2018
time="2018-12-07T15:25:19Z" level=info msg="Controller ready"
```

## Reinstall default

To reinstall the default Compose on Kubernetes on Docker Desktop, simply restart
your Kubernetes cluster. You can do this by deactivating and then reactivating
Kubernetes or by restarting Docker Desktop.
See the [contributing](./CONTRIBUTING.md) and [debugging](./DEBUGGING.md) guides.

# Deploying Compose on Kubernetes

- Guide for [Azure AKS](./docs/install-on-aks.md).
