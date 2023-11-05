package global

import (
	"testing"

	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/caddyserver/ingress/pkg/store"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"

	"github.com/stretchr/testify/assert"
)

func TestIngressTlsSkipCertificates(t *testing.T) {
	testCases := []struct {
		desc                string
		skippedCertsDomains []string
		ingresses           []*networkingv1.Ingress
	}{
		{
			desc:                "No ingress registered",
			skippedCertsDomains: []string{},
			ingresses:           []*networkingv1.Ingress{},
		},
		{
			desc:                "One ingress registered with certificate with one domain",
			skippedCertsDomains: []string{"domain1.tld"},
			ingresses: []*networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID("first"),
					},
					Spec: networkingv1.IngressSpec{
						TLS: []networkingv1.IngressTLS{{
							Hosts: []string{"domain1.tld"},
						}},
					},
				},
			},
		},
		{
			desc:                "One ingress registered with certificate with multiple domains",
			skippedCertsDomains: []string{"domain1.tld", "domain2.tld", "domain3.tld"},
			ingresses: []*networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID("first"),
					},
					Spec: networkingv1.IngressSpec{
						TLS: []networkingv1.IngressTLS{{
							Hosts: []string{"domain1.tld", "domain2.tld", "domain3.tld"},
						}},
					},
				},
			},
		},
		{
			desc:                "Two ingress registered with certificate one domain each",
			skippedCertsDomains: []string{"domain1.tld", "domain2.tld"},
			ingresses: []*networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID("first"),
					},
					Spec: networkingv1.IngressSpec{
						TLS: []networkingv1.IngressTLS{{
							Hosts: []string{"domain1.tld"},
						}},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID("second"),
					},
					Spec: networkingv1.IngressSpec{
						TLS: []networkingv1.IngressTLS{{
							Hosts: []string{"domain2.tld"},
						}},
					},
				},
			},
		},
		{
			desc:                "Two ingress registered with certificate the same domain",
			skippedCertsDomains: []string{"domain1.tld"},
			ingresses: []*networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID("first"),
					},
					Spec: networkingv1.IngressSpec{
						TLS: []networkingv1.IngressTLS{{
							Hosts: []string{"domain1.tld"},
						}},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID("second"),
					},
					Spec: networkingv1.IngressSpec{
						TLS: []networkingv1.IngressTLS{{
							Hosts: []string{"domain1.tld"},
						}},
					},
				},
			},
		},
		{
			desc:                "Two ingress registered with certificate with multiple domains each",
			skippedCertsDomains: []string{"domain1a.tld", "domain1b.tld", "domain2a.tld", "domain2b.tld"},
			ingresses: []*networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID("first"),
					},
					Spec: networkingv1.IngressSpec{
						TLS: []networkingv1.IngressTLS{{
							Hosts: []string{"domain1a.tld", "domain1b.tld"},
						}},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID("second"),
					},
					Spec: networkingv1.IngressSpec{
						TLS: []networkingv1.IngressTLS{{
							Hosts: []string{"domain2a.tld", "domain2b.tld"},
						}},
					},
				},
			},
		},
		{
			desc:                "Two ingress registered with certificate with multiple domains each and partial domain overlap",
			skippedCertsDomains: []string{"domain1.tld", "domain2a.tld", "domain2b.tld"},
			ingresses: []*networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID("first"),
					},
					Spec: networkingv1.IngressSpec{
						TLS: []networkingv1.IngressTLS{{
							Hosts: []string{"domain1.tld", "domain2a.tld"},
						}},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID("second"),
					},
					Spec: networkingv1.IngressSpec{
						TLS: []networkingv1.IngressTLS{{
							Hosts: []string{"domain1.tld", "domain2b.tld"},
						}},
					},
				},
			},
		},
		{
			desc:                "One ingress registered without certificate",
			skippedCertsDomains: []string{},
			ingresses: []*networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID("first"),
					},
					Spec: networkingv1.IngressSpec{},
				},
			},
		},
		{
			desc:                "Two ingresses registered without certificate",
			skippedCertsDomains: []string{},
			ingresses: []*networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID("first"),
					},
					Spec: networkingv1.IngressSpec{},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID("second"),
					},
					Spec: networkingv1.IngressSpec{},
				},
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			tp := TLSPlugin{}
			c := converter.NewConfig()
			s := store.NewStore(store.Options{}, &store.PodInfo{})

			for _, ing := range tC.ingresses {
				s.AddIngress(ing)
			}

			tp.GlobalHandler(c, s)

			toSkip := c.GetHTTPServer().AutoHTTPS.SkipCerts
			assert.ElementsMatch(t, toSkip, tC.skippedCertsDomains, "List of certificate to skip don't match expectation")
		})
	}
}
