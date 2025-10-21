#!/bin/bash
#
# EdgeLink ADR Creation Script
#
# 自动创建新的架构决策记录（Architecture Decision Record）
#
# Usage:
#   ./scripts/new-adr.sh "Decision Title"
#   ./scripts/new-adr.sh "使用Redis作为缓存层"
#

set -euo pipefail

# Configuration
ADR_DIR="docs/adr"
TEMPLATE_FILE="$ADR_DIR/template.md"
README_FILE="$ADR_DIR/README.md"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

show_usage() {
    cat <<EOF
Usage: $0 "<Decision Title>"

Create a new Architecture Decision Record (ADR) from template.

Arguments:
  Decision Title    Title of the decision (in Chinese or English)

Examples:
  $0 "使用Redis作为缓存层"
  $0 "Use gRPC for internal communication"
  $0 "Implement rate limiting with token bucket"

Options:
  -h, --help       Show this help message

The script will:
  1. Find the next available ADR number (e.g., 0014)
  2. Convert title to kebab-case filename
  3. Create ADR file from template
  4. Open the file in your editor (if EDITOR is set)

EOF
}

# Get next ADR number
get_next_adr_number() {
    local max_num=0
    
    # Find all ADR files and extract numbers
    for file in "$ADR_DIR"/*.md; do
        if [[ $(basename "$file") =~ ^([0-9]{4})- ]]; then
            local num=$((10#${BASH_REMATCH[1]}))
            if [ $num -gt $max_num ]; then
                max_num=$num
            fi
        fi
    done
    
    # Return next number, padded to 4 digits
    printf "%04d" $((max_num + 1))
}

# Convert title to kebab-case filename
title_to_filename() {
    local title="$1"
    
    # Convert to lowercase
    title=$(echo "$title" | tr '[:upper:]' '[:lower:]')
    
    # Replace spaces with hyphens
    title=$(echo "$title" | tr ' ' '-')
    
    # Remove special characters except hyphens and Chinese characters
    title=$(echo "$title" | sed 's/[^a-z0-9\-\u4e00-\u9fa5]//g')
    
    # Remove multiple consecutive hyphens
    title=$(echo "$title" | sed 's/--*/-/g')
    
    # Remove leading/trailing hyphens
    title=$(echo "$title" | sed 's/^-//; s/-$//')
    
    echo "$title"
}

# Create ADR file from template
create_adr_file() {
    local adr_number="$1"
    local title="$2"
    local filename="$3"
    local current_date=$(date +%Y-%m-%d)
    local current_user=$(git config user.name 2>/dev/null || echo "Unknown")
    
    local adr_file="$ADR_DIR/${adr_number}-${filename}.md"
    
    # Check if file already exists
    if [ -f "$adr_file" ]; then
        log_error "ADR file already exists: $adr_file"
        exit 1
    fi
    
    # Copy template and replace placeholders
    cp "$TEMPLATE_FILE" "$adr_file"
    
    # Replace NNNN with actual number
    sed -i "s/ADR-NNNN/ADR-${adr_number}/g" "$adr_file"
    
    # Replace title
    sed -i "s/\[简短的决策标题\]/${title}/g" "$adr_file"
    
    # Replace date
    sed -i "s/YYYY-MM-DD/${current_date}/g" "$adr_file"
    
    # Replace author
    sed -i "s/\[名字\/团队\]/${current_user}/g" "$adr_file"
    
    log_info "Created ADR file: $adr_file"
    echo "$adr_file"
}

# Update README index
update_readme_index() {
    local adr_number="$1"
    local title="$2"
    local filename="$3"
    local current_date=$(date +%Y-%m-%d)
    
    local adr_link="[${adr_number}](${adr_number}-${filename}.md)"
    local new_row="| ${adr_link} | ${title} | Proposed | ${current_date} |"
    
    log_info "Please manually add the following row to $README_FILE:"
    echo ""
    echo "$new_row"
    echo ""
    log_warn "Automatic README update not implemented yet."
}

# Main logic
main() {
    # Check arguments
    if [ $# -eq 0 ]; then
        log_error "Missing decision title"
        show_usage
        exit 1
    fi
    
    if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
        show_usage
        exit 0
    fi
    
    local title="$1"
    
    # Check if ADR directory exists
    if [ ! -d "$ADR_DIR" ]; then
        log_error "ADR directory not found: $ADR_DIR"
        exit 1
    fi
    
    # Check if template exists
    if [ ! -f "$TEMPLATE_FILE" ]; then
        log_error "Template file not found: $TEMPLATE_FILE"
        exit 1
    fi
    
    # Get next ADR number
    local adr_number=$(get_next_adr_number)
    log_info "Next ADR number: $adr_number"
    
    # Convert title to filename
    local filename=$(title_to_filename "$title")
    log_info "Filename: ${adr_number}-${filename}.md"
    
    # Create ADR file
    local adr_file=$(create_adr_file "$adr_number" "$title" "$filename")
    
    # Update README (manual for now)
    update_readme_index "$adr_number" "$title" "$filename"
    
    # Open in editor if EDITOR is set
    if [ -n "${EDITOR:-}" ]; then
        log_info "Opening ADR in editor: $EDITOR"
        $EDITOR "$adr_file"
    else
        log_info "ADR created successfully!"
        log_warn "Tip: Set EDITOR environment variable to auto-open files"
        log_info "Example: export EDITOR=vim"
    fi
    
    echo ""
    log_info "Next steps:"
    echo "  1. Edit the ADR file: $adr_file"
    echo "  2. Fill in all sections (Context, Options, Decision, Consequences)"
    echo "  3. Add the ADR to the README index in $README_FILE"
    echo "  4. Commit and create PR for team review"
    echo "  5. After approval, update status to 'Accepted'"
}

# Run main
main "$@"
