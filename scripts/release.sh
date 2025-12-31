#!/bin/bash
#
# Release script for choir
# Creates and pushes a new version tag following semantic versioning.
#
# Usage:
#   ./scripts/release.sh [major|minor|patch] [--force]
#
# Examples:
#   ./scripts/release.sh patch        # v0.0.3 -> v0.0.4
#   ./scripts/release.sh minor        # v0.0.3 -> v0.1.0
#   ./scripts/release.sh major        # v0.0.3 -> v1.0.0
#   ./scripts/release.sh patch --force  # Skip confirmation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Parse arguments
BUMP_TYPE=""
FORCE=false

for arg in "$@"; do
    case $arg in
        major|minor|patch)
            BUMP_TYPE="$arg"
            ;;
        --force|-f)
            FORCE=true
            ;;
        --help|-h)
            echo "Usage: $0 [major|minor|patch] [--force]"
            echo ""
            echo "Arguments:"
            echo "  major    Bump major version (v1.2.3 -> v2.0.0)"
            echo "  minor    Bump minor version (v1.2.3 -> v1.3.0)"
            echo "  patch    Bump patch version (v1.2.3 -> v1.2.4)"
            echo ""
            echo "Options:"
            echo "  --force, -f    Skip confirmation prompt"
            echo "  --help, -h     Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}error: unknown argument '$arg'${NC}" >&2
            echo "Usage: $0 [major|minor|patch] [--force]" >&2
            exit 1
            ;;
    esac
done

# Validate bump type is provided
if [[ -z "$BUMP_TYPE" ]]; then
    echo -e "${RED}error: bump type required (major, minor, or patch)${NC}" >&2
    echo "Usage: $0 [major|minor|patch] [--force]" >&2
    exit 1
fi

# Check we're on main branch
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
    echo -e "${RED}error: must be on main branch to release (currently on '$CURRENT_BRANCH')${NC}" >&2
    exit 1
fi

# Fetch latest from origin
echo "Fetching latest from origin..."
git fetch origin main --quiet

# Check local is up-to-date with origin/main
LOCAL_SHA=$(git rev-parse HEAD)
REMOTE_SHA=$(git rev-parse origin/main)

if [[ "$LOCAL_SHA" != "$REMOTE_SHA" ]]; then
    echo -e "${RED}error: local main is not up-to-date with origin/main${NC}" >&2
    echo "Run 'git pull origin main' first" >&2
    exit 1
fi

# Get latest tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

# Parse version (strip leading 'v')
VERSION="${LATEST_TAG#v}"
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"

# Default to 0 if parsing failed
MAJOR=${MAJOR:-0}
MINOR=${MINOR:-0}
PATCH=${PATCH:-0}

# Validate parsed version components are numeric
if ! [[ "$MAJOR" =~ ^[0-9]+$ ]] || ! [[ "$MINOR" =~ ^[0-9]+$ ]] || ! [[ "$PATCH" =~ ^[0-9]+$ ]]; then
    echo -e "${RED}error: could not parse version from tag '${LATEST_TAG}'${NC}" >&2
    echo "Expected format: vMAJOR.MINOR.PATCH (e.g., v1.2.3)" >&2
    exit 1
fi

# Calculate next version
case $BUMP_TYPE in
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        ;;
esac

NEXT_TAG="v${MAJOR}.${MINOR}.${PATCH}"

# Show what will happen
echo ""
echo -e "${GREEN}Release Summary${NC}"
echo "  Current version: ${LATEST_TAG}"
echo "  Next version:    ${NEXT_TAG}"
echo "  Bump type:       ${BUMP_TYPE}"
echo ""

# Confirm unless --force
if [[ "$FORCE" != true ]]; then
    read -p "Create and push tag ${NEXT_TAG}? [y/N] " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 0
    fi
fi

# Create and push tag
echo ""
echo "Creating tag ${NEXT_TAG}..."
git tag -a "${NEXT_TAG}" -m "Release ${NEXT_TAG}"

echo "Pushing tag to origin..."
git push origin "${NEXT_TAG}"

echo ""
echo -e "${GREEN}Release ${NEXT_TAG} created and pushed successfully!${NC}"
echo ""
echo "The release workflow will now:"
echo "  1. Build binaries for macOS and Linux (arm64, amd64)"
echo "  2. Create GitHub release with artifacts"
echo "  3. Update Homebrew tap (Quidge/homebrew-choir)"
echo ""
echo "Monitor progress: https://github.com/Quidge/choir/actions"
