class Ailloy < Formula
  desc "Package manager for AI instructions"
  homepage "https://github.com/nimble-giant/ailloy"
  version "0.6.18"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/nimble-giant/ailloy/releases/download/v#{version}/ailloy-darwin-arm64"
      sha256 "d410762b53f1111790328fb136c95dd77b73e37a795ea00b1754939a872427eb"
    end
    on_intel do
      url "https://github.com/nimble-giant/ailloy/releases/download/v#{version}/ailloy-darwin-amd64"
      sha256 "e99c75c95f88d37730d509ac48395d1420440cdf9a470a10167c71e92f7746d6"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/nimble-giant/ailloy/releases/download/v#{version}/ailloy-linux-arm64"
      sha256 "8857e704891c1f54531fdb5561f099001a01e1713c1884467fead35a132513e6"
    end
    on_intel do
      url "https://github.com/nimble-giant/ailloy/releases/download/v#{version}/ailloy-linux-amd64"
      sha256 "639cc758beed3c6eb0164f495a3c0033d728f29fdeb555daa3ea0e2caddd1c5d"
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
