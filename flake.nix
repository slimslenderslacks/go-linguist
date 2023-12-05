{
  description = "go-linguist";

  inputs = {
    platform-engineering = {
      url = "github:slimslenderslacks/nix-modules";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { nixpkgs, ... }@inputs:
    inputs.platform-engineering.golang-project
      {
        inherit nixpkgs;
        dir = ./.;
        name = "go-linguist";
        version = "0.1.0";
        package-overlay = pkgs: packages:
          packages;       
      };
}
