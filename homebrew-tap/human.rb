# typed: false
# frozen_string_literal: true

# Homebrew formula for the Human compiler.
# This file is auto-updated by GoReleaser on each release.
# Manual edits will be overwritten.

class Human < Formula
  desc "Programming language where you write in English and get production-ready applications"
  homepage "https://github.com/barun-bash/human"
  license "MIT"
  version "0.4.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/barun-bash/human/releases/download/v#{version}/human_#{version}_darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_DARWIN_ARM64"
    else
      url "https://github.com/barun-bash/human/releases/download/v#{version}/human_#{version}_darwin_amd64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_DARWIN_AMD64"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/barun-bash/human/releases/download/v#{version}/human_#{version}_linux_arm64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_ARM64"
    else
      url "https://github.com/barun-bash/human/releases/download/v#{version}/human_#{version}_linux_amd64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_AMD64"
    end
  end

  def install
    bin.install "human"
  end

  test do
    assert_match "human v#{version}", shell_output("#{bin}/human --version")
  end
end
