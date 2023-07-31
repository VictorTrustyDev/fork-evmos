// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package keeper

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
)

func (k Keeper) CalculateBaseFee(ctx sdk.Context) *big.Int {
	return k.CalculateBaseFee2(ctx, false)
}

func PrintDebugCalculateBaseFee(debug bool, msg string) {
	if debug {
		fmt.Println("CalculateBaseFee", msg)
	}
}

// CalculateBaseFee calculates the base fee for the current block. This is only calculated once per
// block during BeginBlock. If the NoBaseFee parameter is enabled or below activation height, this function returns nil.
// NOTE: This code is inspired from the go-ethereum EIP1559 implementation and adapted to Cosmos SDK-based
// chains. For the canonical code refer to: https://github.com/ethereum/go-ethereum/blob/master/consensus/misc/eip1559.go
func (k Keeper) CalculateBaseFee2(ctx sdk.Context, debug bool) *big.Int {
	PrintDebugCalculateBaseFee(debug, "1")

	params := k.GetParams(ctx)

	PrintDebugCalculateBaseFee(debug, "2")

	// Ignore the calculation if not enabled
	if !params.IsBaseFeeEnabled(ctx.BlockHeight()) {
		PrintDebugCalculateBaseFee(debug, "2.1")
		return nil
	}

	PrintDebugCalculateBaseFee(debug, "3")

	consParams := ctx.ConsensusParams()

	PrintDebugCalculateBaseFee(debug, "4")

	// If the current block is the first EIP-1559 block, return the base fee
	// defined in the parameters (DefaultBaseFee if it hasn't been changed by
	// governance).
	if ctx.BlockHeight() == params.EnableHeight {
		PrintDebugCalculateBaseFee(debug, "5")
		return params.BaseFee.BigInt()
	}

	PrintDebugCalculateBaseFee(debug, "6")

	// get the block gas used and the base fee values for the parent block.
	// NOTE: this is not the parent's base fee but the current block's base fee,
	// as it is retrieved from the transient store, which is committed to the
	// persistent KVStore after EndBlock (ABCI Commit).
	parentBaseFee := params.BaseFee.BigInt()

	PrintDebugCalculateBaseFee(debug, "7")

	if parentBaseFee == nil {
		return nil
	}

	PrintDebugCalculateBaseFee(debug, "8")

	parentGasUsed := k.GetBlockGasWanted(ctx)

	PrintDebugCalculateBaseFee(debug, "9")

	gasLimit := new(big.Int).SetUint64(math.MaxUint64)

	PrintDebugCalculateBaseFee(debug, "10")

	// NOTE: a MaxGas equal to -1 means that block gas is unlimited
	if consParams != nil && consParams.Block.MaxGas > -1 {
		PrintDebugCalculateBaseFee(debug, "11")
		gasLimit = big.NewInt(consParams.Block.MaxGas)
		PrintDebugCalculateBaseFee(debug, "12")
	}

	PrintDebugCalculateBaseFee(debug, "13")

	// CONTRACT: ElasticityMultiplier cannot be 0 as it's checked in the params
	// validation
	parentGasTargetBig := new(big.Int).Div(gasLimit, new(big.Int).SetUint64(uint64(params.ElasticityMultiplier)))

	PrintDebugCalculateBaseFee(debug, "14")
	if !parentGasTargetBig.IsUint64() {
		PrintDebugCalculateBaseFee(debug, "15")
		return nil
	}

	PrintDebugCalculateBaseFee(debug, "16")

	parentGasTarget := parentGasTargetBig.Uint64()

	PrintDebugCalculateBaseFee(debug, "17")

	baseFeeChangeDenominator := new(big.Int).SetUint64(uint64(params.BaseFeeChangeDenominator))

	PrintDebugCalculateBaseFee(debug, "18")

	// If the parent gasUsed is the same as the target, the baseFee remains
	// unchanged.
	if parentGasUsed == parentGasTarget {
		PrintDebugCalculateBaseFee(debug, "19")
		return new(big.Int).Set(parentBaseFee)
	}

	PrintDebugCalculateBaseFee(debug, "20")

	if parentGasUsed > parentGasTarget {
		PrintDebugCalculateBaseFee(debug, "21")
		// If the parent block used more gas than its target, the baseFee should
		// increase.
		gasUsedDelta := new(big.Int).SetUint64(parentGasUsed - parentGasTarget)

		PrintDebugCalculateBaseFee(debug, "22")

		x := new(big.Int).Mul(parentBaseFee, gasUsedDelta)

		PrintDebugCalculateBaseFee(debug, "23")

		y := x.Div(x, parentGasTargetBig)

		PrintDebugCalculateBaseFee(debug, "24")

		baseFeeDelta := math.BigMax(
			x.Div(y, baseFeeChangeDenominator),
			common.Big1,
		)

		PrintDebugCalculateBaseFee(debug, "25")

		return x.Add(parentBaseFee, baseFeeDelta)
	}

	PrintDebugCalculateBaseFee(debug, "26")

	// Otherwise if the parent block used less gas than its target, the baseFee
	// should decrease.
	gasUsedDelta := new(big.Int).SetUint64(parentGasTarget - parentGasUsed)
	PrintDebugCalculateBaseFee(debug, "27")
	x := new(big.Int).Mul(parentBaseFee, gasUsedDelta)
	PrintDebugCalculateBaseFee(debug, "28")
	y := x.Div(x, parentGasTargetBig)
	PrintDebugCalculateBaseFee(debug, "29")
	baseFeeDelta := x.Div(y, baseFeeChangeDenominator)
	PrintDebugCalculateBaseFee(debug, "30")

	// Set global min gas price as lower bound of the base fee, transactions below
	// the min gas price don't even reach the mempool.
	minGasPrice := params.MinGasPrice.TruncateInt().BigInt()
	PrintDebugCalculateBaseFee(debug, "31")
	return math.BigMax(x.Sub(parentBaseFee, baseFeeDelta), minGasPrice)
}
