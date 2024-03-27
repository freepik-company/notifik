package alertmanager

// TODO: Review if this is needed
type AlertList []Alert

// Alert represents the structure of an alert in Alertmanager
// Ref: https://prometheus.io/docs/alerting/latest/clients/
// Ref: https://raw.githubusercontent.com/prometheus/alertmanager/main/api/v2/openapi.yaml
type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`

	// Optional params
	StartAt      string `json:"startAt,omitempty"`
	EndsAt       string `json:"endsAt,omitempty"`
	GeneratorUrl string `json:"generatorURL,omitempty"`
}
