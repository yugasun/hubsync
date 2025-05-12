class Hubsync < Formula
  desc "Docker Hub Image Synchronization Tool"
  homepage "https://github.com/yugasun/hubsync"
  version "0.1.0"
  license "MIT"

  if OS.mac?
    if Hardware::CPU.intel?
      url "https://github.com/yugasun/hubsync/releases/download/v#{version}/hubsync-darwin-amd64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"  # darwin amd64
    elsif Hardware::CPU.arm?
      url "https://github.com/yugasun/hubsync/releases/download/v#{version}/hubsync-darwin-arm64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"  # darwin arm64
    end
  elsif OS.linux?
    if Hardware::CPU.intel?
      url "https://github.com/yugasun/hubsync/releases/download/v#{version}/hubsync-linux-amd64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"  # linux amd64
    elsif Hardware::CPU.arm?
      url "https://github.com/yugasun/hubsync/releases/download/v#{version}/hubsync-linux-arm64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"  # linux arm64
    end
  end

  def install
    # The downloaded binary is already executable, just need to move it to the bin directory
    bin.install Dir["hubsync-*"].first => "hubsync"
  end

  test do
    assert_match "HubSync version #{version}", shell_output("#{bin}/hubsync version")
  end

  def caveats
    <<~EOS
      To use HubSync, you need to create a .env file with your Docker credentials:
      
      DOCKER_USERNAME=your_username
      DOCKER_PASSWORD=your_token_or_password
      DOCKER_REPOSITORY=your_repository (optional)
      DOCKER_NAMESPACE=your_namespace (optional)
      
      Run 'hubsync --help' for more information.
    EOS
  end
end