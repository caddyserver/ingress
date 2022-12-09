package storage

import (
	"context"
	"fmt"
	"io/fs"
	"regexp"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/certmagic"
	"github.com/google/uuid"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	leaseDuration      = 5 * time.Second
	leaseRenewInterval = 2 * time.Second
	leasePollInterval  = 5 * time.Second
	leasePrefix        = "caddy-lock-"

	keyPrefix = "caddy.ingress--"
)

// matchLabels are attached to each resource so that they can be found in the future.
var matchLabels = map[string]string{
	"manager": "caddy",
}

// specialChars is a regex that matches all special characters except '-'.
var specialChars = regexp.MustCompile("[^\\da-zA-Z-]+")

// cleanKey strips all special characters that are not supported by kubernetes names and converts them to a '.'.
// sequences like '.*.' are also converted to a single '.'.
func cleanKey(key string, prefix string) string {
	return prefix + specialChars.ReplaceAllString(key, ".")
}

// SecretStorage facilitates storing certificates retrieved by certmagic in kubernetes secrets.
type SecretStorage struct {
	Namespace string
	LeaseId   string

	kubeClient *kubernetes.Clientset
	logger     *zap.Logger
}

func init() {
	caddy.RegisterModule(SecretStorage{})
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

	s.logger = ctx.Logger(s)
	s.kubeClient = clientset
	if s.LeaseId == "" {
		s.LeaseId = uuid.New().String()
	}
	return nil
}

// CertMagicStorage returns a certmagic storage type to be used by caddy.
func (s *SecretStorage) CertMagicStorage() (certmagic.Storage, error) {
	return s, nil
}

// Exists returns true if key exists in fs.
func (s *SecretStorage) Exists(ctx context.Context, key string) bool {
	s.logger.Debug("finding secret", zap.String("name", key))
	secrets, err := s.kubeClient.CoreV1().Secrets(s.Namespace).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%v", cleanKey(key, keyPrefix)),
	})

	if err != nil {
		return false
	}

	var found bool
	for _, i := range secrets.Items {
		if i.ObjectMeta.Name == cleanKey(key, keyPrefix) {
			found = true
			break
		}
	}

	return found
}

// Store saves value at key. More than certs and keys are stored by certmagic in secrets.
func (s *SecretStorage) Store(ctx context.Context, key string, value []byte) error {
	se := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   cleanKey(key, keyPrefix),
			Labels: matchLabels,
		},
		Data: map[string][]byte{
			"value": value,
		},
	}

	var err error
	if s.Exists(ctx, key) {
		s.logger.Debug("creating secret", zap.String("name", key))
		_, err = s.kubeClient.CoreV1().Secrets(s.Namespace).Update(context.TODO(), &se, metav1.UpdateOptions{})
	} else {
		s.logger.Debug("updating secret", zap.String("name", key))
		_, err = s.kubeClient.CoreV1().Secrets(s.Namespace).Create(context.TODO(), &se, metav1.CreateOptions{})
	}

	if err != nil {
		return err
	}

	return nil
}

// Load retrieves the value at the given key.
func (s *SecretStorage) Load(ctx context.Context, key string) ([]byte, error) {
	secret, err := s.kubeClient.CoreV1().Secrets(s.Namespace).Get(context.TODO(), cleanKey(key, keyPrefix), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fs.ErrNotExist
		}
		return nil, err
	}

	s.logger.Debug("loading secret", zap.String("name", key))
	return secret.Data["value"], nil
}

