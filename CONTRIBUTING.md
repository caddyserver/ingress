## Requirements

We will explain how to contribute to this project using a linux machine, in order to be able ot easly contribute you need:

 - A machine with a public IP in order to use let's encrypt (you can provision ad-hoc machine on any clud provider you use)
 - A domain that redirect to server IP
 - [kind](https://github.com/kubernetes-sigs/kind) (to create a development cluster)
 - [skaffold](https://skaffold.dev/) (to improve development experience)
 - [Docker HUB](https://hub.docker.com) account (to store your docker images)

## Setup a development cluster

We create a three node cluster (master plus two worker), we start to setup the configuration:

```bash
cat <<EOF >> cluster.yml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
EOF
```
than we create the cluster
```bash
kind create cluster --config=cluster.yml
```
and activate the `kubectl` config via:
```
kind export kubeconfig
```

## Configure your docker credentials

Authenticate your docker intance:
```
docker login
```

## Setup development env

Replace the docker image you are going to use in `kubernetes/generated/deployment.yaml` and `skaffold.yaml` replacing `MYACCOUNT` with your Docker Hub account in `docker.io/MYACCOUNT/caddy-ingress-controller`

Replace also the domain name to use in `hack/test/example-ingress.yaml` from `MYDOMAIN.TDL` to your domain (ensore also that the subdomain `example1` and `example2` are resolved to the server public IP)

Than we can start skaffold using:
```
skaffold dev --port-forward
```
this will automatically:
 - build your docker image every time you change some code
 - update kubernetes config every time you change some file
 - expose the caddy ingress controller (port 80 and 443) on publc server
