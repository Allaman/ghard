{
  description = "ghard - Go port of khard vCard address book manager";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = self.shortRev or "dev";
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "ghard";
          inherit version;
          src = ./.;

          # Run `nix build` once with a wrong hash to get the correct one
          # from the error output, then replace the placeholder below.
          vendorHash = "sha256-Bui4Z9+EEdWEe3hdtSJaGktZJBLkwZx+qoPSQ3bG0cs=";

          ldflags = [
            "-s"
            "-w"
            "-X github.com/allaman/ghard/internal/version.Version=${version}"
          ];

          subPackages = [ "cmd/ghard" ];

          meta = with pkgs.lib; {
            description = "Go port of khard, a command-line vCard address book manager";
            homepage = "https://github.com/allaman/ghard";
            license = licenses.mit;
            mainProgram = "ghard";
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            go-task
            golangci-lint
            govulncheck
          ];

          shellHook = ''
            echo "ghard dev shell — Go $(go version | awk '{print $3}')"
          '';
        };
      }
    );
}
