package configs

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
	corev1 "k8s.io/api/core/v1"

	"gopkg.in/yaml.v3"
)

type TLSConfig struct {
	CertFile string `yaml:"cert_file" default:"/run/secrets/tls/webhook-server-tls.crt" validate:"required"`
	KeyFile  string `yaml:"key_file" default:"/run/secrets/tls/webhook-server-tls.key" validate:"required"`
}

type Log struct {
	Level      string `yaml:"level" default:"info" validate:"required,oneof=debug info warn error fatal"` // Required with validation
	Format     string `yaml:"format" default:"json" validate:"required,oneof=json text"`                  // Optional: json (default), text
	Output     string `yaml:"output" default:"stdout" validate:"required,oneof=stdout file both"`         // Optional: stdout (default), file, or both
	FilePath   string `yaml:"file_path" default:"log.json" validate:"required_if=Output file"`            // File path when output includes file
	MaxSize    int    `yaml:"max_size" default:"100" validate:"required_if=Output file,numeric,min=1"`    // Maximum file size in MB (default: 100MB)
	MaxBackups int    `yaml:"max_backups" default:"5" validate:"required_if=Output file,numeric,min=1"`   // Maximum number of backup files (default: 5)
	MaxAge     int    `yaml:"max_age" default:"30" validate:"required_if=Output file,numeric,min=1"`      // Maximum age of log files in days (default: 30)
	Compress   bool   `yaml:"compress" default:"true" `                                                   // Whether to compress old log files (default: true)
}
type Config struct {
	TLS                    TLSConfig         `yaml:"tls"`
	Port                   string            `yaml:"port" default:"8443" validate:"required,numeric,min=1"`
	Host                   string            `yaml:"host" default:"0.0.0.0" validate:"required,ip"`
	Log                    Log               `yaml:"log"`
	Registries             map[string]string `yaml:"registry" validate:"min=1,required"`
	ImagePullSecret        string            `yaml:"image_pull_secret"`
	AppendImagePullSecret  bool              `yaml:"image_pull_secret_append" default:"false"`
	ForceImagePullPolicy   bool              `yaml:"force_image_pull_policy"`
	ImagePullPolicyToForce corev1.PullPolicy `yaml:"image_pull_policy_to_force" default:"IfNotPresent" validate:"oneof=Always IfNotPresent Never"`
	DefaultStorageClass    string            `yaml:"default_storage_class"`
	// ExcludeNamespaces      []string          `yaml:"exclude_namespaces"`
	IncludeNamespaces []string `yaml:"include_namespaces"`
}

func NewConfig(filename string) (*Config, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var config Config
	// Step 1: apply defaults
	if err := defaults.Set(&config); err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, err
	}
	err = config.validate()
	if err != nil {
		return nil, err
	}
	return &config, nil
}
func (c *Config) validate() error {
	validate := validator.New()
	return validate.Struct(c)
}
