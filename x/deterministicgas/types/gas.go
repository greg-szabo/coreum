package types

import (
	"context"
	"fmt"

	"github.com/armon/go-metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/grpc"
	"github.com/gogo/protobuf/proto"
	googlegrpc "google.golang.org/grpc"

	"github.com/CoreumFoundation/coreum/x/deterministicgas"
)

const fuseGasMultiplier = 5

// NewDeterministicGasRouter returns wrapped router charging deterministic amount of gas for defined message types
func NewDeterministicGasRouter(baseRouter sdk.Router, deterministicGasRequirements deterministicgas.DeterministicGasRequirements) sdk.Router {
	return &deterministicGasRouter{
		baseRouter:                   baseRouter,
		deterministicGasRequirements: deterministicGasRequirements,
	}
}

type deterministicGasRouter struct {
	baseRouter                   sdk.Router
	deterministicGasRequirements deterministicgas.DeterministicGasRequirements
}

func (r *deterministicGasRouter) AddRoute(route sdk.Route) sdk.Router {
	r.baseRouter.AddRoute(sdk.NewRoute(route.Path(), r.handler(route.Handler())))
	return r
}

func (r *deterministicGasRouter) Route(ctx sdk.Context, path string) sdk.Handler {
	return r.baseRouter.Route(ctx, path)
}

func (r *deterministicGasRouter) handler(baseHandler sdk.Handler) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx, _, _ = ctxForDeterministicGas(ctx, msg, r.deterministicGasRequirements)
		return baseHandler(ctx, msg)
	}
}

// NewDeterministicMsgServer returns wrapped message server charging deterministic amount of gas for defined message types
func NewDeterministicMsgServer(baseServer grpc.Server, deterministicGasRequirements deterministicgas.DeterministicGasRequirements) grpc.Server {
	return &deterministicMsgServer{
		baseServer:                   baseServer,
		deterministicGasRequirements: deterministicGasRequirements,
	}
}

type deterministicMsgServer struct {
	baseServer                   grpc.Server
	deterministicGasRequirements deterministicgas.DeterministicGasRequirements
}

func (s *deterministicMsgServer) RegisterService(sd *googlegrpc.ServiceDesc, handler interface{}) {
	// To understand this implementation it is recommended to study the code in
	// https://github.com/cosmos/cosmos-sdk/blob/ff416ee63d32da5d520a8b2d16b00da762416146/baseapp/msg_service_router.go#L109

	// `sd` argument contains service description generated by protobuf. An example of simple description might be found here:
	// https://github.com/cosmos/cosmos-sdk/blob/ff416ee63d32da5d520a8b2d16b00da762416146/x/crisis/types/tx.pb.go#L208
	// Below, we replace original `Handler` of every message with our wrapper charging constant gas amount.
	// The signature of handler is
	//
	// func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor googlegrpc.UnaryServerInterceptor) (interface{}, error)
	//
	// Handler is called by GRPC framework passing an `interceptor`. The signature of `interceptor` is:
	//
	// func(ctx context.Context, req interface{}, info *UnaryServerInfo, handler UnaryHandler) (resp interface{}, err error)
	//
	// The last argument (`handler`) is the final function which must be called to handle the request.
	// We must call it passing message object as an argument (here called `req`).
	//
	// In original code, Cosmos SDK creates special interceptor which configures sdk context object: https://github.com/cosmos/cosmos-sdk/blob/ff416ee63d32da5d520a8b2d16b00da762416146/baseapp/msg_service_router.go#L111
	// We need to replace gas meter inside that object.
	// To do that we replace the original `Handler` with a function, which receives original `interceptor` created by Cosmos SDK.
	// But we don't call it directly. Instead, we pass our own interceptor function which calls the original one.
	// That interceptor wrapper receives the `handler` argument. But again, instead of calling it directly we pass our function
	// which is called by Cosmos SDK here: https://github.com/cosmos/cosmos-sdk/blob/ff416ee63d32da5d520a8b2d16b00da762416146/baseapp/msg_service_router.go#L113
	// giving us `ctx` containing cosmos context.
	//
	// Then we extract cosmos context from `ctx` replace gas meter, pack it into `ctx` again and hall final handler.

	for i, method := range sd.Methods {
		method := method
		sd.Methods[i].Handler = func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor googlegrpc.UnaryServerInterceptor) (interface{}, error) {
			return method.Handler(srv, ctx, dec, func(ctx context.Context, req interface{}, info *googlegrpc.UnaryServerInfo, handler googlegrpc.UnaryHandler) (resp interface{}, err error) {
				return interceptor(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
					sdkCtx := sdk.UnwrapSDKContext(ctx)
					msg := req.(sdk.Msg)
					newSDKCtx, gasBefore, isDeterministic := ctxForDeterministicGas(sdkCtx, msg, s.deterministicGasRequirements)
					//nolint:contextcheck // Naming sdk functions (sdk.WrapSDKContext) is not our responsibility
					res, err := handler(sdk.WrapSDKContext(newSDKCtx), req)
					// gas metrics are reported only if message type is deterministic, and was successful
					// CheckTx and ReCheckTx phases are ignored, since are only interested in the real execution
					// of the message at DeliverTx phase.
					if err == nil &&
						isDeterministic &&
						!newSDKCtx.IsCheckTx() &&
						!newSDKCtx.IsReCheckTx() {
						reportDeterministicGasMetric(sdkCtx, newSDKCtx, gasBefore, proto.MessageName(msg))
					}
					return res, err
				})
			})
		}
	}
	s.baseServer.RegisterService(sd, handler)
}

func ctxForDeterministicGas(ctx sdk.Context, msg sdk.Msg, deterministicGasRequirements deterministicgas.DeterministicGasRequirements) (sdk.Context, sdk.Gas, bool) {
	gasRequired, exists := deterministicGasRequirements.GasRequiredByMessage(msg)
	gasBefore := ctx.GasMeter().GasConsumed()
	if exists {
		// Fixed gas is consumed on original gas meter to require and report deterministic gas amount
		ctx.GasMeter().ConsumeGas(gasRequired, fmt.Sprintf("DeterministicGas (gas required: %d, message type: %T)", gasRequired, msg))

		// We pass much higher amount of gas to hanfdler to be sure that it succeeds.
		// We want to avoid passing infinite gas meter to always have a limit in case of mistake.
		ctx = ctx.WithGasMeter(sdk.NewGasMeter(fuseGasMultiplier * gasRequired))
	}
	return ctx, gasBefore, exists
}

func reportDeterministicGasMetric(oldCtx, newCtx sdk.Context, gasBefore sdk.Gas, msgName string) {
	deterministicGas := oldCtx.GasMeter().GasConsumed() - gasBefore
	if deterministicGas == 0 {
		return
	}

	nondeterministicGas := newCtx.GasMeter().GasConsumed()

	gasFactor := float32(nondeterministicGas) / float32(deterministicGas)
	metrics.AddSampleWithLabels([]string{"deterministic_gas_factor"}, gasFactor, []metrics.Label{
		{Name: "msg_name", Value: msgName},
	})
}
