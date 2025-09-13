package crm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CRMProvider defines the interface for CRM integrations
type CRMProvider interface {
	GetName() string
	CreateLead(ctx context.Context, lead *Lead, config *Config) (*Response, error)
	UpdateLead(ctx context.Context, leadID string, lead *Lead, config *Config) (*Response, error)
	GetLead(ctx context.Context, leadID string, config *Config) (*Lead, error)
	SearchLeads(ctx context.Context, query *SearchQuery, config *Config) (*SearchResult, error)
	TestConnection(ctx context.Context, config *Config) error
}

// Config contains CRM integration configuration
type Config struct {
	Provider       string            `json:"provider"`
	APIKey         string            `json:"api_key"`
	APISecret      string            `json:"api_secret,omitempty"`
	BaseURL        string            `json:"base_url,omitempty"`
	WebhookURL     string            `json:"webhook_url,omitempty"`
	FieldMapping   map[string]string `json:"field_mapping"`
	Options        map[string]interface{} `json:"options,omitempty"`
	RateLimit      *RateLimitConfig  `json:"rate_limit,omitempty"`
	Timeout        time.Duration     `json:"timeout,omitempty"`
}

// RateLimitConfig defines rate limiting parameters
type RateLimitConfig struct {
	RequestsPerMinute int           `json:"requests_per_minute"`
	BurstSize         int           `json:"burst_size"`
	RetryAfter        time.Duration `json:"retry_after"`
}

// Lead represents a standardized lead structure
type Lead struct {
	ID                  string                 `json:"id,omitempty"`
	ExternalID          string                 `json:"external_id,omitempty"`
	TenantID           string                 `json:"tenant_id"`
	Source             string                 `json:"source"`

	// Contact Information
	FirstName          string                 `json:"first_name"`
	LastName           string                 `json:"last_name"`
	FullName           string                 `json:"full_name,omitempty"`
	Email              string                 `json:"email,omitempty"`
	Phone              string                 `json:"phone"`
	MobilePhone        string                 `json:"mobile_phone,omitempty"`

	// Address Information
	Address            string                 `json:"address,omitempty"`
	City               string                 `json:"city,omitempty"`
	State              string                 `json:"state,omitempty"`
	ZipCode            string                 `json:"zip_code,omitempty"`
	Country            string                 `json:"country,omitempty"`

	// Project Information
	ProjectType        string                 `json:"project_type"`
	ProjectDescription string                 `json:"project_description,omitempty"`
	Timeline           string                 `json:"timeline,omitempty"`
	BudgetRange        string                 `json:"budget_range,omitempty"`

	// Lead Qualification
	LeadScore          int                    `json:"lead_score"`
	LeadStatus         string                 `json:"lead_status"`
	LeadStage          string                 `json:"lead_stage,omitempty"`
	Priority           string                 `json:"priority,omitempty"`

	// Conversation Data
	CallID             string                 `json:"call_id,omitempty"`
	Transcript         string                 `json:"transcript,omitempty"`
	Sentiment          string                 `json:"sentiment,omitempty"`
	Intent             string                 `json:"intent,omitempty"`

	// Metadata
	Notes              string                 `json:"notes,omitempty"`
	Tags               []string               `json:"tags,omitempty"`
	CustomFields       map[string]interface{} `json:"custom_fields,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at,omitempty"`

	// Assignment
	AssignedTo         string                 `json:"assigned_to,omitempty"`
	OwnerID            string                 `json:"owner_id,omitempty"`
	TeamID             string                 `json:"team_id,omitempty"`
}

