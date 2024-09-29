package v1alpha1

// ConfigurationT TODO
type ConfigurationT struct {
	Integrations []IntegrationT `yaml:"integrations"`
}

// IntegrationT TODO
type IntegrationT struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Webhook WebhookT `yaml:"webhook,omitempty"`
}

// WebhookT TODO
type WebhookT struct {
	Url       string            `yaml:"url"`
	Verb      string            `yaml:"verb"`
	Headers   map[string]string `yaml:"headers,omitempty"`
	Validator string            `yaml:"validator,omitempty"`
}
