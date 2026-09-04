#!/bin/sh
set -eu

repo="kokoichi206/talknote-cli"
install_dir="${TN_INSTALL_DIR:-${HOME}/.local/bin}"

fail() {
	echo "error: $1" >&2
	exit 1
}

os=$(uname -s)
case "$os" in
Darwin) os="darwin" ;;
Linux) os="linux" ;;
*) fail "unsupported OS: ${os} (only darwin and linux are supported)" ;;
esac

arch=$(uname -m)
case "$arch" in
x86_64 | amd64) arch="amd64" ;;
arm64 | aarch64) arch="arm64" ;;
*) fail "unsupported architecture: ${arch} (only amd64 and arm64 are supported)" ;;
esac

command -v curl >/dev/null 2>&1 || fail "curl is required"
if command -v shasum >/dev/null 2>&1; then
	checksum_tool="shasum"
elif command -v sha256sum >/dev/null 2>&1; then
	checksum_tool="sha256sum"
else
	fail "shasum or sha256sum is required"
fi

tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

tag=$(curl -fsSL -H "Accept: application/vnd.github+json" \
	"https://api.github.com/repos/${repo}/releases/latest" |
	sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' |
	head -n 1)
[ -n "$tag" ] || fail "could not determine the latest release tag"

version="${tag#v}"
asset="talknote-cli_${version}_${os}_${arch}.tar.gz"

curl -fsSL -o "${tmp_dir}/${asset}" \
	"https://github.com/${repo}/releases/download/${tag}/${asset}"
curl -fsSL -o "${tmp_dir}/checksums.txt" \
	"https://github.com/${repo}/releases/download/${tag}/checksums.txt"

expected_checksum=$(awk -v asset="$asset" '$2 == asset { print $1 }' "${tmp_dir}/checksums.txt")
[ -n "$expected_checksum" ] || fail "checksum for ${asset} was not found"
if [ "$checksum_tool" = "shasum" ]; then
	actual_checksum=$(shasum -a 256 "${tmp_dir}/${asset}" | awk '{ print $1 }')
else
	actual_checksum=$(sha256sum "${tmp_dir}/${asset}" | awk '{ print $1 }')
fi
[ "$actual_checksum" = "$expected_checksum" ] || fail "checksum verification failed for ${asset}"

tar -xzf "${tmp_dir}/${asset}" -C "$tmp_dir" tn
chmod +x "${tmp_dir}/tn"
mkdir -p "$install_dir"
mv "${tmp_dir}/tn" "${install_dir}/tn"

echo "tn ${tag} installed to ${install_dir}/tn"
