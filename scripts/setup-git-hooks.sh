#!/bin/bash
# Setup Git Hooks for Conventional Commits

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

# Install commit-msg hook
echo "📝 Installing commit-msg hook..."
cp "$SCRIPT_DIR/commit-msg-hook.sh" "$GIT_HOOKS_DIR/commit-msg"
chmod +x "$GIT_HOOKS_DIR/commit-msg"

echo "✅ Git hooks installed successfully!"
echo ""
echo "📋 Commit message format:"
echo "   <type>(<scope>): <subject>"
echo ""
echo "   Types: feat, fix, docs, style, refactor, perf, test, chore, ci, build"
echo ""
echo "   Examples:"
echo "   - feat(api): add device registration endpoint"
echo "   - fix(database): resolve connection pool leak"
echo "   - docs(readme): update installation instructions"
echo ""
echo "💡 Tip: Your commit messages will be validated automatically"
