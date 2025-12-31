package config

// GlobalConfigTemplate is the default template for ~/.config/choir/config.yaml.
// It includes comments explaining each option.
const GlobalConfigTemplate = `# Choir global configuration
# Location: ~/.config/choir/config.yaml

# Schema version (required)
version: 1

# Default backend when --backend flag not specified
default_backend: local

# Credential paths (defaults shown)
credentials:
  claude_config: ~/.claude
  ssh_keys: ~/.ssh
  git_config: ~/.gitconfig
  github_cli: ~/.config/gh

# Backend definitions
backends:
  local:
    type: lima

    # Default VM resources
    cpus: 4
    memory: 4GB
    disk: 50GB

    # Lima-specific options: vz (recommended) or qemu
    vm_type: vz

  # Future backend example (not implemented in v1)
  # aws:
  #   type: ec2
  #   region: us-west-2
  #   instance_type: t3.medium
`

// ProjectConfigTemplate is the default template for .choir.yaml.
// It includes commented examples for all configuration options.
const ProjectConfigTemplate = `# Choir project configuration
# Location: .choir.yaml (repository root)

# Schema version (required)
version: 1

# Base image override (optional)
# If omitted, backend uses its default base image
# base_image: ubuntu:24.04

# Additional system packages to install
# packages:
#   - python3
#   - python3-pip
#   - redis-tools

# Environment variables
# env:
#   # Literal value
#   NODE_ENV: development
#
#   # Reference host environment variable
#   DATABASE_URL: ${DATABASE_URL}
#
#   # Reference file contents (entire file becomes value)
#   API_KEY:
#     from_file: ~/.secrets/project-api-key

# Files to copy into VM
# files:
#   - source: ~/.aws
#     target: /home/ubuntu/.aws
#     readonly: true
#
#   - source: .env.local
#     target: /home/ubuntu/workspace/.env.local

# Commands to run after clone, before agent is ready
# Working directory: repository root
# Run as: default VM user (e.g., ubuntu)
# setup:
#   - docker compose up -d
#   - npm install

# Resource overrides (optional)
# resources:
#   memory: 8GB
#   cpus: 8
#   disk: 100GB

# Branch naming convention
# Final branch name: {prefix}{task-id}
branch_prefix: agent/
`

// ProjectConfigMinimalTemplate is a minimal template without comments.
const ProjectConfigMinimalTemplate = `version: 1
branch_prefix: agent/
`