// Response represents a standardized CRM response
type Response struct {
	Success     bool                   `json:"success"`
	LeadID      string                 `json:"lead_id,omitempty"`
	ExternalID  string                 `json:"external_id,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StatusCode  int                    `json:"status_code,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SearchQuery defines parameters for lead search
type SearchQuery struct {
	Email       string            `json:"email,omitempty"`
	Phone       string            `json:"phone,omitempty"`
	Name        string            `json:"name,omitempty"`
	Status      string            `json:"status,omitempty"`
	Source      string            `json:"source,omitempty"`
	DateFrom    *time.Time        `json:"date_from,omitempty"`
	DateTo      *time.Time        `json:"date_to,omitempty"`
	Limit       int               `json:"limit,omitempty"`
	Offset      int               `json:"offset,omitempty"`
	CustomFilters map[string]interface{} `json:"custom_filters,omitempty"`
}

// SearchResult contains search results
type SearchResult struct {
	Leads      []*Lead `json:"leads"`
	Total      int     `json:"total"`
	Limit      int     `json:"limit"`
	Offset     int     `json:"offset"`
	HasMore    bool    `json:"has_more"`
}

// Manager manages CRM integrations
type Manager struct {
	providers map[string]CRMProvider
	client    *http.Client
}

// NewManager creates a new CRM manager
func NewManager() *Manager {
	return &Manager{
		providers: make(map[string]CRMProvider),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterProvider registers a CRM provider
func (m *Manager) RegisterProvider(provider CRMProvider) {
	m.providers[provider.GetName()] = provider
}

// GetProvider retrieves a CRM provider by name
func (m *Manager) GetProvider(name string) (CRMProvider, error) {
	provider, exists := m.providers[name]
	if !exists {
		return nil, fmt.Errorf("CRM provider '%s' not found", name)
	}
	return provider, nil
}

// ListProviders returns all registered provider names
func (m *Manager) ListProviders() []string {
	var names []string
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// HubSpotProvider implements HubSpot CRM integration
type HubSpotProvider struct {
	client *http.Client
}

// NewHubSpotProvider creates a new HubSpot provider
func NewHubSpotProvider() *HubSpotProvider {
	return &HubSpotProvider{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (h *HubSpotProvider) GetName() string {
	return "hubspot"
}

func (h *HubSpotProvider) CreateLead(ctx context.Context, lead *Lead, config *Config) (*Response, error) {
	endpoint := "https://api.hubapi.com/crm/v3/objects/contacts"

	// Map lead data to HubSpot format
	properties := h.mapLeadToHubSpot(lead, config.FieldMapping)

	payload := map[string]interface{}{
		"properties": properties,
	}

	resp, err := h.makeRequest(ctx, "POST", endpoint, payload, config)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}, err
	}

	var hubspotResp map[string]interface{}
	if err := json.Unmarshal(resp, &hubspotResp); err != nil {
		return &Response{Success: false, Error: "Failed to parse response"}, err
	}

	leadID := ""
	if id, ok := hubspotResp["id"].(string); ok {
		leadID = id
	}

	return &Response{
		Success:    true,
		LeadID:     leadID,
		ExternalID: leadID,
		Message:    "Lead created successfully",
		Data:       hubspotResp,
	}, nil
}

func (h *HubSpotProvider) UpdateLead(ctx context.Context, leadID string, lead *Lead, config *Config) (*Response, error) {
	endpoint := fmt.Sprintf("https://api.hubapi.com/crm/v3/objects/contacts/%s", leadID)

	properties := h.mapLeadToHubSpot(lead, config.FieldMapping)
	payload := map[string]interface{}{
		"properties": properties,
	}

	resp, err := h.makeRequest(ctx, "PATCH", endpoint, payload, config)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}, err
	}

	var hubspotResp map[string]interface{}
	if err := json.Unmarshal(resp, &hubspotResp); err != nil {
		return &Response{Success: false, Error: "Failed to parse response"}, err
	}

	return &Response{
		Success:    true,
		LeadID:     leadID,
		ExternalID: leadID,
		Message:    "Lead updated successfully",
		Data:       hubspotResp,
	}, nil
}

func (h *HubSpotProvider) GetLead(ctx context.Context, leadID string, config *Config) (*Lead, error) {
	endpoint := fmt.Sprintf("https://api.hubapi.com/crm/v3/objects/contacts/%s", leadID)

	resp, err := h.makeRequest(ctx, "GET", endpoint, nil, config)
	if err != nil {
		return nil, err
	}

	var hubspotResp map[string]interface{}
	if err := json.Unmarshal(resp, &hubspotResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	lead := h.mapHubSpotToLead(hubspotResp, config.FieldMapping)
	return lead, nil
}

func (h *HubSpotProvider) SearchLeads(ctx context.Context, query *SearchQuery, config *Config) (*SearchResult, error) {
	endpoint := "https://api.hubapi.com/crm/v3/objects/contacts/search"

	// Build HubSpot search payload
	filters := []map[string]interface{}{}

	if query.Email != "" {
		filters = append(filters, map[string]interface{}{
			"propertyName": "email",
			"operator":     "EQ",
			"value":        query.Email,
		})
	}

	if query.Phone != "" {
		filters = append(filters, map[string]interface{}{
			"propertyName": "phone",
			"operator":     "EQ",
			"value":        query.Phone,
		})
	}

	payload := map[string]interface{}{
		"filterGroups": []map[string]interface{}{
			{
				"filters": filters,
			},
		},
		"limit": 10,
	}

	if query.Limit > 0 {
		payload["limit"] = query.Limit
	}

	resp, err := h.makeRequest(ctx, "POST", endpoint, payload, config)
	if err != nil {
		return nil, err
	}

	var searchResp map[string]interface{}
	if err := json.Unmarshal(resp, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	result := &SearchResult{
		Leads: []*Lead{},
	}

	if results, ok := searchResp["results"].([]interface{}); ok {
		for _, r := range results {
			if resultMap, ok := r.(map[string]interface{}); ok {
				lead := h.mapHubSpotToLead(resultMap, config.FieldMapping)
				result.Leads = append(result.Leads, lead)
			}
		}
		result.Total = len(result.Leads)
	}

	return result, nil
}

func (h *HubSpotProvider) TestConnection(ctx context.Context, config *Config) error {
	endpoint := "https://api.hubapi.com/crm/v3/objects/contacts"
	params := url.Values{}
	params.Add("limit", "1")

	fullURL := endpoint + "?" + params.Encode()
	_, err := h.makeRequest(ctx, "GET", fullURL, nil, config)
	return err
}

func (h *HubSpotProvider) makeRequest(ctx context.Context, method, endpoint string, payload interface{}, config *Config) ([]byte, error) {
	var body io.Reader

	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (h *HubSpotProvider) mapLeadToHubSpot(lead *Lead, fieldMapping map[string]string) map[string]interface{} {
	properties := make(map[string]interface{})

	// Default mappings
	defaultMappings := map[string]string{
		"first_name": "firstname",
		"last_name":  "lastname",
		"email":      "email",
		"phone":      "phone",
		"city":       "city",
		"state":      "state",
		"zip_code":   "zip",
		"notes":      "notes",
	}

	// Apply field mappings
	mappings := make(map[string]string)
	for k, v := range defaultMappings {
		mappings[k] = v
	}
	for k, v := range fieldMapping {
		mappings[k] = v
	}

	// Map lead fields
	if lead.FirstName != "" && mappings["first_name"] != "" {
		properties[mappings["first_name"]] = lead.FirstName
	}
	if lead.LastName != "" && mappings["last_name"] != "" {
		properties[mappings["last_name"]] = lead.LastName
	}
	if lead.Email != "" && mappings["email"] != "" {
		properties[mappings["email"]] = lead.Email
	}
	if lead.Phone != "" && mappings["phone"] != "" {
		properties[mappings["phone"]] = lead.Phone
	}
	if lead.City != "" && mappings["city"] != "" {
		properties[mappings["city"]] = lead.City
	}
	if lead.State != "" && mappings["state"] != "" {
		properties[mappings["state"]] = lead.State
	}
	if lead.ZipCode != "" && mappings["zip_code"] != "" {
		properties[mappings["zip_code"]] = lead.ZipCode
	}

	// Add lead score if mapping exists
	if mappings["lead_score"] != "" {
		properties[mappings["lead_score"]] = lead.LeadScore
	}

	// Add project type as a note if no specific field
	notes := lead.Notes
	if lead.ProjectType != "" {
		if notes != "" {
			notes += "\n"
		}
		notes += fmt.Sprintf("Project Type: %s", lead.ProjectType)
	}
	if lead.Timeline != "" {
		if notes != "" {
			notes += "\n"
		}
		notes += fmt.Sprintf("Timeline: %s", lead.Timeline)
	}
	if notes != "" && mappings["notes"] != "" {
		properties[mappings["notes"]] = notes
	}

	// Add custom fields
	for key, value := range lead.CustomFields {
		if mappedKey, exists := mappings[key]; exists {
			properties[mappedKey] = value
		}
	}

	return properties
}

func (h *HubSpotProvider) mapHubSpotToLead(hubspotData map[string]interface{}, fieldMapping map[string]string) *Lead {
	lead := &Lead{
		CustomFields: make(map[string]interface{}),
	}

	if id, ok := hubspotData["id"].(string); ok {
		lead.ExternalID = id
	}

	if properties, ok := hubspotData["properties"].(map[string]interface{}); ok {
		// Reverse field mapping
		reverseMapping := make(map[string]string)
		for k, v := range fieldMapping {
			reverseMapping[v] = k
		}

		// Default reverse mappings
		defaultReverse := map[string]string{
			"firstname": "first_name",
			"lastname":  "last_name",
			"email":     "email",
			"phone":     "phone",
			"city":      "city",
			"state":     "state",
			"zip":       "zip_code",
			"notes":     "notes",
		}

		for k, v := range defaultReverse {
			if _, exists := reverseMapping[k]; !exists {
				reverseMapping[k] = v
			}
		}

		// Map properties back to lead
		for hubspotField, value := range properties {
			if leadField, exists := reverseMapping[hubspotField]; exists {
				switch leadField {
				case "first_name":
					if str, ok := value.(string); ok {
						lead.FirstName = str
					}
				case "last_name":
					if str, ok := value.(string); ok {
						lead.LastName = str
					}
				case "email":
					if str, ok := value.(string); ok {
						lead.Email = str
					}
				case "phone":
					if str, ok := value.(string); ok {
						lead.Phone = str
					}
				case "city":
					if str, ok := value.(string); ok {
						lead.City = str
					}
				case "state":
					if str, ok := value.(string); ok {
						lead.State = str
					}
				case "zip_code":
					if str, ok := value.(string); ok {
						lead.ZipCode = str
					}
				case "notes":
					if str, ok := value.(string); ok {
						lead.Notes = str
					}
				default:
					lead.CustomFields[leadField] = value
				}
			}
		}
	}

	// Set full name if not already set
	if lead.FullName == "" && (lead.FirstName != "" || lead.LastName != "") {
		lead.FullName = strings.TrimSpace(lead.FirstName + " " + lead.LastName)
	}

	return lead
}

// SalesforceProvider implements Salesforce CRM integration
type SalesforceProvider struct {
	client *http.Client
}

func NewSalesforceProvider() *SalesforceProvider {
	return &SalesforceProvider{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *SalesforceProvider) GetName() string {
	return "salesforce"
}

func (s *SalesforceProvider) CreateLead(ctx context.Context, lead *Lead, config *Config) (*Response, error) {
	// Implement Salesforce lead creation
	return &Response{Success: true, Message: "Salesforce integration not fully implemented"}, nil
}

func (s *SalesforceProvider) UpdateLead(ctx context.Context, leadID string, lead *Lead, config *Config) (*Response, error) {
	return &Response{Success: true, Message: "Salesforce integration not fully implemented"}, nil
}

func (s *SalesforceProvider) GetLead(ctx context.Context, leadID string, config *Config) (*Lead, error) {
	return &Lead{ExternalID: leadID}, nil
}

func (s *SalesforceProvider) SearchLeads(ctx context.Context, query *SearchQuery, config *Config) (*SearchResult, error) {
	return &SearchResult{Leads: []*Lead{}}, nil
}

func (s *SalesforceProvider) TestConnection(ctx context.Context, config *Config) error {
	return nil
}

// PipedriveProvider implements Pipedrive CRM integration
type PipedriveProvider struct {
	client *http.Client
}

func NewPipedriveProvider() *PipedriveProvider {
	return &PipedriveProvider{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *PipedriveProvider) GetName() string {
	return "pipedrive"
}

func (p *PipedriveProvider) CreateLead(ctx context.Context, lead *Lead, config *Config) (*Response, error) {
	return &Response{Success: true, Message: "Pipedrive integration not fully implemented"}, nil
}

func (p *PipedriveProvider) UpdateLead(ctx context.Context, leadID string, lead *Lead, config *Config) (*Response, error) {
	return &Response{Success: true, Message: "Pipedrive integration not fully implemented"}, nil
}

func (p *PipedriveProvider) GetLead(ctx context.Context, leadID string, config *Config) (*Lead, error) {
	return &Lead{ExternalID: leadID}, nil
}

func (p *PipedriveProvider) SearchLeads(ctx context.Context, query *SearchQuery, config *Config) (*SearchResult, error) {
	return &SearchResult{Leads: []*Lead{}}, nil
}

func (p *PipedriveProvider) TestConnection(ctx context.Context, config *Config) error {
	return nil
}

// Helper function to initialize default providers
func SetupDefaultProviders() *Manager {
	manager := NewManager()
	manager.RegisterProvider(NewHubSpotProvider())
	manager.RegisterProvider(NewSalesforceProvider())
	manager.RegisterProvider(NewPipedriveProvider())
	return manager
}