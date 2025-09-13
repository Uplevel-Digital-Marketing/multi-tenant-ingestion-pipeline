# Research Parameters

## Date Scope
- Current Date: September 13, 2025
- Research Window: August 1, 2025 to September 13, 2025 (Latest documentation only)
- Focus: Most recent Google Cloud services and features announced since August 2025

## Search Constraints
- MANDATORY: Google Cloud Platform services prioritized
- MANDATORY: Documentation updated since August 2025 only
- MANDATORY: Current library versions and APIs
- MANDATORY: Verified implementation patterns for multi-tenant systems

## Project Focus: Multi-Tenant Ingestion Pipeline
### Core Requirements
- **Inbound Processing**: JSON requests from forms/calls/calendar bookings
- **AI Processing**: Gemini/Go agent integration
- **Database**: Cloud Spanner (upai-customers instance, agent_platform database)
- **Workflow Steps**: Communication detection, audio processing, spam validation, service area validation, CRM integration, notifications

### Current Infrastructure
- Cloud Spanner Instance: upai-customers (Enterprise, us-central1, Autoscaling)
- Database: agent_platform (Google Standard SQL)
- Architecture: Flexible data values vs rigid JSON schemas

## Research Targets
- Latest GCP AI/ML services (Gemini integration)
- Advanced Cloud Spanner features and patterns
- Multi-tenant architecture best practices on GCP
- Real-time processing and workflow orchestration
- Audio processing and transcription services
- Integration patterns with external services (MCP, SendGrid)