package v1alpha1

// ConfigurationT TODO
type ConfigurationT struct {
	Integrations IntegrationsT `yaml:"integrations"`
}

// IntegrationsT TODO
type IntegrationsT struct {
	Alertmanager AlertmanagerT `yaml:"alertmanager,omitempty"`
	Webhook      WebhookT      `yaml:"webhook,omitempty"`
}

// AlertmanagerT TODO
type AlertmanagerT struct {
	Url     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

// WebhookT TODO
type WebhookT struct {
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers,omitempty"`
}
