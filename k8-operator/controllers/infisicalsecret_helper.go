package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/Infisical/infisical/k8-operator/api/v1alpha1"
	"github.com/Infisical/infisical/k8-operator/packages/model"
	"github.com/Infisical/infisical/k8-operator/packages/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const SERVICE_ACCOUNT_ACCESS_KEY = "serviceAccountAccessKey"
const SERVICE_ACCOUNT_PUBLIC_KEY = "serviceAccountPublicKey"
const SERVICE_ACCOUNT_PRIVATE_KEY = "serviceAccountPrivateKey"

const INFISICAL_MACHINE_IDENTITY_CLIENT_ID = "clientId"
const INFISICAL_MACHINE_IDENTITY_CLIENT_SECRET = "clientSecret"

const INFISICAL_TOKEN_SECRET_KEY_NAME = "infisicalToken"
const SECRET_VERSION_ANNOTATION = "secrets.infisical.com/version" // used to set the version of secrets via Etag
const OPERATOR_SETTINGS_CONFIGMAP_NAME = "infisical-config"
const OPERATOR_SETTINGS_CONFIGMAP_NAMESPACE = "infisical-operator-system"
const INFISICAL_DOMAIN = "https://app.infisical.com/api"

type AuthStrategyType string

var AuthStrategy = struct {
	SERVICE_TOKEN              AuthStrategyType
	SERVICE_ACCOUNT            AuthStrategyType
	UNIVERSAL_MACHINE_IDENTITY AuthStrategyType
	KUBERNETES                 AuthStrategyType
}{
	SERVICE_TOKEN:              "SERVICE_TOKEN",
	SERVICE_ACCOUNT:            "SERVICE_ACCOUNT",
	UNIVERSAL_MACHINE_IDENTITY: "UNIVERSAL_MACHINE_IDENTITY",
	KUBERNETES:                 "KUBERNETES",
}

var machineIdentityTokenInstance *util.MachineIdentityToken

func (r *InfisicalSecretReconciler) GetInfisicalConfigMap(ctx context.Context) (configMap map[string]string, errToReturn error) {
	// default key values
	defaultConfigMapData := make(map[string]string)
	defaultConfigMapData["hostAPI"] = INFISICAL_DOMAIN

	kubeConfigMap := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: OPERATOR_SETTINGS_CONFIGMAP_NAMESPACE,
		Name:      OPERATOR_SETTINGS_CONFIGMAP_NAME,
	}, kubeConfigMap)

	if err != nil {
		if errors.IsNotFound(err) {
			kubeConfigMap = nil
		} else {
			return nil, fmt.Errorf("GetConfigMapByNamespacedName: unable to fetch config map in [namespacedName=%s] [err=%s]", OPERATOR_SETTINGS_CONFIGMAP_NAMESPACE, err)
		}
	}

	if kubeConfigMap == nil {
		return defaultConfigMapData, nil
	} else {
		for key, value := range defaultConfigMapData {
			_, exists := kubeConfigMap.Data[key]
			if !exists {
				kubeConfigMap.Data[key] = value
			}
		}

		return kubeConfigMap.Data, nil
	}
}

func (r *InfisicalSecretReconciler) GetKubeSecretByNamespacedName(ctx context.Context, namespacedName types.NamespacedName) (*corev1.Secret, error) {
	kubeSecret := &corev1.Secret{}
	err := r.Client.Get(ctx, namespacedName, kubeSecret)
	if err != nil {
		kubeSecret = nil
	}

	return kubeSecret, err
}

func (r *InfisicalSecretReconciler) GetInfisicalTokenFromKubeSecret(ctx context.Context, infisicalSecret v1alpha1.InfisicalSecret) (string, error) {
	// default to new secret ref structure
	secretName := infisicalSecret.Spec.Authentication.ServiceToken.ServiceTokenSecretReference.SecretName
	secretNamespace := infisicalSecret.Spec.Authentication.ServiceToken.ServiceTokenSecretReference.SecretNamespace
	// fall back to previous secret ref
	if secretName == "" {
		secretName = infisicalSecret.Spec.TokenSecretReference.SecretName
	}

	if secretNamespace == "" {
		secretNamespace = infisicalSecret.Spec.TokenSecretReference.SecretNamespace
	}

	tokenSecret, err := r.GetKubeSecretByNamespacedName(ctx, types.NamespacedName{
		Namespace: secretNamespace,
		Name:      secretName,
	})

	if errors.IsNotFound(err) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("failed to read Infisical token secret from secret named [%s] in namespace [%s]: with error [%w]", infisicalSecret.Spec.TokenSecretReference.SecretName, infisicalSecret.Spec.TokenSecretReference.SecretNamespace, err)
	}

	infisicalServiceToken := tokenSecret.Data[INFISICAL_TOKEN_SECRET_KEY_NAME]

	return strings.Replace(string(infisicalServiceToken), " ", "", -1), nil
}

