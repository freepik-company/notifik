package webhook

// TODO
type AlertmanagerAlertList []AlertmanagerAlert

// Alert represents the structure of an alert in Alertmanager
// Ref: https://prometheus.io/docs/alerting/latest/clients/
// Ref: https://raw.githubusercontent.com/prometheus/alertmanager/main/api/v2/openapi.yaml
type AlertmanagerAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`

	// Optional params
	StartsAt     string `json:"startsAt,omitempty"`
	EndsAt       string `json:"endsAt,omitempty"`
	GeneratorUrl string `json:"generatorURL,omitempty"`
}
