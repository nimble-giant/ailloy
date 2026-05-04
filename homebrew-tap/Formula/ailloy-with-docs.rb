class AilloyWithDocs < Formula
  desc "Ailloy plus the docs extension preinstalled (rich in-CLI TUI)"
  homepage "https://github.com/nimble-giant/ailloy"
  version "0.1.0"
  license "Apache-2.0"

  # Sibling formula: depends on the main ailloy formula and runs the
  # extension installer post-install so users get the rich TUI on
  # their very first `ailloy docs` invocation, no consent prompt.
  depends_on "nimble-giant/tap/ailloy"

  def install
    # No binary of our own; everything lives in the dependency. We
    # install a tiny wrapper that exists only to signal "with-docs"
    # was used, which Homebrew's audit tools require for the formula
    # to have something to install.
    (bin/"ailloy-with-docs").write <<~SH
      #!/bin/sh
      # Sentinel. Use `ailloy` directly; this script just confirms the
      # docs extension was installed when ailloy-with-docs was set up.
      exec ailloy "$@"
    SH
    chmod 0755, bin/"ailloy-with-docs"
  end

  def post_install
    ailloy_bin = Formula["ailloy"].opt_bin/"ailloy"
    return unless ailloy_bin.executable?

    ohai "Installing the docs extension via #{ailloy_bin}..."
    system ailloy_bin, "extensions", "install", "docs"
  end

  test do
    # Verify the dependency is wired and that the docs extension
    # presents itself as installed.
    assert_match "ailloy", shell_output("#{bin}/ailloy-with-docs --version")
    assert_match "docs", shell_output("#{Formula["ailloy"].opt_bin}/ailloy ext list")
  end
end
