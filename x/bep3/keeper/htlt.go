package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/x/bep3/types"
)

// AddHTLT adds an htlt
func (k Keeper) AddHTLT(ctx sdk.Context, from sdk.AccAddress, to sdk.AccAddress, recipientOtherChain,
	senderOtherChain string, randomNumberHash string, timestamp int64, coins sdk.Coins,
	expectedIncome string, heightSpan int64, crossChain bool) (string, sdk.Error) {

	// _, found := k.GetHTLT(ctx, expectedSwapID)
	// if found {
	// 	return "", types.ErrHTLTAlreadyExists(k.codespace, expectedSwapID)
	// }

	err := k.ValidateAsset(ctx, coins)
	if err != nil {
		return "", err
	}

	htlt := types.NewHTLT(from, to, recipientOtherChain, senderOtherChain,
		randomNumberHash, timestamp, coins, expectedIncome, heightSpan,
		crossChain)

	// Send coins from sender to the bep3 module
	err = k.supplyKeeper.SendCoinsFromAccountToModule(ctx, from, types.ModuleName, coins)
	if err != nil {
		return "", sdk.ErrInternal(err.Error())
	}

	swapID, sdkErr := k.StoreNewHTLT(ctx, htlt)
	if sdkErr != nil {
		return "", sdk.ErrInternal(sdkErr.Error())
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCreateHtlt,
			sdk.NewAttribute(types.AttributeKeyHtltSwapID, fmt.Sprintf("%s", swapID)),
			sdk.NewAttribute(types.AttributeKeyFrom, fmt.Sprintf("%s", htlt.From)),
			sdk.NewAttribute(types.AttributeKeyTo, fmt.Sprintf("%s", htlt.To)),
			sdk.NewAttribute(types.AttributeKeyCoinDenom, fmt.Sprintf("%s", htlt.Amount[0].Denom)),
			sdk.NewAttribute(types.AttributeKeyCoinAmount, fmt.Sprintf("%d", htlt.Amount[0].Amount.Int64())),
		),
	)

	return swapID, nil
}

// DepositHTLT deposits funds in an existing HTLT
func (k Keeper) DepositHTLT(ctx sdk.Context, from sdk.AccAddress, swapID string, coins sdk.Coins) sdk.Error {
	decodedSwapID, err := types.HexEncodedStringToBytes(swapID)
	if err != nil {
		return sdk.ErrInternal(err.Error())
	}

	htlt, found := k.GetHTLT(ctx, decodedSwapID)
	if !found {
		return types.ErrHTLTNotFound(k.codespace, swapID)
	}

	htltCoin := htlt.Amount[0]
	coin := coins[0]

	// Validate new deposit
	if htltCoin.Denom != coin.Denom {
		return types.ErrInvalidCoinDenom(k.codespace, htltCoin.Denom, coin.Denom)
	}
	if coin.Amount.IsZero() {
		return types.ErrAmountTooSmall(k.codespace, coin)
	}
	// TODO: Param validation
	// if AssetSupply + coin.Amount > AssetSupplyLimit {
	// 	return a, types.ErrAmountTooLarge(k.codespace, coin.Amount)
	// }

	// Send coins from depositor to the bep3 module
	err = k.supplyKeeper.SendCoinsFromAccountToModule(ctx, from, types.ModuleName, coins)
	if err != nil {
		return sdk.ErrInternal(err.Error())
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDepositHtlt,
			sdk.NewAttribute(types.AttributeKeyHtltSwapID, fmt.Sprintf("%s", swapID)),
			sdk.NewAttribute(types.AttributeKeyCoinDenom, fmt.Sprintf("%s", coin.Denom)),
			sdk.NewAttribute(types.AttributeKeyCoinAmount, fmt.Sprintf("%d", coin.Amount.Int64())),
		),
	)

	// Update HTLT state
	htlt.Amount = htlt.Amount.Add(coins)
	currExpectedIncome, _ := sdk.ParseCoins(htlt.ExpectedIncome)
	htlt.ExpectedIncome = currExpectedIncome.Add(coins).String()

	k.SetHTLT(ctx, htlt, decodedSwapID)

	return nil
}

// GetAllHtlts returns all HTLTs from the store
func (k Keeper) GetAllHtlts(ctx sdk.Context) (htlts types.HTLTs) {
	k.IterateHTLTs(ctx, func(htlt types.HTLT) bool {
		htlts = append(htlts, htlt)
		return false
	})
	return
}

// ValidateAsset validates that a amount is valid for HTLTs
func (k Keeper) ValidateAsset(ctx sdk.Context, assets sdk.Coins) sdk.Error {
	if len(assets) != 1 {
		return sdk.ErrInternal("HTLTs currently only support 1 asset at a time")
	}
	// TODO: param validation
	// _, found := k.GetAsset(ctx, assets[0].Denom)
	// if !found {
	// 	return types.ErrAssetNotSupported(k.codespace, assets[0].Denom)
	// }
	return nil
}