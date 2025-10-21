#!/usr/bin/env bash
# scripts/setup-git-hooks.sh - Setup Git hooks for Conventional Commits
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
GIT_HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

echo "🔧 Setting up Git hooks for Conventional Commits..."

# Check if we're in a git repository
if [ ! -d "$PROJECT_ROOT/.git" ]; then
    echo "❌ Error: Not a git repository"
    exit 1
fi

# Create hooks directory if it doesn't exist
mkdir -p "$GIT_HOOKS_DIR"

# Install commit-msg hook using validate-commit-msg.sh
echo "📝 Installing commit-msg hook..."
cat > "$GIT_HOOKS_DIR/commit-msg" <<'EOF'
#!/usr/bin/env bash
# Git commit-msg hook - Validates commit messages
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

exec "$PROJECT_ROOT/scripts/validate-commit-msg.sh" "$1"
EOF

chmod +x "$GIT_HOOKS_DIR/commit-msg"

echo "✅ Git hooks installed successfully!"
echo ""
echo "📋 Commit message format:"
echo "   <type>(<scope>): <subject>"
echo ""
echo "   Types: feat, fix, docs, style, refactor, perf, test, chore, ci, build, revert"
echo ""
echo "   Examples:"
echo "   - feat(api): add device registration endpoint"
echo "   - fix(database): resolve connection pool leak"
echo "   - docs(readme): update installation instructions"
echo "   - perf(query): optimize database indexes"
echo ""
echo "💡 Tip: Your commit messages will be validated automatically"
