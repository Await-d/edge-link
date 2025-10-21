#!/bin/sh
# .git/hooks/commit-msg
# Validates commit messages against Conventional Commits format

commit_msg_file=$1
commit_msg=$(cat "$commit_msg_file")

# Conventional Commits regex pattern
conventional_commit_regex='^(feat|fix|docs|style|refactor|perf|test|chore|ci|build|revert)(\(.+\))?: .{1,100}'

# Allow merge commits and revert commits
if echo "$commit_msg" | grep -qE "^Merge "; then
    exit 0
fi

if echo "$commit_msg" | grep -qE "^Revert "; then
    exit 0
fi

# Validate commit message format
if ! echo "$commit_msg" | grep -qE "$conventional_commit_regex"; then
    echo "❌ ERROR: Commit message does not follow Conventional Commits format"
    echo ""
    echo "Expected format:"
    echo "  <type>(<scope>): <subject>"
    echo ""
    echo "Types: feat, fix, docs, style, refactor, perf, test, chore, ci, build, revert"
    echo ""
    echo "Examples:"
    echo "  feat(api-gateway): add rate limiting"
    echo "  fix(database): resolve connection leak"
    echo "  docs(readme): update installation guide"
    echo ""
    echo "Your commit message:"
    echo "  $commit_msg"
    echo ""
    exit 1
fi

exit 0