func (r *InfisicalSecretReconciler) GetInfisicalUniversalAuthFromKubeSecret(ctx context.Context, infisicalSecret v1alpha1.InfisicalSecret) (machineIdentityDetails model.MachineIdentityDetails, err error) {

	universalAuthCredsFromKubeSecret, err := r.GetKubeSecretByNamespacedName(ctx, types.NamespacedName{
		Namespace: infisicalSecret.Spec.Authentication.UniversalAuth.CredentialsRef.SecretNamespace,
		Name:      infisicalSecret.Spec.Authentication.UniversalAuth.CredentialsRef.SecretName,
	})

	if errors.IsNotFound(err) {
		return model.MachineIdentityDetails{}, nil
	}

	if err != nil {
		return model.MachineIdentityDetails{}, fmt.Errorf("something went wrong when fetching your machine identity credentials [err=%s]", err)
	}

	clientIdFromSecret := universalAuthCredsFromKubeSecret.Data[INFISICAL_MACHINE_IDENTITY_CLIENT_ID]
	clientSecretFromSecret := universalAuthCredsFromKubeSecret.Data[INFISICAL_MACHINE_IDENTITY_CLIENT_SECRET]

	return model.MachineIdentityDetails{ClientId: string(clientIdFromSecret), ClientSecret: string(clientSecretFromSecret)}, nil
}

// Fetches service account credentials from a Kubernetes secret specified in the infisicalSecret object, extracts the access key, public key, and private key from the secret, and returns them as a ServiceAccountCredentials object.
// If any keys are missing or an error occurs, returns an empty object or an error object, respectively.
func (r *InfisicalSecretReconciler) GetInfisicalServiceAccountCredentialsFromKubeSecret(ctx context.Context, infisicalSecret v1alpha1.InfisicalSecret) (serviceAccountDetails model.ServiceAccountDetails, err error) {
	serviceAccountCredsFromKubeSecret, err := r.GetKubeSecretByNamespacedName(ctx, types.NamespacedName{
		Namespace: infisicalSecret.Spec.Authentication.ServiceAccount.ServiceAccountSecretReference.SecretNamespace,
		Name:      infisicalSecret.Spec.Authentication.ServiceAccount.ServiceAccountSecretReference.SecretName,
	})

	if errors.IsNotFound(err) {
		return model.ServiceAccountDetails{}, nil
	}

	if err != nil {
		return model.ServiceAccountDetails{}, fmt.Errorf("something went wrong when fetching your service account credentials [err=%s]", err)
	}

	accessKeyFromSecret := serviceAccountCredsFromKubeSecret.Data[SERVICE_ACCOUNT_ACCESS_KEY]
	publicKeyFromSecret := serviceAccountCredsFromKubeSecret.Data[SERVICE_ACCOUNT_PUBLIC_KEY]
	privateKeyFromSecret := serviceAccountCredsFromKubeSecret.Data[SERVICE_ACCOUNT_PRIVATE_KEY]

	if accessKeyFromSecret == nil || publicKeyFromSecret == nil || privateKeyFromSecret == nil {
		return model.ServiceAccountDetails{}, nil
	}

	return model.ServiceAccountDetails{AccessKey: string(accessKeyFromSecret), PrivateKey: string(privateKeyFromSecret), PublicKey: string(publicKeyFromSecret)}, nil
}

