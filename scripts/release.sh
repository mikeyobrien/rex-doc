#!/bin/sh
# Release rdxt: build for macOS and Linux, publish a GitHub release, and
# update the Homebrew formula in travelingflwr/homebrew-tap.
#
# Usage: scripts/release.sh v0.1.0 [notes-file]
set -eu

version=${1:?usage: scripts/release.sh vX.Y.Z [notes-file]}
notes=${2:-}
case $version in v[0-9]*) ;; *) echo "version must look like v0.1.0" >&2; exit 2 ;; esac
plain=${version#v}
repo=travelingflwr/rex-doc
tap=travelingflwr/homebrew-tap
root=$(cd "$(dirname "$0")/.." && pwd)
dist="$root/dist/$version"

cd "$root"
if [ -n "$(git status --porcelain)" ]; then
  echo "commit your changes first" >&2
  exit 1
fi
go vet ./...
go test ./...

rm -rf "$dist"
mkdir -p "$dist"
for target in darwin/arm64 darwin/amd64 linux/arm64 linux/amd64; do
  os=${target%/*}
  arch=${target#*/}
  name="rdxt_${plain}_${os}_${arch}"
  stage="$dist/$name"
  mkdir -p "$stage"
  GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X main.version=$version" -o "$stage/rdxt" ./cmd/rdxt
  cp LICENSE README.md "$stage/"
  tar -C "$dist" -czf "$dist/$name.tar.gz" "$name"
  rm -rf "$stage"
  echo "built $name.tar.gz"
done
(cd "$dist" && shasum -a 256 ./*.tar.gz | sed 's| \./| |' > checksums.txt)

git tag -a "$version" -m "rdxt $version"
git push origin "$version"
if [ -n "$notes" ]; then
  gh release create "$version" "$dist"/*.tar.gz "$dist/checksums.txt" --repo "$repo" --title "rdxt $version" --notes-file "$notes"
else
  gh release create "$version" "$dist"/*.tar.gz "$dist/checksums.txt" --repo "$repo" --title "rdxt $version" --generate-notes
fi

# The Homebrew formula.
sum() { grep "rdxt_${plain}_$1.tar.gz" "$dist/checksums.txt" | cut -d' ' -f1; }
url="https://github.com/$repo/releases/download/$version"
formula=$(cat <<EOF
class Rdxt < Formula
  desc "Markdown, made for Rex: read and write Markdown as real text in your terminal"
  homepage "https://github.com/$repo"
  version "$plain"
  license "MIT"

  on_macos do
    on_arm do
      url "$url/rdxt_${plain}_darwin_arm64.tar.gz"
      sha256 "$(sum darwin_arm64)"
    end
    on_intel do
      url "$url/rdxt_${plain}_darwin_amd64.tar.gz"
      sha256 "$(sum darwin_amd64)"
    end
  end

  on_linux do
    on_arm do
      url "$url/rdxt_${plain}_linux_arm64.tar.gz"
      sha256 "$(sum linux_arm64)"
    end
    on_intel do
      url "$url/rdxt_${plain}_linux_amd64.tar.gz"
      sha256 "$(sum linux_amd64)"
    end
  end

  def install
    bin.install "rdxt"
    bin.install_symlink "rdxt" => "rex-doc"
  end

  def caveats
    <<~EOS
      Make rdxt feel built into Rex (Cmd-click, the palette, agent skills):
        rdxt setup
      ★ Like it? A star helps others find it: https://github.com/$repo
    EOS
  end

  test do
    assert_match "rdxt $version", shell_output("#{bin}/rdxt --version")
  end
end
EOF
)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
gh repo clone "$tap" "$work/tap" -- --quiet
mkdir -p "$work/tap/Formula"
printf '%s\n' "$formula" > "$work/tap/Formula/rdxt.rb"
cd "$work/tap"
git add Formula/rdxt.rb
git commit -qm "rdxt $version"
git push -q
echo "released $version: https://github.com/$repo/releases/tag/$version"
