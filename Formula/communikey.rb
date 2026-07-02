# Formule Homebrew — communikey (build Go depuis les sources).
#
# Cette formule COMPILE communikey depuis le tarball source d'une release GitHub, plutôt
# que de télécharger un binaire pré-construit. Avantages : reproductible, auditable,
# pas de binaire opaque. Elle vise le tap personnel `aissablk1/homebrew-tap`, où elle
# cohabite avec la formule binaire générée automatiquement par GoReleaser
# (`.goreleaser.yaml`) ; garde celle qui convient à ton usage.
#
# AVANT PUBLICATION (à faire une fois la release v0.2.0 taguée) :
#   1. `git tag v0.2.0 && git push --tags`  (déclenche la release GitHub).
#   2. Récupérer le sha256 du tarball source :
#        curl -fsSL https://github.com/aissablk1/communikey/archive/refs/tags/v0.2.0.tar.gz \
#          | shasum -a 256
#   3. Remplacer le PLACEHOLDER `sha256` ci-dessous par cette valeur.
#   4. `brew install --build-from-source ./Formula/communikey.rb` pour vérifier en local,
#      puis `brew test communikey` et `brew audit --strict communikey`.
#
# Statut : alpha (v0.2.0-dev). Tant que la release n'est pas taguée, installe plutôt
# via `go install github.com/aissablk1/communikey@latest` ou `make install`.

class Communikey < Formula
  desc "Bus de messages inter-agents pour CLI (state-aware, chiffré E2E, zéro dépendance)"
  homepage "https://communikey.dev"
  url "https://github.com/aissablk1/communikey/archive/refs/tags/v0.2.0.tar.gz"
  sha256 "REMPLACER_PAR_LE_SHA256_DU_TARBALL_SOURCE_v0.2.0"
  license "Apache-2.0"
  head "https://github.com/aissablk1/communikey.git", branch: "main"

  # communikey ne dépend QUE de la toolchain Go pour compiler ; aucune dépendance runtime.
  depends_on "go" => :build

  def install
    # Build reproductible, sans CGO, symboles strippés. -X main.version est posé
    # pour le jour où le binaire exposera sa version (variable à câbler côté code).
    ldflags = "-s -w -X main.version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags), "."
    bin.install_symlink "communikey" => "comkey" # alias court
  end

  test do
    # `communikey help` doit répondre et mentionner le nom de l'outil. Pas de réseau,
    # pas d'état requis : test hermétique, valable en sandbox Homebrew.
    assert_match "communikey", shell_output("#{bin}/communikey help")
  end
end
