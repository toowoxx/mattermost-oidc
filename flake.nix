{
  description = "Mattermost OIDC SSO Provider";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go toolchain
            go_1_24
            gopls
            golangci-lint
            delve

            # Build tools
            gnumake
            gcc

            # Utilities
            curl
            jq
            git

            # For testing with containers
            docker
            docker-compose
          ];

          shellHook = ''
            echo "Mattermost OIDC development environment"
            echo "Go version: $(go version)"
            echo ""
            echo "Commands:"
            echo "  go test ./...           - Run tests"
            echo "  go build ./...          - Build module"
            echo "  golangci-lint run       - Run linter"
            echo ""

            # Set up Go environment
            export GOPATH="$HOME/go"
            export PATH="$GOPATH/bin:$PATH"

            # Bypass Go proxy for Mattermost modules (they don't publish server/v8)
            export GOPRIVATE="github.com/mattermost/*"

            # Hint about go.work for local development
            if [ -d "../mattermost/server" ] && [ ! -f "go.work" ]; then
              echo "Tip: Create go.work to use local mattermost checkout:"
              echo "  echo -e 'go 1.24.6\n\nuse (\n    .\n    ../mattermost/server\n)' > go.work"
              echo ""
            fi
          '';
        };

        packages.default = pkgs.buildGoModule {
          pname = "mattermost-oidc";
          version = "0.1.0";
          src = ./.;

          # This would need to be updated after running go mod tidy
          vendorHash = null;

          meta = with pkgs.lib; {
            description = "Generic OIDC SSO provider for Mattermost";
            homepage = "https://github.com/toowoxx/mattermost-oidc";
            license = licenses.agpl3Only;
            maintainers = [ ];
          };
        };
      }
    );
}
