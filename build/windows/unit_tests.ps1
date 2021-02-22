echo "--- Running tests"

go mod download
go test ./...
if (-not $?)
{
    echo "Failed running tests"
    exit -1
}
