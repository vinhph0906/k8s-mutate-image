package main

import (
	"flag"
	"net/http"

	corev1 "k8s.io/api/core/v1"

	"github.com/sirupsen/logrus"
	"github.com/sqooba/k8s-mutate-image-and-policy/configs"
)

type mutationWH struct {
	registries             map[string]string
	imagePullSecret        string
	appendImagePullSecret  bool
	forceImagePullPolicy   bool
	imagePullPolicyToForce corev1.PullPolicy
	defaultStorageClass    string
	excludedNamespaces     map[string]bool
	logger                 *logrus.Logger
}

func main() {

	// Command line flags
	var (
		configFile = flag.String("config", "config.yaml", "Path to YAML configuration file (default: config.yaml)")
	)
	flag.Parse()

	// Load configuration
	cfg, err := configs.NewConfig(*configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	logger := SetupLogger(&cfg.Log)
	logger.WithFields(logrus.Fields{
		"version": Version,
		"commit":  GitCommit,
		"build":   BuildDate,
		"osarch":  OsArch,
	}).Info("k8s-mutate-image-and-policy-webhook is starting...")
	// Validate pull policy
	excludedNamespaces := make(map[string]bool)
	for _, ns := range cfg.ExcludeNamespaces {
		excludedNamespaces[ns] = true
	}
	wh := mutationWH{
		registries:             cfg.Registries,
		imagePullSecret:        cfg.ImagePullSecret,
		appendImagePullSecret:  cfg.AppendImagePullSecret,
		forceImagePullPolicy:   cfg.ForceImagePullPolicy,
		imagePullPolicyToForce: cfg.ImagePullPolicyToForce,
		defaultStorageClass:    cfg.DefaultStorageClass,
		excludedNamespaces:     excludedNamespaces,
		logger:                 logger,
	}

	mux := http.NewServeMux()

	wh.routes(mux)

	server := &http.Server{
		// We listen on port 8443 such that we do not need root privileges or extra capabilities for this server.
		// The Service object will take care of mapping this port to the HTTPS port 443.
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	wh.logger.Fatal(server.ListenAndServeTLS(cfg.TLS.CertFile, cfg.TLS.KeyFile))
}