// Delete deletes the value at the given key.
func (s *SecretStorage) Delete(ctx context.Context, key string) error {
	err := s.kubeClient.CoreV1().Secrets(s.Namespace).Delete(context.TODO(), cleanKey(key, keyPrefix), metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	s.logger.Debug("deleting secret", zap.String("name", key))
	return nil
}

// List returns all keys that match prefix.
func (s *SecretStorage) List(ctx context.Context, prefix string, recursive bool) ([]string, error) {
	var keys []string

	s.logger.Debug("listing secrets", zap.String("name", prefix))
	secrets, err := s.kubeClient.CoreV1().Secrets(s.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(matchLabels).String(),
	})
	if err != nil {
		return keys, err
	}

	// TODO :- do we need to handle the recursive flag?
	for _, secret := range secrets.Items {
		key := secret.ObjectMeta.Name
		if strings.HasPrefix(key, cleanKey(prefix, keyPrefix)) {
			keys = append(keys, strings.TrimPrefix(key, keyPrefix))
		}
	}

	return keys, err
}

// Stat returns information about key.
func (s *SecretStorage) Stat(ctx context.Context, key string) (certmagic.KeyInfo, error) {
	secret, err := s.kubeClient.CoreV1().Secrets(s.Namespace).Get(context.TODO(), cleanKey(key, keyPrefix), metav1.GetOptions{})
	if err != nil {
		return certmagic.KeyInfo{}, err
	}

	s.logger.Debug("stats secret", zap.String("name", key))

	return certmagic.KeyInfo{
		Key:        key,
		Modified:   secret.GetCreationTimestamp().UTC(),
		Size:       int64(len(secret.Data["value"])),
		IsTerminal: false,
	}, nil
}

func (s *SecretStorage) Lock(ctx context.Context, key string) error {
	for {
		_, err := s.tryAcquireOrRenew(ctx, cleanKey(key, leasePrefix), false)
		if err == nil {
			go s.keepLockUpdated(ctx, cleanKey(key, leasePrefix))
			return nil
		}

		select {
		case <-time.After(leasePollInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *SecretStorage) keepLockUpdated(ctx context.Context, key string) {
	for {
		time.Sleep(leaseRenewInterval)
		done, err := s.tryAcquireOrRenew(ctx, key, true)
		if err != nil {
			return
		}
		if done {
			return
		}
	}
}

func (s *SecretStorage) tryAcquireOrRenew(ctx context.Context, key string, shouldExist bool) (bool, error) {
	now := metav1.Now()
	lock := resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      key,
			Namespace: s.Namespace,
		},
		Client: s.kubeClient.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: s.LeaseId,
		},
	}

	ler := resourcelock.LeaderElectionRecord{
		HolderIdentity:       lock.Identity(),
		LeaseDurationSeconds: 5,
		AcquireTime:          now,
		RenewTime:            now,
	}

	currLer, _, err := lock.Get(ctx)

	// 1. obtain or create the ElectionRecord
	if err != nil {
		if !errors.IsNotFound(err) {
			return true, err
		}
		if shouldExist {
			return true, nil // Lock has been released
		}
		if err = lock.Create(ctx, ler); err != nil {
			return true, err
		}
		return false, nil
	}

	// 2. Record obtained, check the Identity & Time
	if currLer.HolderIdentity != "" &&
		currLer.RenewTime.Add(leaseDuration).After(now.Time) &&
		currLer.HolderIdentity != lock.Identity() {
		return true, fmt.Errorf("lock is held by %v and has not yet expired", currLer.HolderIdentity)
	}

	// 3. We're going to try to update the existing one
	if currLer.HolderIdentity == lock.Identity() {
		ler.AcquireTime = currLer.AcquireTime
		ler.LeaderTransitions = currLer.LeaderTransitions
	} else {
		ler.LeaderTransitions = currLer.LeaderTransitions + 1
	}

	if err = lock.Update(ctx, ler); err != nil {
		return true, fmt.Errorf("failed to update lock: %v", err)
	}
	return false, nil
}

func (s *SecretStorage) Unlock(ctx context.Context, key string) error {
	err := s.kubeClient.CoordinationV1().Leases(s.Namespace).Delete(context.TODO(), cleanKey(key, leasePrefix), metav1.DeleteOptions{})
	return err
}
