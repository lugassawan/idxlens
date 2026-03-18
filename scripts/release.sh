#!/bin/sh
set -eu

usage() {
    echo "Usage: $0 <version|--auto>"
    echo ""
    echo "Create and push a release tag to trigger the GitHub Actions release workflow."
    echo ""
    echo "Examples:"
    echo "  $0 1.1.0"
    echo "  $0 v1.1.0"
    echo "  $0 --auto"
    exit 1
}

die() {
    echo "Error: $1" >&2
    exit 1
}

# Detect next version by analyzing commits since the last tag.
# Classifies commits by conventional commit prefix (feat:/fix:/etc.)
# and determines the appropriate semver bump.
detect_next_version() {
    latest_tag=$(git tag -l 'v*' | sort -V | tail -1)
    if [ -z "$latest_tag" ]; then
        latest_tag="v0.0.0"
        echo "No existing tags found. Starting from v0.0.0."
        commit_range="HEAD"
    else
        commit_range="${latest_tag}..HEAD"
    fi

    commits=$(git log "$commit_range" --oneline 2>/dev/null || true)
    if [ -z "$commits" ]; then
        echo "No commits since $latest_tag. Nothing to release."
        exit 0
    fi

    commit_count=$(echo "$commits" | wc -l | tr -d ' ')

    bump="patch"
    feat_commits=""
    fix_commits=""
    other_commits=""

    OLD_IFS="$IFS"
    IFS='
'
    for line in $commits; do
        subject=$(echo "$line" | sed 's/^[a-f0-9]* //')
        case "$subject" in
            feat:*)
                if [ "$bump" != "major" ]; then
                    bump="minor"
                fi
                feat_commits="${feat_commits}    - ${subject}
"
                ;;
            fix:*)
                fix_commits="${fix_commits}    - ${subject}
"
                ;;
            *)
                other_commits="${other_commits}    - ${subject}
"
                ;;
        esac
    done
    IFS="$OLD_IFS"

    # Parse current version
    major=$(echo "$latest_tag" | sed 's/^v//' | cut -d. -f1)
    minor=$(echo "$latest_tag" | sed 's/^v//' | cut -d. -f2)
    patch=$(echo "$latest_tag" | sed 's/^v//' | cut -d. -f3)

    case "$bump" in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
    esac

    next_version="v${major}.${minor}.${patch}"

    echo ""
    echo "Current version: $latest_tag"
    echo "Commits since $latest_tag: $commit_count"
    echo ""

    if [ -n "$feat_commits" ]; then
        echo "  Features (minor):"
        printf "%s" "$feat_commits"
        echo ""
    fi

    if [ -n "$fix_commits" ]; then
        echo "  Bug fixes (patch):"
        printf "%s" "$fix_commits"
        echo ""
    fi

    if [ -n "$other_commits" ]; then
        echo "  Other:"
        printf "%s" "$other_commits"
        echo ""
    fi

    echo "Next version: $next_version"
    echo ""

    printf "Continue with release? [y/N] "
    read -r confirm
    case "$confirm" in
        [yY]|[yY][eE][sS]) ;;
        *) echo "Aborted."; exit 0 ;;
    esac

    version="$next_version"
}

# Require exactly one argument
[ $# -eq 1 ] || usage

if [ "$1" = "--auto" ]; then
    detect_next_version
else
    version="$1"

    # Normalize: ensure v prefix
    case "$version" in
        v*) ;;
        *)  version="v${version}" ;;
    esac
fi

# Validate semver format (vX.Y.Z)
echo "$version" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$' \
    || die "invalid version format: $version (expected vX.Y.Z)"

# Ensure we're on main
branch=$(git rev-parse --abbrev-ref HEAD)
if [ "$branch" != "main" ]; then
    echo "Not on main (currently on $branch). Switching to main..."
    git checkout main
    git pull origin main
fi

# Ensure working tree is clean
if [ -n "$(git status --porcelain)" ]; then
    die "working tree is dirty — commit or stash changes first"
fi

# Ensure tag doesn't already exist
if git rev-parse "$version" >/dev/null 2>&1; then
    die "tag $version already exists"
fi

echo "Creating tag $version..."
git tag "$version"

echo "Pushing tag $version to origin..."
git push origin "$version"

echo ""
echo "Released $version — GitHub Actions will build and publish the release."
