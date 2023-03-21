

build:
	go build -ldflags \
        "-X github.com/cosmos/cosmos-sdk/version.Name=coreum \
        -X github.com/cosmos/cosmos-sdk/version.AppName=cored \
        -X github.com/cosmos/cosmos-sdk/version.Version=v1.0.0 \
        -X github.com/cosmos/cosmos-sdk/version.Commit=$(git rev-parse HEAD) -w -s" \
        -trimpath ./cmd/cored