func (r *InfisicalSecretReconciler) CreateInfisicalManagedKubeSecret(ctx context.Context, infisicalSecret v1alpha1.InfisicalSecret, secretsFromAPI []model.SingleEnvironmentVariable, ETag string) error {
	plainProcessedSecrets := make(map[string][]byte)
	secretType := infisicalSecret.Spec.ManagedSecretReference.SecretType

	for _, secret := range secretsFromAPI {
		plainProcessedSecrets[secret.Key] = []byte(secret.Value) // plain process
	}

	// copy labels and annotations from InfisicalSecret CRD
	labels := map[string]string{}
	for k, v := range infisicalSecret.Labels {
		labels[k] = v
	}

	annotations := map[string]string{}
	systemPrefixes := []string{"kubectl.kubernetes.io/", "kubernetes.io/", "k8s.io/", "helm.sh/"}
	for k, v := range infisicalSecret.Annotations {
		isSystem := false
		for _, prefix := range systemPrefixes {
			if strings.HasPrefix(k, prefix) {
				isSystem = true
				break
			}
		}
		if !isSystem {
			annotations[k] = v
		}
	}

	annotations[SECRET_VERSION_ANNOTATION] = ETag

	// create a new secret as specified by the managed secret spec of CRD
	newKubeSecretInstance := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        infisicalSecret.Spec.ManagedSecretReference.SecretName,
			Namespace:   infisicalSecret.Spec.ManagedSecretReference.SecretNamespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Type: corev1.SecretType(secretType),
		Data: plainProcessedSecrets,
	}

	if infisicalSecret.Spec.ManagedSecretReference.CreationPolicy == "Owner" {
		// Set InfisicalSecret instance as the owner and controller of the managed secret
		err := ctrl.SetControllerReference(&infisicalSecret, newKubeSecretInstance, r.Scheme)
		if err != nil {
			return err
		}
	}

	err := r.Client.Create(ctx, newKubeSecretInstance)
	if err != nil {
		return fmt.Errorf("unable to create the managed Kubernetes secret : %w", err)
	}

	fmt.Printf("Successfully created a managed Kubernetes secret with your Infisical secrets. Type: %s\n", secretType)
	return nil
}

func (r *InfisicalSecretReconciler) UpdateInfisicalManagedKubeSecret(ctx context.Context, managedKubeSecret corev1.Secret, secretsFromAPI []model.SingleEnvironmentVariable, ETag string) error {
	plainProcessedSecrets := make(map[string][]byte)
	for _, secret := range secretsFromAPI {
		plainProcessedSecrets[secret.Key] = []byte(secret.Value)
	}

	managedKubeSecret.Data = plainProcessedSecrets
	managedKubeSecret.ObjectMeta.Annotations = map[string]string{}
	managedKubeSecret.ObjectMeta.Annotations[SECRET_VERSION_ANNOTATION] = ETag

	err := r.Client.Update(ctx, &managedKubeSecret)
	if err != nil {
		return fmt.Errorf("unable to update Kubernetes secret because [%w]", err)
	}

	fmt.Println("successfully updated managed Kubernetes secret")
	return nil
}

