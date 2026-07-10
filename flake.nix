{
  description = "development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {inherit system;};
    in {
      devShells.default = pkgs.mkShell {
        buildInputs = with pkgs; [
          python312Packages.python # Use the base python
          python312Packages.pip
          python312Packages.virtualenv
          python312Packages.tkinter

          stdenv.cc.cc.lib
          zlib
        ];

        shellHook = ''
          # 0. Make Nix store libraries discoverable at runtime by pip-installed packages
          export LD_LIBRARY_PATH="${pkgs.lib.makeLibraryPath [pkgs.zlib pkgs.stdenv.cc.cc.lib]}:$LD_LIBRARY_PATH"

          # 1. Create venv if it doesn't exist
          if [ ! -d ".venv" ]; then
            echo "Creating new virtual environment..."
            python -m venv .venv
          fi

          # 2. Source the activation script
          # Note: shellHook runs in bash, so we source the bash version
          # even if your main shell is fish.
          source .venv/bin/activate

          # 3. Upgrade pip in the LOCAL venv first to avoid Nix store issues
          # pip install --upgrade pip

          # 4. Install requirements
          if [ -f "requirements.txt" ]; then
            pip install -r requirements.txt
          fi

          echo "🚀 Python ready!"
        '';
      };
    });
}

