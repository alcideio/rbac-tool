package whoami

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/alcideio/rbac-tool/pkg/kube"
	"github.com/kylelemons/godebug/pretty"
	v1 "k8s.io/api/authentication/v1"
	authz "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/transport"
	"k8s.io/klog"
)

// tokenExtractor helps to retrieve token
type tokenExtractor struct {
	rountTripper http.RoundTripper
	token        string
}

// RoundTrip gets token
func (t *tokenExtractor) RoundTrip(req *http.Request) (*http.Response, error) {
	header := req.Header.Get("authorization")

	if strings.HasPrefix(header, "Bearer ") {
		t.token = strings.ReplaceAll(header, "Bearer ", "")
		klog.V(5).Infof("extracted token successfully")
	} else {
		klog.V(5).Infof("could not extract token '%v'", req.Header)
	}

	return t.rountTripper.RoundTrip(req)
}

func extractToken(client *kube.KubeClient) (string, error) {
	config := client.Config
	tokenExtractor := &tokenExtractor{}
	config.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		tokenExtractor.rountTripper = rt
		return tokenExtractor
	})

	k8sClient, err := clientset.NewForConfig(config)
	if err != nil {
		return "", err
	}

	_, err = k8sClient.AuthorizationV1().SelfSubjectAccessReviews().Create(
		context.Background(),
		&authz.SelfSubjectAccessReview{
			Spec: authz.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authz.ResourceAttributes{
					Namespace: "",
				},
			},
		},
		metav1.CreateOptions{},
	)

	return tokenExtractor.token, nil
}

func ExtractUserInfo(client *kube.KubeClient) (*v1.UserInfo, error) {
	var token string
	userInfo := &v1.UserInfo{}

	config := client.Config

	c, err := config.TransportConfig()
	if err != nil {
		return nil, err
	}

	klog.V(9).Infof("Config:\n%v\n", pretty.Sprint(config))

	// Token based authentication has preference over basic auth and certificate auth
	if c.HasTokenAuth() {
		if config.BearerTokenFile != "" {
			klog.V(5).Infof("extracting token from file '%v'", config.BearerTokenFile)
			d, err := os.ReadFile(config.BearerTokenFile)
			if err != nil {
				return nil, err
			}

			token = string(d)
		}

		if config.BearerToken != "" {
			klog.V(5).Infof("extracting token bearer")
			token = config.BearerToken
		}
	}

	if token == "" {
		klog.V(5).Infof("extracting token actively")
		t, err := extractToken(client)
		if err != nil {
			return nil, err
		}

		token = t
	}

	user, err := client.TokenReview(token)

	if err != nil {
		klog.V(5).Infof("TokenReview failed to '%v'", err)

		if c.HasCertAuth() {
			cert, err := getClientCertificate(c)
			if err != nil {
				return nil, err
			}
			klog.V(5).Infof("extracting from cert '%v'", pretty.Sprint(cert.Subject))
			userInfo.Username = cert.Subject.CommonName
			userInfo.Groups = cert.Subject.Organization
			return userInfo, nil
		}

		if c.HasBasicAuth() {
			klog.V(5).Infof("extracting from basic auth")
			userInfo.Username = config.Username
			return userInfo, nil
		}

		return nil, fmt.Errorf("Failed to extract user information")
	}

	klog.V(5).Infof("TokenReview extraction completed ok '%v'", user)
	return &user, nil
}

func getClientCertificate(c *transport.Config) (*x509.Certificate, error) {
	tlsConfig, err := transport.TLSConfigFor(c)
	if err != nil {
		return nil, err
	}
	// GetClientCertificate has been set in transport.TLSConfigFor,
	// so it is not nil
	cert, err := tlsConfig.GetClientCertificate(nil)
	if err != nil {
		return nil, err
	}
	if cert.Leaf != nil {
		return cert.Leaf, nil
	}
	return x509.ParseCertificate(cert.Certificate[0])
}
