#!/usr/bin/env bash
set -euo pipefail

REPO="nimble-giant/ailloy"
FORMULA="Formula/ailloy.rb"

cd "$(dirname "$0")/.."

tag=$(gh release view --repo "$REPO" --json tagName --jq .tagName)
version="${tag#v}"

current=""
if [[ -f "$FORMULA" ]]; then
  current=$(grep -E '^\s*version "' "$FORMULA" | head -1 | sed -E 's/.*version "([^"]+)".*/\1/')
fi

if [[ "$current" == "$version" ]]; then
  echo "ailloy formula already at $version"
  exit 0
fi

echo "Updating ailloy formula: ${current:-none} -> $version"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

gh release download "$tag" --repo "$REPO" --pattern checksums.txt --dir "$tmp"

sha_for() {
  local name="$1"
  local sha
  sha=$(awk -v n="$name" '$2 == n { print $1 }' "$tmp/checksums.txt")
  if [[ -z "$sha" ]]; then
    echo "error: no sha256 for $name in checksums.txt" >&2
    exit 1
  fi
  printf '%s' "$sha"
}

darwin_arm64=$(sha_for "ailloy-darwin-arm64")
darwin_amd64=$(sha_for "ailloy-darwin-amd64")
linux_arm64=$(sha_for "ailloy-linux-arm64")
linux_amd64=$(sha_for "ailloy-linux-amd64")

mkdir -p Formula
cat > "$FORMULA" <<EOF
class Ailloy < Formula
  desc "Package manager for AI instructions"
  homepage "https://github.com/nimble-giant/ailloy"
  version "$version"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/nimble-giant/ailloy/releases/download/v#{version}/ailloy-darwin-arm64"
      sha256 "$darwin_arm64"
    end
    on_intel do
      url "https://github.com/nimble-giant/ailloy/releases/download/v#{version}/ailloy-darwin-amd64"
      sha256 "$darwin_amd64"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/nimble-giant/ailloy/releases/download/v#{version}/ailloy-linux-arm64"
      sha256 "$linux_arm64"
    end
    on_intel do
      url "https://github.com/nimble-giant/ailloy/releases/download/v#{version}/ailloy-linux-amd64"
      sha256 "$linux_amd64"
    end
  end

  def install
    suffix = OS.mac? ? "darwin" : "linux"
    arch = Hardware::CPU.arm? ? "arm64" : "amd64"
    bin.install "ailloy-#{suffix}-#{arch}" => "ailloy"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/ailloy --version")
  end
end
EOF

echo "Wrote $FORMULA at version $version"
