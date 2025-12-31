#!/bin/bash
# Read JSON input from stdin
input=$(cat)

# Debug: Log the input to a file for inspection (uncomment to debug)
# echo "$input" > /tmp/statusline_debug.json

# Example shape as of Claude 1.0.96 and 2025/08/28:
# {
#   "session_id": "2b6b04b2-2a9a-482b-aea2-9e0e7d8b125f",
#   "transcript_path": "/Users/lirum/.claude/projects/-Users-lirum-projects-vcto-asl-qb-api/2b6b04b2-2a9a-482b-aea2-9e0e7d8b125f.jsonl",
#   "cwd": "/Users/lirum/projects/vcto/asl/qb-api",
#   "model": {
#     "id": "claude-sonnet-4-20250514",
#     "display_name": "Sonnet 4"
#   },
#   "workspace": {
#     "current_dir": "/Users/lirum/projects/vcto/asl/qb-api",
#     "project_dir": "/Users/lirum/projects/vcto/asl/qb-api"
#   },
#   "version": "1.0.96",
#   "output_style": {
#     "name": "default"
#   },
#   "cost": {
#     "total_cost_usd": 0.03645105,
#     "total_duration_ms": 8464,
#     "total_api_duration_ms": 6453,
#     "total_lines_added": 0,
#     "total_lines_removed": 0
#   },
#   "exceeds_200k_tokens": false
# }


# Extract values using jq
MODEL_DISPLAY=$(echo "$input" | jq -r '.model.display_name')
TOTAL_COST=$(echo "$input" | jq -r '.cost.total_cost_usd')

echo "[$MODEL_DISPLAY] || \$$(printf "%.5f" "$TOTAL_COST")"
