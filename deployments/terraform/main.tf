terraform {
  required_version = ">= 1.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }

  backend "gcs" {
    bucket = "terraform-state-account-strategy-464106"
    prefix = "ingestion-pipeline"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Variables
variable "project_id" {
  description = "GCP Project ID"
  type        = string
  default     = "account-strategy-464106"
}

variable "region" {
  description = "GCP Region"
  type        = string
  default     = "us-central1"
}

variable "spanner_instance" {
  description = "Cloud Spanner instance name"
  type        = string
  default     = "upai-customers"
}

variable "spanner_database" {
  description = "Cloud Spanner database name"
  type        = string
  default     = "agent_platform"
}

variable "environment" {
  description = "Environment (development, staging, production)"
  type        = string
  default     = "production"
}

# Service Account for Cloud Run services
resource "google_service_account" "ingestion_pipeline" {
  account_id   = "ingestion-pipeline"
  display_name = "Ingestion Pipeline Service Account"
  description  = "Service account for the multi-tenant ingestion pipeline"
}

# IAM roles for service account
resource "google_project_iam_member" "spanner_user" {
  project = var.project_id
  role    = "roles/spanner.databaseUser"
  member  = "serviceAccount:${google_service_account.ingestion_pipeline.email}"
}

resource "google_project_iam_member" "storage_admin" {
  project = var.project_id
  role    = "roles/storage.admin"
  member  = "serviceAccount:${google_service_account.ingestion_pipeline.email}"
}

resource "google_project_iam_member" "speech_user" {
  project = var.project_id
  role    = "roles/speech.editor"
  member  = "serviceAccount:${google_service_account.ingestion_pipeline.email}"
}

resource "google_project_iam_member" "aiplatform_user" {
  project = var.project_id
  role    = "roles/aiplatform.user"
  member  = "serviceAccount:${google_service_account.ingestion_pipeline.email}"
}

resource "google_project_iam_member" "secretmanager_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.ingestion_pipeline.email}"
}

resource "google_project_iam_member" "logging_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.ingestion_pipeline.email}"
}

resource "google_project_iam_member" "monitoring_writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.ingestion_pipeline.email}"
}

# Cloud Storage bucket for audio files
resource "google_storage_bucket" "audio_files" {
  name     = "tenant-audio-files-${var.project_id}"
  location = var.region

  uniform_bucket_level_access = true

  lifecycle_rule {
    condition {
      age = 90
    }
    action {
      type          = "SetStorageClass"
      storage_class = "COLDLINE"
    }
  }

  lifecycle_rule {
    condition {
      age = 365
    }
    action {
      type          = "SetStorageClass"
      storage_class = "ARCHIVE"
    }
  }

  lifecycle_rule {
    condition {
      age = 2555 # 7 years
    }
    action {
      type = "Delete"
    }
  }

  versioning {
    enabled = false
  }
}

# IAM for storage bucket
resource "google_storage_bucket_iam_member" "audio_bucket_admin" {
  bucket = google_storage_bucket.audio_files.name
  role   = "roles/storage.admin"
  member = "serviceAccount:${google_service_account.ingestion_pipeline.email}"
}

# Secret Manager secrets
resource "google_secret_manager_secret" "callrail_webhook_secret" {
  secret_id = "callrail-webhook-secret"

  replication {
    user_managed {
      replicas {
        location = var.region
      }
    }
  }
}

# Secret version (you'll need to add the actual secret value manually)
resource "google_secret_manager_secret_version" "callrail_webhook_secret_v1" {
  secret      = google_secret_manager_secret.callrail_webhook_secret.id
  secret_data = "CHANGE_ME_IN_CONSOLE" # Change this in the GCP console
}

# Cloud Run services will be deployed via Cloud Build
# But we can define some configuration here

# Enable required APIs
resource "google_project_service" "apis" {
  for_each = toset([
    "run.googleapis.com",
    "spanner.googleapis.com",
    "aiplatform.googleapis.com",
    "speech.googleapis.com",
    "storage.googleapis.com",
    "secretmanager.googleapis.com",
    "cloudtasks.googleapis.com",
    "cloudbuild.googleapis.com",
  ])

  project = var.project_id
  service = each.value

  disable_dependent_services = false
}

# Outputs
output "service_account_email" {
  description = "Email of the service account"
  value       = google_service_account.ingestion_pipeline.email
}

output "audio_bucket_name" {
  description = "Name of the audio storage bucket"
  value       = google_storage_bucket.audio_files.name
}

output "webhook_secret_name" {
  description = "Name of the webhook secret"
  value       = google_secret_manager_secret.callrail_webhook_secret.secret_id
}

# Cloud Run configurations (for reference)
locals {
  webhook_processor_config = {
    name     = "webhook-processor"
    memory   = "2Gi"
    cpu      = "2"
    timeout  = "900s"
    max_instances = 100
    env_vars = {
      GOOGLE_CLOUD_PROJECT = var.project_id
      GOOGLE_CLOUD_LOCATION = var.region
      SPANNER_INSTANCE = var.spanner_instance
      SPANNER_DATABASE = var.spanner_database
      AUDIO_STORAGE_BUCKET = google_storage_bucket.audio_files.name
      CALLRAIL_WEBHOOK_SECRET_NAME = google_secret_manager_secret.callrail_webhook_secret.secret_id
    }
  }

  api_gateway_config = {
    name     = "api-gateway"
    memory   = "1Gi"
    cpu      = "1"
    timeout  = "300s"
    max_instances = 50
    env_vars = {
      GOOGLE_CLOUD_PROJECT = var.project_id
      GOOGLE_CLOUD_LOCATION = var.region
      SPANNER_INSTANCE = var.spanner_instance
      SPANNER_DATABASE = var.spanner_database
    }
  }
}