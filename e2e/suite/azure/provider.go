/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
limitations under the License.
*/
package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	kvauth "github.com/Azure/go-autorest/autorest/azure/auth"

	// nolint
	. "github.com/onsi/ginkgo"

	// nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/pointer"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/e2e/framework"
)

type azureProvider struct {
	clientID     string
	clientSecret string
	tenantID     string
	vaultURL     string
	client       *keyvault.BaseClient
	framework    *framework.Framework
}

func newazureProvider(f *framework.Framework, clientID, clientSecret, tenantID, vaultURL string) *azureProvider {
	clientCredentialsConfig := kvauth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	clientCredentialsConfig.Resource = "https://vault.azure.net"
	authorizer, err := clientCredentialsConfig.Authorizer()
	Expect(err).ToNot(HaveOccurred())
	basicClient := keyvault.New()
	basicClient.Authorizer = authorizer

	prov := &azureProvider{
		framework:    f,
		clientID:     clientID,
		clientSecret: clientSecret,
		tenantID:     tenantID,
		vaultURL:     vaultURL,
		client:       &basicClient,
	}
	BeforeEach(prov.BeforeEach)
	return prov
}

func (s *azureProvider) CreateSecret(key, val string) {
	_, err := s.client.SetSecret(
		context.Background(),
		s.vaultURL,
		key,
		keyvault.SecretSetParameters{
			Value: &val,
			SecretAttributes: &keyvault.SecretAttributes{
				RecoveryLevel: keyvault.Purgeable,
				Enabled:       utilpointer.BoolPtr(true),
			},
		})
	Expect(err).ToNot(HaveOccurred())
}

func (s *azureProvider) DeleteSecret(key string) {
	_, err := s.client.DeleteSecret(
		context.Background(),
		s.vaultURL,
		key)
	Expect(err).ToNot(HaveOccurred())
}

func (s *azureProvider) BeforeEach() {
	azureCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "provider-secret",
			Namespace: s.framework.Namespace.Name,
		},
		StringData: map[string]string{
			"client-id":     s.clientID,
			"client-secret": s.clientSecret,
		},
	}
	err := s.framework.CRClient.Create(context.Background(), azureCreds)
	Expect(err).ToNot(HaveOccurred())

	secretStore := &esv1alpha1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.framework.Namespace.Name,
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				AzureKV: &esv1alpha1.AzureKVProvider{
					TenantID: &s.tenantID,
					VaultURL: &s.vaultURL,
					AuthSecretRef: &esv1alpha1.AzureKVAuth{
						ClientID: &esmeta.SecretKeySelector{
							Name: "provider-secret",
							Key:  "client-id",
						},
						ClientSecret: &esmeta.SecretKeySelector{
							Name: "provider-secret",
							Key:  "client-secret",
						},
					},
				},
			},
		},
	}
	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}
