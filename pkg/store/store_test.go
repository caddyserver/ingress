package store

import (
	"testing"

	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func TestStoreIngresses(t *testing.T) {
	tests := []struct {
		name            string
		addIngresses    []string
		removeIngresses []string
		expectCount     int
	}{
		{
			name:            "No ingress added nor removed",
			addIngresses:    []string{},
			removeIngresses: []string{},
			expectCount:     0,
		},
		{
			name:            "One ingress added, no ingress removed",
			addIngresses:    []string{"first"},
			removeIngresses: []string{},
			expectCount:     1,
		},
		{
			name:            "Two ingresses added, no ingress removed",
			addIngresses:    []string{"first", "second"},
			removeIngresses: []string{},
			expectCount:     2,
		},
		{
			name:            "Two ingresses added, first ingress removed",
			addIngresses:    []string{"first", "second"},
			removeIngresses: []string{"first"},
			expectCount:     1,
		},
		{
			name:            "Two ingresses added, second ingress removed",
			addIngresses:    []string{"first", "second"},
			removeIngresses: []string{"second"},
			expectCount:     1,
		},
		{
			name:            "Two ingresses added, both ingresses removed",
			addIngresses:    []string{"first", "second"},
			removeIngresses: []string{"second", "first"},
			expectCount:     0,
		},
		{
			name:            "Two ingresses added, one existing and one non existing ingress removed",
			addIngresses:    []string{"first", "second"},
			removeIngresses: []string{"second", "third"},
			expectCount:     1,
		},
		{
			name:            "Remove non existing ingresses",
			addIngresses:    []string{"first", "second"},
			removeIngresses: []string{"third", "forth"},
			expectCount:     2,
		},
		{
			name:            "Remove non existing ingresses from an empty list",
			addIngresses:    []string{},
			removeIngresses: []string{"first", "second"},
			expectCount:     0,
		},
		{
			name:            "Add the same ingress multiple time",
			addIngresses:    []string{"first", "first", "first"},
			removeIngresses: []string{},
			expectCount:     1,
		},
		{
			name:            "Remove the same ingress multiple time",
			addIngresses:    []string{"first", "second"},
			removeIngresses: []string{"second", "second", "second"},
			expectCount:     1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ingressCache := cache.NewIndexer(cache.MetaNamespaceKeyFunc, make(cache.Indexers))
			s, _ := NewStore(Options{}, "", &PodInfo{}, ingressCache)
			for _, uid := range test.addIngresses {
				i := createIngress(uid)
				ingressCache.Add(&i)
			}

			for _, uid := range test.removeIngresses {
				i := createIngress(uid)
				ingressCache.Delete(&i)
			}

			if test.expectCount != len(s.Ingresses()) {
				t.Errorf("Number of ingresses do not match expectation in %s: got %v, expected %v", test.name, len(s.Ingresses()), test.expectCount)
			}
		})
	}
}

func TestStoreReturnIfHasManagedTLS(t *testing.T) {
	tests := []struct {
		name      string
		ingresses []v1.Ingress
		expect    bool
	}{
		{
			name:      "No ingress",
			ingresses: []v1.Ingress{},
			expect:    false,
		},
		{
			name: "One ingress without certificate",
			ingresses: []v1.Ingress{
				createIngress("first"),
			},
			expect: false,
		},
		{
			name: "One ingress with certificate",
			ingresses: []v1.Ingress{
				createIngressTLS("first", []string{"host1"}, "mysecret1"),
			},
			expect: true,
		},
		{
			name: "Multiple ingresses without certificate",
			ingresses: []v1.Ingress{
				createIngress("first"),
				createIngress("second"),
				createIngress("third"),
			},
			expect: false,
		},
		{
			name: "Three ingresses mixed with and without certificate",
			ingresses: []v1.Ingress{
				createIngressTLS("first", []string{"host1a", "host1b"}, "mysecret1"),
				createIngress("second"),
				createIngressTLS("third", []string{"host3"}, "mysecret3"),
			},
			expect: true,
		},
		{
			name: "Ingress replaced without a certificate",
			ingresses: []v1.Ingress{
				createIngressTLS("first", []string{"host1a", "host1b"}, "mysecret1"),
				createIngress("first"),
			},
			expect: false,
		},
		{
			name: "Ingress replaced with a certificate",
			ingresses: []v1.Ingress{
				createIngress("first"),
				createIngressTLS("first", []string{"host1a", "host1b"}, "mysecret1"),
			},
			expect: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ingressCache := cache.NewIndexer(cache.MetaNamespaceKeyFunc, make(cache.Indexers))
			s, _ := NewStore(Options{}, "", &PodInfo{}, ingressCache)
			for _, i := range test.ingresses {
				ingressCache.Add(&i)
			}

			if test.expect != s.HasManagedTLS() {
				t.Errorf("managed TLS do not match expectation in %s: got %v, expected %v", test.name, s.HasManagedTLS(), test.expect)
			}
		})
	}
}

func createIngressTLS(name string, hosts []string, secret string) v1.Ingress {
	i := createIngress(name)

	i.Spec = v1.IngressSpec{
		TLS: []v1.IngressTLS{
			{
				Hosts:      hosts,
				SecretName: secret,
			},
		},
	}

	return i
}

func createIngress(name string) v1.Ingress {
	return v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
