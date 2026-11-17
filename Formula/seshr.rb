class Seshr < Formula
  desc "List Claude Code and OpenCode sessions by directory and resume them"
  homepage "https://github.com/alialjaffer/seshr"
  url "https://github.com/alialjaffer/seshr/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "PLACEHOLDER"
  license "MIT"
  head "https://github.com/alialjaffer/seshr.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = %W[-X main.version=#{version}]
    system "go", "build", *std_go_args(ldflags:), "."
  end

  test do
    assert_match "seshr version #{version}", shell_output("#{bin}/seshr --version")
  end
end