func (r *InfisicalSecretReconciler) ReconcileInfisicalSecret(ctx context.Context, infisicalSecret v1alpha1.InfisicalSecret) error {
	infisicalToken, err := r.GetInfisicalTokenFromKubeSecret(ctx, infisicalSecret)
	if err != nil {
		return fmt.Errorf("ReconcileInfisicalSecret: unable to get service token from kube secret [err=%s]", err)
	}

	var authStrategy AuthStrategyType

	serviceAccountCreds, err := r.GetInfisicalServiceAccountCredentialsFromKubeSecret(ctx, infisicalSecret)
	if err != nil {
		return fmt.Errorf("ReconcileInfisicalSecret: unable to get service account creds from kube secret [err=%s]", err)
	}

	infisicalMachineIdentityCreds, err := r.GetInfisicalUniversalAuthFromKubeSecret(ctx, infisicalSecret)
	if err != nil {
		return fmt.Errorf("ReconcileInfisicalSecret: unable to get machine identity creds from kube secret [err=%s]", err)
	}

	if serviceAccountCreds.AccessKey != "" || serviceAccountCreds.PrivateKey != "" || serviceAccountCreds.PublicKey != "" {
		authStrategy = AuthStrategy.SERVICE_ACCOUNT
	} else if infisicalToken != "" {
		authStrategy = AuthStrategy.SERVICE_TOKEN
	} else if infisicalMachineIdentityCreds.ClientId != "" && infisicalMachineIdentityCreds.ClientSecret != "" {
		authStrategy = AuthStrategy.UNIVERSAL_MACHINE_IDENTITY
	} else if infisicalSecret.Spec.Authentication.Kubernetes.IdentityId != "" {
		authStrategy = AuthStrategy.KUBERNETES
	} else {
		return fmt.Errorf("no authentication method provided. You must provide either a valid service token or a service account details to fetch secrets\n")
	}

	r.SetInfisicalTokenLoadCondition(ctx, &infisicalSecret, err)
	if err != nil {
		return fmt.Errorf("unable to load Infisical Token from the specified Kubernetes secret with error [%w]", err)
	}

	// Look for managed secret by name and namespace
	managedKubeSecret, err := r.GetKubeSecretByNamespacedName(ctx, types.NamespacedName{
		Name:      infisicalSecret.Spec.ManagedSecretReference.SecretName,
		Namespace: infisicalSecret.Spec.ManagedSecretReference.SecretNamespace,
	})

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("something went wrong when fetching the managed Kubernetes secret [%w]", err)
	}

	// Get exiting Etag if exists
	secretVersionBasedOnETag := ""
	if managedKubeSecret != nil {
		secretVersionBasedOnETag = managedKubeSecret.Annotations[SECRET_VERSION_ANNOTATION]
	}

	if authStrategy == AuthStrategy.UNIVERSAL_MACHINE_IDENTITY && machineIdentityTokenInstance == nil {
		// Create new machine identity token instance
		machineIdentityTokenInstance = util.NewMachineIdentityToken(infisicalMachineIdentityCreds.ClientId, infisicalMachineIdentityCreds.ClientSecret)
	} else if authStrategy == AuthStrategy.KUBERNETES && machineIdentityTokenInstance == nil {
		// Create new machine identity token instance
		machineIdentityTokenInstance, err = util.NewMachineIdentityKubernetesToken(infisicalSecret.Spec.Authentication.Kubernetes.IdentityId, infisicalSecret.Spec.Authentication.Kubernetes.ServiceAccountTokenPath)

		if err != nil {
			return fmt.Errorf("ReconcileInfisicalSecret: unable to create machine identity token instance from kubernetes auth [err=%s]", err)
		}
	}

	var plainTextSecretsFromApi []model.SingleEnvironmentVariable
	var updateDetails model.RequestUpdateUpdateDetails

	if authStrategy == AuthStrategy.SERVICE_ACCOUNT { // Service Account
		plainTextSecretsFromApi, updateDetails, err = util.GetPlainTextSecretsViaServiceAccount(serviceAccountCreds, infisicalSecret.Spec.Authentication.ServiceAccount.ProjectId, infisicalSecret.Spec.Authentication.ServiceAccount.EnvironmentName, secretVersionBasedOnETag)
		if err != nil {
			return fmt.Errorf("\nfailed to get secrets because [err=%v]", err)
		}

		fmt.Println("ReconcileInfisicalSecret: Fetched secrets via service account")

	} else if authStrategy == AuthStrategy.SERVICE_TOKEN { // Service Tokens (deprecated)
		envSlug := infisicalSecret.Spec.Authentication.ServiceToken.SecretsScope.EnvSlug
		secretsPath := infisicalSecret.Spec.Authentication.ServiceToken.SecretsScope.SecretsPath
		recursive := infisicalSecret.Spec.Authentication.ServiceToken.SecretsScope.Recursive

		plainTextSecretsFromApi, updateDetails, err = util.GetPlainTextSecretsViaServiceToken(infisicalToken, secretVersionBasedOnETag, envSlug, secretsPath, recursive)
		if err != nil {
			return fmt.Errorf("\nfailed to get secrets because [err=%v]", err)
		}

		fmt.Println("ReconcileInfisicalSecret: Fetched secrets via service token")
	} else if authStrategy == AuthStrategy.UNIVERSAL_MACHINE_IDENTITY || authStrategy == AuthStrategy.KUBERNETES { // Machine Identity

		accessToken, err := machineIdentityTokenInstance.GetToken()
		if err != nil {
			return fmt.Errorf("%s", "Waiting for access token to become available")
		}

		var scope v1alpha1.MachineIdentityScopeInWorkspace
		var formattedStrategy string
		if authStrategy == AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
			scope = infisicalSecret.Spec.Authentication.UniversalAuth.SecretsScope
			formattedStrategy = "universal auth"
		} else {
			scope = infisicalSecret.Spec.Authentication.Kubernetes.SecretsScope
			formattedStrategy = "kubernetes auth"
		}

		plainTextSecretsFromApi, updateDetails, err = util.GetPlainTextSecretsViaUniversalAuth(accessToken, secretVersionBasedOnETag, scope)

		if err != nil {
			return fmt.Errorf("\nfailed to get secrets because [err=%v]", err)
		}
		fmt.Printf("ReconcileInfisicalSecret: Fetched secrets via %s\n", formattedStrategy)

	} else {
		return fmt.Errorf("no authentication method provided. You must provide either a valid service token or a service account details to fetch secrets")
	}

	if !updateDetails.Modified {
		fmt.Println("No secrets modified so reconcile not needed")
		return nil
	}

	if managedKubeSecret == nil {
		return r.CreateInfisicalManagedKubeSecret(ctx, infisicalSecret, plainTextSecretsFromApi, updateDetails.ETag)
	} else {
		return r.UpdateInfisicalManagedKubeSecret(ctx, *managedKubeSecret, plainTextSecretsFromApi, updateDetails.ETag)
	}

}
