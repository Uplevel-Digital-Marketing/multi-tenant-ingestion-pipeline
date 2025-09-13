# 🤖 Gemini Agent & MCP CRM Integration - Detailed Plan

## ❗ **You're Right - Key Components Need Clarification!**

Looking at the Complete Implementation Guide, there are two critical components that need detailed explanation:

1. **"Go/Gemini agent"** (mentioned in line 5 of the mission statement)
2. **"MCP framework for CRM integration"** (mentioned in line 304)

Let me break down exactly how these work and integrate into our system.

---

## 🧠 **The Go/Gemini Agent Architecture**

### **What is the Go/Gemini Agent?**

The **Go/Gemini agent** is the core intelligent processing service that runs on Cloud Run. It's not separate from our pipeline - it IS the pipeline's brain.

```go
// cmd/gemini-agent/main.go
type GeminiAgent struct {
    VertexAIClient   *aiplatform.PredictionClient
    WorkflowEngine   *workflow.Engine
    CRMConnectors    map[string]crm.Connector
    ConfigManager    *config.Manager
}

func (g *GeminiAgent) ProcessRequest(ctx context.Context, req *models.IncomingRequest) error {
    // 1. Load tenant configuration
    config := g.ConfigManager.GetTenantConfig(req.TenantID)

    // 2. Intelligent content analysis
    analysis := g.analyzeWithGemini(req.Content, config.AISettings)

    // 3. Workflow orchestration
    workflow := g.WorkflowEngine.CreateWorkflow(config.WorkflowConfig)

    // 4. Execute workflow steps
    return workflow.Execute(ctx, req, analysis)
}
```

### **How Gemini Integration Works**

The agent uses **Vertex AI Gemini 2.5 Flash** for multiple intelligence tasks:

```go
// internal/ai/gemini_agent.go
type GeminiIntelligenceService struct {
    client *aiplatform.PredictionClient
    model  string // "gemini-2.5-flash"
}

// Multi-step Gemini analysis
func (g *GeminiIntelligenceService) AnalyzeContent(content string, metadata map[string]interface{}) (*Analysis, error) {
    // Step 1: Content understanding
    contentAnalysis := g.analyzeContent(content)

    // Step 2: Intent classification
    intent := g.classifyIntent(content, metadata)

    // Step 3: Lead scoring
    leadScore := g.calculateLeadScore(contentAnalysis, intent, metadata)

    // Step 4: Workflow recommendations
    actions := g.recommendActions(leadScore, intent, metadata)

    return &Analysis{
        Content:      contentAnalysis,
        Intent:       intent,
        LeadScore:    leadScore,
        Actions:      actions,
        Confidence:   contentAnalysis.Confidence,
    }, nil
}
```

### **Gemini Agent Workflow Steps**

1. **Intake Analysis**: Gemini analyzes incoming content (form, call transcript, chat, etc.)
2. **Context Understanding**: Understands business context (home remodeling industry)
3. **Intent Classification**: Quote request, information seeking, complaint, etc.
4. **Lead Qualification**: Scores lead quality (1-100) based on multiple factors
5. **Action Orchestration**: Determines next steps (CRM integration, notifications, etc.)
6. **Workflow Execution**: Executes the determined actions through MCP connectors

---

## 🔌 **MCP Framework for CRM Integration**

### **What is MCP in This Context?**

**MCP (Model Context Protocol)** provides a standardized way to integrate with multiple CRM systems dynamically. Think of it as a universal CRM adapter.

```go
// internal/mcp/crm_connector.go
type CRMConnector interface {
    Connect(credentials map[string]string) error
    CreateContact(contact *models.Contact) (*models.CRMContact, error)
    UpdateContact(id string, updates map[string]interface{}) error
    SearchContacts(criteria map[string]interface{}) ([]*models.CRMContact, error)
    CreateOpportunity(opportunity *models.Opportunity) error
    GetFieldMapping() map[string]string
}

// Specific implementations
type HubSpotConnector struct {
    apiKey    string
    client    *hubspot.Client
    mapping   map[string]string
}

type SalesforceConnector struct {
    oauth     *salesforce.OAuthConfig
    client    *salesforce.Client
    mapping   map[string]string
}

type PipedriveConnector struct {
    apiToken  string
    client    *pipedrive.Client
    mapping   map[string]string
}
```

### **Dynamic CRM Selection & Field Mapping**

The Gemini agent uses MCP to dynamically select and configure CRM integrations:

```go
// internal/workflow/crm_integration.go
func (w *WorkflowEngine) ExecuteCRMIntegration(analysis *Analysis, config *TenantConfig) error {
    // 1. Dynamic CRM connector selection
    connector := w.getCRMConnector(config.CRMProvider)

    // 2. Load tenant-specific field mapping
    mapping := config.CRMFieldMapping

    // 3. Transform data using Gemini insights
    contact := w.transformToContact(analysis, mapping)

    // 4. Execute CRM integration
    crmContact, err := connector.CreateContact(contact)
    if err != nil {
        return w.handleCRMError(err, contact)
    }

    // 5. Create opportunity if lead score is high
    if analysis.LeadScore > config.MinOpportunityScore {
        opportunity := w.createOpportunity(crmContact, analysis)
        return connector.CreateOpportunity(opportunity)
    }

    return nil
}
```

