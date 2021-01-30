## Requirements

 - A running kubernetes cluster (if you don't have one see *Setup a local cluster* section)
 - [helm 3](https://helm.sh/) installed on your machine
 - [skaffold](https://skaffold.dev/) installed on your machine

### Setup a local cluster

 - You need a machine with [docker](https://docker.io) up & running
 - You need to install [kind](https://kind.sigs.k8s.io/) on your machine

Than we can create a two nodes cluster (one master and one worker):

```bash
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
EOF
```
and activate the `kubectl` config via:
```
kind export kubeconfig
```

## Setup development env

Replace also the domain name to use in `kubernetes/sample/example-ingress.yaml` from `kubernetes.localhost` to your domain (ensure also that the subdomain `example1` and `example2` are resolved to the server public IP)

Create a namespace to host the caddy ingress controller:
```
kubectl create ns caddy-system
```

Than we can start skaffold using:
```
skaffold dev --port-forward
```

this will automatically:

 - build your docker image every time you change some code
 - update the helm release every time you change the helm chart
 - expose the caddy ingress controller (port 8080 and 8443)

You can test that all work as expected with:
```
curl -H 'Host: example1.kubernetes.localhost' http://127.0.0.1:80/hello1
curl -H 'Host: example1.kubernetes.localhost' http://127.0.0.1:80/hello2
curl -H 'Host: example2.kubernetes.localhost' http://127.0.0.1:80/hello1
curl -H 'Host: example2.kubernetes.localhost' http://127.0.0.1:80/hello2
```

## Notes

 - You can change local port forwarded by skaffold by changing the port values in the `skaffold.yaml` file on section `portForward` `localPort`. Remind that you can forward only port greather than 1024 if you execute it as non root user
 - You can delete your local cluster with the command `kind delete cluster`
 - To use TLS your domain should be publically resolved to your cluster IP in order to allow Let's Encript to validate the domain
