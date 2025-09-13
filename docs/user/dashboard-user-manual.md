# Dashboard User Manual

## Table of Contents
- [Overview](#overview)
- [Getting Started](#getting-started)
- [Dashboard Navigation](#dashboard-navigation)
- [Call Analytics](#call-analytics)
- [Lead Management](#lead-management)
- [CRM Integration Status](#crm-integration-status)
- [Performance Metrics](#performance-metrics)
- [Reports and Exports](#reports-and-exports)
- [Settings and Configuration](#settings-and-configuration)
- [Troubleshooting](#troubleshooting)

## Overview

The Ingestion Pipeline Dashboard provides real-time visibility into your call processing, lead generation, and CRM integration performance. Monitor key metrics, analyze call data, and optimize your lead conversion process all from a single interface.

### Key Features
- **Real-time Call Processing**: Monitor incoming calls as they're processed
- **AI-Powered Analytics**: View AI-generated lead scores and insights
- **CRM Integration Status**: Track successful pushes to your CRM system
- **Performance Metrics**: Monitor system health and processing times
- **Custom Reports**: Generate detailed reports for specific time periods
- **Multi-tenant Support**: Isolated data views for each business location

## Getting Started

### First-Time Login

1. **Access the Dashboard**
   - Navigate to `https://dashboard.pipeline.com`
   - Enter your tenant credentials provided during onboarding

2. **Initial Setup Verification**
   - Verify your business information is correct
   - Check CallRail integration status
   - Confirm CRM connection is active
   - Review notification preferences

3. **Dashboard Orientation**
   - Take the guided tour (first-time users)
   - Customize your dashboard layout
   - Set up alerts and notifications

### User Roles and Permissions

| Role | Permissions | Access Level |
|------|-------------|--------------|
| **Owner** | Full access, billing, user management | All features |
| **Manager** | View all data, manage settings | Reports, settings, users |
| **Agent** | View assigned leads, basic reporting | Leads, basic reports |
| **Viewer** | Read-only access to reports | Reports only |

## Dashboard Navigation

### Main Navigation Menu

#### 🏠 **Home Dashboard**
Quick overview of key metrics and recent activity.

**Widgets Available:**
- Today's call summary
- Lead score distribution
- CRM push status
- Recent high-value leads
- System health indicators

#### 📞 **Calls**
Detailed view of all processed calls with filtering and search capabilities.

**Features:**
- Real-time call feed
- Advanced filtering (date, lead score, project type)
- Audio playback and transcripts
- AI analysis results
- Export capabilities

#### 🎯 **Leads**
Lead management interface with CRM integration status.

**Features:**
- Lead pipeline visualization
- Follow-up task management
- Lead scoring analytics
- CRM sync status
- Duplicate detection alerts

#### 📊 **Analytics**
Comprehensive reporting and analytics tools.

**Available Reports:**
- Call volume trends
- Lead quality metrics
- Conversion analytics
- Performance benchmarks
- Cost analysis

#### ⚙️ **Settings**
System configuration and integration management.

**Configuration Options:**
- CallRail webhook settings
- CRM integration setup
- Notification preferences
- User management
- Billing and usage

### Quick Actions Toolbar

Located at the top of every page:
- 🔍 **Global Search**: Search calls, leads, or contacts
- 🔔 **Notifications**: Recent alerts and system notifications
- 📥 **Export**: Quick export of current view data
- ❓ **Help**: Context-sensitive help and documentation

## Call Analytics

### Call Feed Interface

#### Real-Time Call Display
Calls appear in real-time as they're processed through the pipeline.

```
┌─────────────────────────────────────────────────────────────────┐
│ 🔴 LIVE    📞 +1 (555) 123-4567 → Main Office                  │
│            ⏱️  2:34    🏠 Kitchen Remodel    ⭐ Score: 87      │
│            📍 Los Angeles, CA    💬 "Looking for quote..."      │
│            🎯 High Priority    ✅ Pushed to HubSpot             │
└─────────────────────────────────────────────────────────────────┘
```

#### Call Details View
Click any call to view comprehensive details:

**Basic Information:**
- Call ID and timestamp
- Customer phone number and name
- Call duration and status
- Business phone number called

**AI Analysis Results:**
- **Project Type**: Kitchen, Bathroom, Whole Home, etc.
- **Timeline**: Immediate, 1-3 months, 3-6 months, 6+ months
- **Budget Indicator**: High, Medium, Low
- **Intent**: Quote request, Information seeking, Appointment booking
- **Sentiment**: Positive, Neutral, Negative
- **Lead Score**: 0-100 AI-generated quality score

**Call Recording & Transcript:**
- Audio player with playback controls
- Full transcription with speaker identification
- Confidence scores for transcription accuracy
- Key moments and highlights

### Advanced Filtering

#### Filter Options
- **Date Range**: Custom date picker or preset ranges
- **Lead Score**: Minimum score threshold
- **Project Type**: Filter by detected project categories
- **Call Status**: Answered, missed, voicemail
- **CRM Status**: Successfully pushed, failed, pending
- **Location**: Filter by customer city/state

#### Search Functionality
- **Customer Information**: Search by phone, name, or email
- **Call Content**: Search within transcriptions
- **Call ID**: Direct lookup by CallRail call ID
- **Address**: Search by customer location

### Call Performance Metrics

#### Key Performance Indicators (KPIs)

**Call Volume Metrics:**
```
📞 Total Calls Today: 47
   ↗️ +12% vs yesterday
   📈 7-day average: 42

⏱️ Average Duration: 3:24
   ↗️ +8% vs last week
   🎯 Target: 3:00+

✅ Answer Rate: 89%
   ↗️ +3% vs last month
   🎯 Target: 85%+
```

**Lead Quality Metrics:**
```
⭐ Average Lead Score: 76
   ↗️ +5 points vs last week
   🎯 Target: 70+

🏆 High-Value Leads: 23
   (Score 80+)
   ↗️ +15% vs last week

🎯 Conversion Rate: 34%
   (Leads → Appointments)
   ↗️ +2% vs last month
```

## Lead Management

### Lead Pipeline View

#### Pipeline Stages
Visualize leads as they move through your sales process:

```
New Leads (15) → Contacted (8) → Qualified (5) → Appointment (3) → Closed (2)
     ↓              ↓              ↓               ↓              ↓
   87 avg          82 avg         85 avg          91 avg         94 avg
   score           score          score           score          score
```

#### Lead Cards
Each lead displays essential information:

```
┌─────────────────────────────────────────────────────────────────┐
│ 👤 John Smith                                   ⭐ Score: 87    │
│ 📞 (555) 123-4567                              🏠 Kitchen       │
│ 📍 Los Angeles, CA                             ⏰ 2 hours ago   │
│ 💰 Budget: Medium                              🎯 Quote Request │
│ ✅ In HubSpot    📅 Follow-up: Today 3:00 PM                   │
│                                                                 │
│ 🎧 "Hi, I'm interested in a kitchen remodel..."                │
│                                                                 │
│ [📞 Call Back] [📧 Email] [📅 Schedule] [👁️ View Details]      │
└─────────────────────────────────────────────────────────────────┘
```

### Lead Scoring System

#### Score Breakdown
Understand how AI calculates lead scores:

**Score Components (0-100):**
- **Intent Clarity (25 points)**: How clear is the customer's intent?
- **Project Scope (20 points)**: Size and complexity of potential project
- **Timeline Urgency (20 points)**: How soon does customer need work?
- **Budget Indicators (15 points)**: Signals about customer's budget
- **Contact Quality (10 points)**: Quality of contact information
- **Engagement Level (10 points)**: Customer engagement during call

**Score Ranges:**
- **90-100**: 🏆 Exceptional lead - immediate follow-up required
- **80-89**: ⭐ High-quality lead - prioritize for contact
- **70-79**: ✅ Good lead - standard follow-up process
- **60-69**: ⚠️ Fair lead - nurture with marketing
- **Below 60**: ❌ Low-quality lead - minimal effort

### CRM Integration Dashboard

#### Sync Status Overview
Monitor the health of your CRM integration:

```
🟢 HubSpot Integration: Active
   ✅ Last sync: 2 minutes ago
   📊 Success rate: 98.5% (last 24h)
   📈 Total contacts created: 1,247

📋 Recent Sync Activity:
   ✅ John Smith - Contact created (2 min ago)
   ✅ Jane Doe - Deal updated (5 min ago)
   ⚠️ Bob Wilson - Duplicate detected (8 min ago)
   ❌ Alice Brown - Field mapping error (12 min ago)
```

#### Field Mapping Status
Verify that call data is properly mapped to CRM fields:

| Call Data | CRM Field | Status | Last Updated |
|-----------|-----------|--------|--------------|
| Customer Name | First Name, Last Name | ✅ Active | - |
| Phone Number | Phone | ✅ Active | - |
| Project Type | Custom: AI Project Type | ✅ Active | 2 days ago |
| Lead Score | Custom: AI Lead Score | ✅ Active | - |
| Call Recording | Custom: Recording URL | ⚠️ Warning | Field not found |

#### Duplicate Detection
Monitor and manage duplicate contact prevention:

```
🛡️ Duplicate Prevention: Active
   📊 Duplicates prevented: 23 (this month)
   🔍 Match criteria: Phone number, Email
   ⚙️ Action: Update existing contact

Recent Duplicates Detected:
• (555) 123-4567 - Updated existing contact (John Smith)
• jane@email.com - Merged with existing lead (Jane Doe)
```

## Performance Metrics

### System Health Dashboard

#### Real-Time Status
Monitor system performance and health:

```
🟢 System Status: All Systems Operational

📊 Performance Metrics (Last Hour):
┌─────────────────────────────────────────────────────────────────┐
│ ⚡ Processing Speed                                              │
│    Average: 1.2 seconds                              🎯 <2.0s   │
│    ████████████████████████████████████░░░ 92%                  │
│                                                                 │
│ 🎯 Success Rate                                                 │
│    Webhook Processing: 99.8%                         🎯 >99%    │
│    CRM Push: 98.2%                                   🎯 >95%    │
│    Transcription: 96.5%                              🎯 >95%    │
│                                                                 │
│ 📈 Volume                                                       │
│    Calls/Hour: 12                                               │
│    Peak Hour: 2:00 PM (34 calls)                               │
└─────────────────────────────────────────────────────────────────┘
```

#### Error Monitoring
Track and analyze system errors:

```
⚠️ Recent Issues (Last 24 Hours):
┌─────────────────────────────────────────────────────────────────┐
│ 🔴 2 CRM Push Failures                             ⏰ 3:22 PM   │
│    → HubSpot API rate limit exceeded                            │
│    → Resolution: Automatic retry successful                     │
│                                                                 │
│ 🟡 1 Transcription Warning                          ⏰ 1:45 PM   │
│    → Low confidence score (78%)                                 │
│    → Call duration: 4:23 (background noise detected)           │
│                                                                 │
│ 🟢 No authentication errors                         ⏰ All day   │
│ 🟢 No webhook failures                              ⏰ All day   │
└─────────────────────────────────────────────────────────────────┘
```

### Cost Analytics

#### Usage and Billing Overview
Track costs and usage patterns:

```
💰 Current Month Usage:
┌─────────────────────────────────────────────────────────────────┐
│ 📞 Calls Processed: 847                                         │
│    💵 Processing cost: $42.35                                   │
│    📊 Average per call: $0.05                                   │
│                                                                 │
│ 🎤 Audio Transcription: 42.3 hours                             │
│    💵 Speech-to-Text: $61.15                                    │
│    📊 Per minute: $0.024                                        │
│                                                                 │
│ 🧠 AI Analysis: 847 requests                                    │
│    💵 Vertex AI: $21.18                                         │
│    📊 Per analysis: $0.025                                      │
│                                                                 │
│ 📦 Storage: 12.4 GB                                            │
│    💵 Cloud Storage: $3.72                                      │
│                                                                 │
│ 💰 Total Monthly Cost: $128.40                                  │
│    📈 Projected: $145.20                                        │
└─────────────────────────────────────────────────────────────────┘
```

## Reports and Exports

### Standard Reports

#### Daily Summary Report
Automated daily report sent via email:

```
📧 Daily Summary - September 13, 2025

📞 Call Activity:
• Total calls: 47 (↗️ +12% vs yesterday)
• Answered calls: 42 (89% answer rate)
• Average duration: 3:24
• Total talk time: 2.4 hours

🎯 Lead Generation:
• Total leads: 42
• High-value leads (80+): 18 (43%)
• Average lead score: 76
• Top project type: Kitchen (23 leads)

🔄 CRM Integration:
• Successful pushes: 41/42 (98%)
• Failed pushes: 1 (retried successfully)
• New contacts created: 35
• Duplicates prevented: 7

⚡ System Performance:
• Average processing time: 1.2s
• Transcription accuracy: 96%
• No system issues detected
```

#### Weekly Performance Report
Comprehensive weekly analysis:

**Week of September 7-13, 2025**

```
📊 Weekly Highlights:
┌─────────────────────────────────────────────────────────────────┐
│ 📈 Call Volume Trend                                            │
│    Mon  Tue  Wed  Thu  Fri  Sat  Sun                           │
│     45   52   48   41   55   23   12                           │
│     ██   ███  ██   ██   ███  █         📊 Avg: 39/day          │
│                                                                 │
│ 🏆 Top Performers                                               │
│    Highest Lead Score: 98 (Kitchen project, LA)                │
│    Longest Call: 12:34 (Whole home remodel)                    │
│    Fastest Processing: 0.3s                                    │
│                                                                 │
│ 🎯 Lead Quality Distribution                                    │
│    90-100: ████████ 15%     (Exceptional)                      │
│    80-89:  ████████████ 28% (High Quality)                     │
│    70-79:  ████████████████ 35% (Good)                         │
│    60-69:  ████████ 15%     (Fair)                             │
│    <60:    ████ 7%          (Low)                              │
└─────────────────────────────────────────────────────────────────┘
```

### Custom Reports

#### Report Builder
Create custom reports with flexible filtering:

**Available Dimensions:**
- Time period (hour, day, week, month)
- Project type
- Lead score ranges
- Geographic location
- Call duration
- Customer source (Google, Facebook, etc.)

**Available Metrics:**
- Call volume
- Lead conversion rates
- Average lead scores
- Processing times
- CRM push success rates
- Cost per lead

#### Export Options

**CSV Export:**
```
Date,Call ID,Customer Phone,Lead Score,Project Type,Duration,CRM Status
2025-09-13,CAL123456,(555) 123-4567,87,Kitchen,00:03:24,Success
2025-09-13,CAL123457,(555) 987-6543,72,Bathroom,00:02:15,Success
```

**Excel Export:**
- Multiple worksheets (Summary, Calls, Leads, Errors)
- Formatted charts and graphs
- Pivot table ready data

**PDF Report:**
- Executive summary format
- Charts and visualizations
- Print-ready layout

## Settings and Configuration

### Account Settings

#### Business Information
Update your business details:

```
🏢 Business Information:
┌─────────────────────────────────────────────────────────────────┐
│ Company Name: [ACME Remodeling Company                        ] │
│ Phone: [(555) 123-4567                                        ] │
│ Email: [admin@acmeremodeling.com                              ] │
│ Address: [123 Main St, Los Angeles, CA 90210                  ] │
│ Timezone: [America/Los_Angeles                ▼]               │
│                                                                 │
│ Business Hours:                                                 │
│ Monday    [08:00] to [18:00]  ☑️ Open                          │
│ Tuesday   [08:00] to [18:00]  ☑️ Open                          │
│ Wednesday [08:00] to [18:00]  ☑️ Open                          │
│ Thursday  [08:00] to [18:00]  ☑️ Open                          │
│ Friday    [08:00] to [18:00]  ☑️ Open                          │
│ Saturday  [09:00] to [15:00]  ☑️ Open                          │
│ Sunday    ☐ Closed                                             │
└─────────────────────────────────────────────────────────────────┘
```

#### Service Area Configuration
Define your service coverage:

```
📍 Service Area:
┌─────────────────────────────────────────────────────────────────┐
│ Primary City: [Los Angeles                                     ] │
│ Service Radius: [25] miles                                      │
│                                                                 │
│ Included Cities:                                                │
│ • Los Angeles        [Remove]                                   │
│ • Beverly Hills      [Remove]                                   │
│ • Santa Monica       [Remove]                                   │
│ • West Hollywood     [Remove]                                   │
│                                                                 │
│ [+ Add City]                                                    │
│                                                                 │
│ Included Zip Codes:                                             │
│ 90210, 90211, 90212, 90213, 90401, 90402, 90403               │
│                                                                 │
│ [📝 Edit Zip Codes]                                            │
└─────────────────────────────────────────────────────────────────┘
```

### Integration Settings

#### CallRail Configuration
Manage your CallRail webhook integration:

```
📞 CallRail Integration:
┌─────────────────────────────────────────────────────────────────┐
│ Status: 🟢 Active                                               │
│ Company ID: [67890                                            ] │
│ Webhook URL: https://api.pipeline.com/v1/callrail/webhook       │
│ Last Webhook: 2 minutes ago                                     │
│                                                                 │
│ Webhook Events:                                                 │
│ ☑️ call_completed                                              │
│ ☐ call_started                                                │
│ ☐ text_message                                                │
│                                                                 │
│ Custom Fields:                                                  │
│ • tenant_id: tenant_acme_remodeling                            │
│ • callrail_company_id: 67890                                   │
│                                                                 │
│ [🔧 Test Webhook] [📋 Copy URL] [🔄 Regenerate Secret]        │
└─────────────────────────────────────────────────────────────────┘
```

#### CRM Integration Settings
Configure your CRM connection:

```
🔗 CRM Integration:
┌─────────────────────────────────────────────────────────────────┐
│ CRM Type: [HubSpot                    ▼]                       │
│ Status: 🟢 Connected                                            │
│ Last Sync: 30 seconds ago                                       │
│ Success Rate: 98.5% (24h)                                       │
│                                                                 │
│ Field Mappings:                                                 │
│ Customer Name → First Name, Last Name          ✅              │
│ Phone Number → Phone                           ✅              │
│ Email → Email                                  ✅              │
│ City → City                                    ✅              │
│ State → State                                  ✅              │
│ Project Type → AI Project Type                ✅              │
│ Lead Score → AI Lead Score                     ✅              │
│ Call Recording → Recording URL                 ⚠️ Field Missing │
│                                                                 │
│ [🔧 Test Connection] [📝 Edit Mappings] [🔄 Reconnect]       │
└─────────────────────────────────────────────────────────────────┘
```

### Notification Preferences

#### Alert Configuration
Set up alerts for important events:

```
🔔 Notification Settings:
┌─────────────────────────────────────────────────────────────────┐
│ High-Value Leads (Score 85+):                                  │
│ ☑️ Email notification                                          │
│ ☑️ SMS notification                                            │
│ ☐ Slack notification                                           │
│                                                                 │
│ System Errors:                                                  │
│ ☑️ Email notification                                          │
│ ☐ SMS notification                                             │
│ ☑️ Slack notification                                          │
│                                                                 │
│ Daily Reports:                                                  │
│ ☑️ Email at [09:00] AM                                        │
│ ☑️ Include performance metrics                                 │
│ ☑️ Include lead summary                                        │
│                                                                 │
│ Weekly Reports:                                                 │
│ ☑️ Email on [Monday] at [09:00] AM                            │
│ ☑️ Include cost analysis                                       │
│                                                                 │
│ Contact Information:                                            │
│ Primary Email: [admin@acmeremodeling.com                     ] │
│ SMS Number: [(555) 123-4567                                  ] │
│ Slack Webhook: [https://hooks.slack.com/...                  ] │
└─────────────────────────────────────────────────────────────────┘
```

## Troubleshooting

### Common Issues

#### "No calls appearing in dashboard"

**Possible Causes:**
1. CallRail webhook not configured
2. Incorrect tenant ID in webhook
3. Webhook authentication failure

**Resolution Steps:**
1. **Check CallRail Configuration:**
   - Verify webhook URL: `https://api.pipeline.com/v1/callrail/webhook`
   - Confirm events include "call_completed"
   - Check custom fields include tenant_id

2. **Test Webhook:**
   - Use CallRail test webhook feature
   - Check webhook logs in Settings → CallRail Integration

3. **Verify Tenant Mapping:**
   - Ensure tenant_id matches your account
   - Check CallRail company ID is correct

#### "CRM push failures"

**Possible Causes:**
1. CRM authentication expired
2. Field mapping errors
3. CRM rate limiting
4. Required fields missing

**Resolution Steps:**
1. **Check Connection:**
   - Go to Settings → CRM Integration
   - Click "Test Connection"
   - Reconnect if authentication failed

2. **Review Field Mappings:**
   - Check for missing required fields
   - Verify custom fields exist in CRM
   - Update field mappings if needed

3. **Check CRM Logs:**
   - Review recent sync activity
   - Look for specific error messages
   - Contact support if persistent issues

#### "Low transcription quality"

**Possible Causes:**
1. Poor call audio quality
2. Background noise
3. Multiple speakers
4. Non-English content

**Resolution Steps:**
1. **Review Call Quality:**
   - Check call duration (very short calls may have issues)
   - Look for background noise indicators
   - Consider calls from mobile vs landline

2. **Check AI Confidence:**
   - Review transcription confidence scores
   - Manually verify unclear transcriptions
   - Report persistent issues to support

### Support Resources

#### Getting Help

**In-Dashboard Help:**
- ❓ Help icon in top navigation
- Context-sensitive help on each page
- Guided tours for new features

**Documentation:**
- Complete user manual
- Video tutorials
- API documentation
- Integration guides

**Support Channels:**
- 📧 Email: support@pipeline.com
- 💬 Live chat (business hours)
- 📞 Phone: 1-800-PIPELINE
- 🎫 Support ticket system

**Community Resources:**
- User community forum
- Best practices blog
- Webinar series
- Customer success stories

#### Emergency Contacts

**System Outages:**
- Status page: status.pipeline.com
- Emergency hotline: 1-800-PIPELINE-911
- Slack alerts (if configured)

**Account Issues:**
- Account manager (enterprise customers)
- Billing support: billing@pipeline.com
- Security issues: security@pipeline.com

The dashboard provides comprehensive visibility and control over your lead generation pipeline. Use this manual as a reference for maximizing the value from your ingestion pipeline investment.