### **Example CRM Integration Configurations**

Each tenant has a configuration like this:

```json
{
  "crm_integration": {
    "enabled": true,
    "provider": "hubspot",
    "credentials_secret_name": "hubspot-api-key-tenant-123",
    "field_mapping": {
      "customer_name": "firstname",
      "customer_phone": "phone",
      "customer_email": "email",
      "lead_score": "hs_lead_score",
      "project_type": "custom_project_type",
      "timeline": "custom_timeline",
      "budget_range": "custom_budget",
      "source": "lead_source",
      "notes": "hs_note_body"
    },
    "push_immediately": true,
    "min_opportunity_score": 70,
    "auto_assign_sales_rep": true,
    "opportunity_pipeline": "New Leads"
  }
}
```

---

## 🔄 **Integrated Workflow: Gemini Agent + MCP CRM**

Here's how they work together in a real scenario:

### **Step 1: Gemini Analysis**
```
Input: "Hi, I need a kitchen remodel ASAP. Budget is around $50K. Please call me at 555-1234."

Gemini Analysis:
- Intent: Quote request
- Project: Kitchen remodel
- Urgency: High ("ASAP")
- Budget: High ($50K)
- Lead Score: 95/100
- Actions: [create_crm_contact, create_opportunity, send_urgent_alert]
```

### **Step 2: MCP CRM Integration**
```go
// Workflow orchestration
workflow := &WorkflowStep{
    Name: "high_value_lead_processing",
    Actions: []Action{
        {
            Type: "crm_integration",
            Provider: "salesforce", // from tenant config
            Data: map[string]interface{}{
                "firstname": "John",
                "phone": "555-1234",
                "lead_score__c": 95,
                "project_type__c": "Kitchen",
                "budget_range__c": "40-60K",
                "urgency__c": "High",
                "lead_source": "phone_call"
            }
        },
        {
            Type: "create_opportunity",
            Data: map[string]interface{}{
                "name": "Kitchen Remodel - John",
                "amount": 50000,
                "stage": "Qualification",
                "close_date": "2025-10-15"
            }
        },
        {
            Type: "alert_sales_team",
            Priority: "urgent",
            Message: "High-value kitchen lead ($50K) - immediate follow-up required"
        }
    }
}
```

---

## 🏗️ **Implementation Architecture**

### **Service Structure**
```
cmd/
├── gemini-agent/          # Main Gemini agent service
│   └── main.go           # Entry point with Gemini integration
├── workflow-engine/       # Workflow orchestration service
│   └── main.go           # MCP workflow execution
└── crm-connector/         # CRM integration service
    └── main.go           # MCP CRM connectors

internal/
├── ai/
│   └── gemini.go         # Vertex AI Gemini integration
├── mcp/
│   ├── crm_connector.go  # CRM connector interface
│   ├── hubspot.go        # HubSpot MCP implementation
│   ├── salesforce.go     # Salesforce MCP implementation
│   └── pipedrive.go      # Pipedrive MCP implementation
├── workflow/
│   ├── engine.go         # Workflow orchestration
│   ├── steps.go          # Individual workflow steps
│   └── conditions.go     # Conditional logic
```

### **Cloud Run Deployment**
```yaml
# deployments/cloud-run/gemini-agent.yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: gemini-agent
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/minScale: "1"
        autoscaling.knative.dev/maxScale: "1000"
    spec:
      containers:
      - image: gcr.io/account-strategy-464106/gemini-agent:latest
        env:
        - name: VERTEX_AI_PROJECT
          value: "account-strategy-464106"
        - name: VERTEX_AI_LOCATION
          value: "us-central1"
        - name: VERTEX_AI_MODEL
          value: "gemini-2.5-flash"
        resources:
          requests:
            cpu: 2
            memory: 4Gi
          limits:
            cpu: 4
            memory: 16Gi
```

---

## 💡 **Key Integration Points**

### **1. Gemini Intelligence Tasks**
- **Content Analysis**: Understanding customer communications
- **Intent Classification**: What does the customer want?
- **Lead Scoring**: How valuable is this lead?
- **Workflow Orchestration**: What actions should we take?

### **2. MCP CRM Integration Tasks**
- **Dynamic Provider Selection**: Choose HubSpot/Salesforce/etc. per tenant
- **Field Mapping**: Transform AI insights to CRM fields
- **Contact Management**: Create/update CRM records
- **Opportunity Creation**: Generate sales opportunities
- **Pipeline Management**: Move leads through sales process

### **3. Configuration-Driven Processing**
Each tenant can configure:
- Which CRM to use
- How to map fields
- When to create opportunities
- What alerts to send
- Custom workflow rules

---

## 🎯 **Summary: You Were Right to Ask!**

The **Go/Gemini agent** and **MCP CRM integration** are central to the system but weren't clearly explained in the original walkthrough. Here's the corrected understanding:

1. **Go/Gemini Agent** = The intelligent Cloud Run service that processes all communications using Vertex AI
2. **MCP Framework** = The standardized CRM integration system that connects to multiple CRM providers dynamically
3. **Integration** = Gemini provides intelligence, MCP executes the CRM actions based on that intelligence

This architecture allows each tenant to have completely customized workflows while sharing the same intelligent processing engine!

Thank you for catching this - it's a critical component that needed proper explanation!