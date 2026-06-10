class Lazyprd < Formula
  desc "Keyboard-driven TUI for browsing local /to-prd Markdown PRDs"
  homepage "https://github.com/baburyx/lazyprd"
  url "https://github.com/baburyx/lazyprd/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "REPLACE_WITH_RELEASE_TARBALL_SHA256"
  # Add a license file before publishing, then set this to the matching SPDX ID.
  # license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w")
  end

  test do
    system "#{bin}/lazyprd", "--help"
  end
end
