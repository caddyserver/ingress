## Requirements

 - A running kubernetes cluster (if you don't have one see *Setup a local cluster* section)
 - [skaffold](https://skaffold.dev/) installed on your machine
 - [helm 3](https://helm.sh/) installed on your machine
 - [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl/) installed on your machine

### Setup a local cluster

 - You need a machine with [docker](https://docker.io) up & running
 - If you are using Docker Desktop enable the kubernetes cluster from the config (see [docs](https://docs.docker.com/desktop/kubernetes/) here)

Otherwise you can decide to use [kind]() or [minukube]()

## Setup development env

Start `skaffold` using the command:
```
make dev
```

this will automatically:

 - build your docker image every time you change some code
 - update the helm release every time you change the helm chart
 - expose the caddy ingress controller (port 8080 and 8443)

You can test that all work as expected with:
```
curl -D - -s -k https://example1.kubernetes.localhost:8443/hello1 --resolve example1.kubernetes.localhost:8443:127.0.0.1
curl -D - -s -k https://example1.kubernetes.localhost:8443/hello2 --resolve example1.kubernetes.localhost:8443:127.0.0.1
curl -D - -s -k https://example2.kubernetes.localhost:8443/hello1 --resolve example2.kubernetes.localhost:8443:127.0.0.1
curl -D - -s -k https://example2.kubernetes.localhost:8443/hello2 --resolve example2.kubernetes.localhost:8443:127.0.0.1
```

You can change domains defined in `kuberentes/sample` folder with some domain that are risolved on your local machine or you can add them in the `/etc/host` file to be automatically resolved as localhost.

## Notes

 - You can change local port forwarded by skaffold by changing the port values in the `skaffold.yaml` file on section `portForward` `localPort`. Remind that you can forward only port greater than 1024 if you execute it as non-root user
 - We use an internal CA for test to simplify the installation, but then you can decide to use external CA for tests, see the `values.yaml` in the helm chart for the info on how to configure it.

## Releasing new helm chart version

If you want to release a new version of the `caddy-ingress-controller` chart, you'll need
to create a new PR with:
- The new chart's `version` in `Chart.yaml`
- The new `image.tag` in `values.yaml` (if you want to update the default image used in the chart)
- The new `appVersion` in `Chart.yaml` (if you did the previous line)

## Releasing a new app version

To release a new caddy-ingress-controller image, you need to create a new semver tag.
It will build and push an image to https://hub.docker.com/r/caddy/ingress.
