package storage

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/certmagic"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// matchLabels are attached to each resource so that they can be found in the future.
var matchLabels = map[string]string{
	"manager": "caddy",
}

// labelSelector is the search string that will return all secrets managed by the caddy ingress controller.
var labelSelector = "manager=caddy"

// specialChars is a regex that matches all special characters except '.' and '-'.
var specialChars = regexp.MustCompile("[^0-9a-zA-Z.-]+")

var keyPrefix = "caddy.ingress--"

// cleanKey strips all special characters that are not supported by kubernetes names and converts them to a '.'.
func cleanKey(key string) string {
	return keyPrefix + specialChars.ReplaceAllString(key, ".")
}

// SecretStorage facilitates storing certificates retrieved by certmagic in kubernetes secrets.
type SecretStorage struct {
	Namespace  string
	KubeClient *kubernetes.Clientset
}

func (SecretStorage) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "caddy.storage.secret_store",
		New: func() caddy.Module { return new(SecretStorage) },
	}
}

// Provisions the SecretStorage instance.
func (s *SecretStorage) Provision(ctx caddy.Context) error {
	config, _ := clientcmd.BuildConfigFromFlags("", "")
	// creates the clientset
	clientset, _ := kubernetes.NewForConfig(config)

	s.KubeClient = clientset
	return nil
}

// CertMagicStorage returns a certmagic storage type to be used by caddy.
func (s *SecretStorage) CertMagicStorage() (certmagic.Storage, error) {
	return s, nil
}

// Exists returns true if key exists in fs.
func (s *SecretStorage) Exists(key string) bool {
	secrets, err := s.KubeClient.CoreV1().Secrets(s.Namespace).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%v", cleanKey(key)),
	})

	if err != nil {
		return false
	}

	var found bool
	for _, i := range secrets.Items {
		if i.ObjectMeta.Name == cleanKey(key) {
			found = true
			break
		}
	}

	return found
}

// Store saves value at key. More than certs and keys are stored by certmagic in secrets.
func (s *SecretStorage) Store(key string, value []byte) error {
	se := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   cleanKey(key),
			Labels: matchLabels,
		},
		Data: map[string][]byte{
			"value": value,
		},
	}

	var err error
	if s.Exists(key) {
		_, err = s.KubeClient.CoreV1().Secrets(s.Namespace).Update(context.TODO(), &se, metav1.UpdateOptions{})
	} else {
		_, err = s.KubeClient.CoreV1().Secrets(s.Namespace).Create(context.TODO(), &se, metav1.CreateOptions{})
	}

	if err != nil {
		return err
	}

	return nil
}

// Load retrieves the value at the given key.
func (s *SecretStorage) Load(key string) ([]byte, error) {
	secret, err := s.KubeClient.CoreV1().Secrets(s.Namespace).Get(context.TODO(), cleanKey(key), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return secret.Data["value"], nil
}

// Delete deletes the value at the given key.
func (s *SecretStorage) Delete(key string) error {
	err := s.KubeClient.CoreV1().Secrets(s.Namespace).Delete(context.TODO(), cleanKey(key), metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

// List returns all keys that match prefix.
func (s *SecretStorage) List(prefix string, recursive bool) ([]string, error) {
	var keys []string

	secrets, err := s.KubeClient.CoreV1().Secrets(s.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return keys, err
	}

	// TODO :- do we need to handle the recursive flag?
	for _, secret := range secrets.Items {
		key := secret.ObjectMeta.Name
		if strings.HasPrefix(key, cleanKey(prefix)) {
			keys = append(keys, strings.TrimPrefix(key, keyPrefix))
		}
	}

	return keys, err
}

// Stat returns information about key.
func (s *SecretStorage) Stat(key string) (certmagic.KeyInfo, error) {
	secret, err := s.KubeClient.CoreV1().Secrets(s.Namespace).Get(context.TODO(), cleanKey(key), metav1.GetOptions{})
	if err != nil {
		return certmagic.KeyInfo{}, err
	}

	return certmagic.KeyInfo{
		Key:        key,
		Modified:   secret.GetCreationTimestamp().UTC(),
		Size:       int64(len(secret.Data["value"])),
		IsTerminal: false,
	}, nil
}

func (s *SecretStorage) Lock(ctx context.Context, key string) error {
	// TODO: implement
	return nil
}

func (s *SecretStorage) Unlock(key string) error {
	// TODO: implement
	return nil
}
