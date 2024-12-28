{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  name = "go-project-env";

  buildInputs = [
    pkgs.go
    pkgs.git
  ];

  shellHook = ''
    export GOPATH=$PWD/go
    export PATH=$GOPATH/bin:$PATH
    mkdir -p $GOPATH
    echo "Go environment set up. GOPATH is set to $GOPATH"
  '';
}
