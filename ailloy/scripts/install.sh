#!/bin/bash
# Auto-generated Ailloy Claude Code Plugin Installation Script
set -e

PLUGIN_NAME="ailloy"
PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "🦊 Installing Ailloy Plugin for Claude Code..."
echo "==========================================="

# Check requirements
if ! command -v gh &> /dev/null; then
    echo "⚠️  GitHub CLI (gh) not found. Please install it."
fi

if ! command -v git &> /dev/null; then
    echo "❌ Git is required but not installed."
    exit 1
fi

echo "✅ Requirements checked"
echo ""
echo "📦 Plugin Structure:"
echo "  Plugin Name: $PLUGIN_NAME"
echo "  Plugin Path: $PLUGIN_DIR"
echo "  Commands:    $(ls -1 "$PLUGIN_DIR/commands" | wc -l) available"
echo ""
echo "Available Commands:"
for cmd in "$PLUGIN_DIR/commands"/*.md; do
    basename "$cmd" .md | sed 's/^/  🔹 \/'$PLUGIN_NAME':/'
done
echo ""
echo "🎉 Plugin Ready!"
echo ""
echo "To use the plugin, run Claude Code with the --plugin-dir flag:"
echo "  claude --plugin-dir $PLUGIN_DIR"
echo ""
echo "Commands are namespaced as /$PLUGIN_NAME:<command-name>"
echo "Example: /$PLUGIN_NAME:create-issue"
