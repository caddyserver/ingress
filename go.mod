module bitbucket.org/lightcodelabs/ingress

go 1.12

require (
	bitbucket.org/lightcodelabs/caddy2 v0.0.0-00010101000000-000000000000
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/mholt/certmagic v0.5.1
	github.com/pkg/errors v0.8.1
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/grpc v1.20.1 // indirect
	gopkg.in/go-playground/pool.v3 v3.1.1
	k8s.io/api v0.0.0-20190503110853-61630f889b3c
	k8s.io/apimachinery v0.0.0-20190502092502-a44ef629a3c9
	k8s.io/client-go v0.0.0-20190425172711-65184652c889
	k8s.io/cloud-provider v0.0.0-20190503112208-4f570a5e5694 // indirect
	k8s.io/klog v0.3.0
	k8s.io/kubernetes v1.14.1
	k8s.io/utils v0.0.0-20190506122338-8fab8cb257d5 // indirect
)

replace bitbucket.org/lightcodelabs/caddy2 => ../caddy2
