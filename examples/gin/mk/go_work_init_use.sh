GO_MINOR_VER=$(go version | grep -E -o "1.[0-9]{2}" | cut -d. -f2)

if [ $GO_MINOR_VER -lt 18 ]; then
	echo "Your go version needs to be 1.18+ to use workspaces."
	echo "GO version is $(go version)"
  exit 1
fi

if [ ! -e go.work ]; then
  echo "go.work file not found, Running: go work init"
  go work init
fi

GO_WORK_USE_ENTRY=$(grep -o "use ../.." go.work)

if [ -z "$GO_WORK_USE_ENTRY" ]; then
  echo "use ../.. go.work entry not found, Running: go work use ../.."
  go work use ../..
fi

cat <<-EOF
You are using go workspaces! Now go commands like build and run on main.go file
will use your local version of gobrake, this is useful if you are doing local
gobrake development.
Want to turn workspaces off for a single go command you can prefix it like:

  GOWORK=off go run main.go
  GOWORK=off go build main.go

To remove the workspace altogether run:

  make remove_workspace
EOF
