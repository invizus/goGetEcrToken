package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// This app can get AWS details from env vars by default.
// Define these env vars to authenticate into your AWS:
// AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION
// you must also define the following env vars to generate the secretmap
// VAR_NAMESPACE, VAR_SECRETNAME

func main() {

	kubeNamespace, ok := os.LookupEnv("VAR_NAMESPACE") // i.e. "shared-secrets"
	if !ok {
		log.Fatal("environment variable VAR_NAMESPACE is not set")
	}

	kubeSecretName, ok := os.LookupEnv("VAR_SECRETNAME") // i.e. "ecr-login"
	if !ok {
		log.Fatal("environment variable VAR_SECRETNAME is not set")
	}

	fmt.Println("Storing secret in namespace:", kubeNamespace)
	fmt.Println("Secret name will be:", kubeSecretName)

	cfg, err := awsConfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error, " + err.Error())
	}
	fmt.Println(cfg)

	// Create ECR client
	ecrClient := ecr.NewFromConfig(cfg)
	resp, err := ecrClient.GetAuthorizationToken(context.TODO(), &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		log.Fatalf("failed to fetch ECR token: %v", err)
	}

	authData := resp.AuthorizationData[0]
	token := *authData.AuthorizationToken
	decodedToken, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		log.Fatalf("failed to decode ECR token: %v", err)
	}

	// Extract username and password
	creds := strings.SplitN(string(decodedToken), ":", 2)
	if len(creds) != 2 {
		log.Fatal("malformed authorization token")
	}

	username, password := creds[0], creds[1]
	server := *authData.ProxyEndpoint

	// test
	//	fmt.Println("Server:", server)
	//	fmt.Println("Username:", username)
	//	fmt.Println("Password:", password)

	// Prepare Docker registry secret
	dockerConfig := map[string]map[string]map[string]string{
		"auths": {
			server: {
				"username": username,
				"password": password,
				"auth":     base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password))),
			},
		},
	}
	dockerConfigJSON, err := json.Marshal(dockerConfig)
	if err != nil {
		log.Fatalf("failed to marshal docker config: %v", err)
	}
	// Create in-cluster Kubernetes client
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("failed to create in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		log.Fatalf("failed to create Kubernetes client: %v", err)
	}

	// Define secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubeSecretName,
			Namespace: kubeNamespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJSON,
		},
	}

	// Create the secret
	_, err = clientset.CoreV1().Secrets(kubeNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		log.Fatalf("failed to create Docker registry secret: %v", err)
	}

	fmt.Println("Docker-registry secret created successfully!")

}
