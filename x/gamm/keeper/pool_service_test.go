package keeper_test

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/osmosis-labs/osmosis/v7/x/gamm/pool-models/balancer"
	balancertypes "github.com/osmosis-labs/osmosis/v7/x/gamm/pool-models/balancer"
	"github.com/osmosis-labs/osmosis/v7/x/gamm/types"
)

var (
	defaultSwapFee    = sdk.MustNewDecFromStr("0.025")
	defaultExitFee    = sdk.MustNewDecFromStr("0.025")
	defaultPoolParams = balancer.PoolParams{
		SwapFee: defaultSwapFee,
		ExitFee: defaultExitFee,
	}
	defaultFutureGovernor = ""

	// pool assets
	defaultFooAsset = balancertypes.PoolAsset{
		Weight: sdk.NewInt(100),
		Token:  sdk.NewCoin("foo", sdk.NewInt(10000)),
	}
	defaultBarAsset = balancertypes.PoolAsset{
		Weight: sdk.NewInt(100),
		Token:  sdk.NewCoin("bar", sdk.NewInt(10000)),
	}
	defaultPoolAssets           = []balancertypes.PoolAsset{defaultFooAsset, defaultBarAsset}
	defaultAcctFunds  sdk.Coins = sdk.NewCoins(
		sdk.NewCoin("uosmo", sdk.NewInt(10000000000)),
		sdk.NewCoin("foo", sdk.NewInt(10000000)),
		sdk.NewCoin("bar", sdk.NewInt(10000000)),
		sdk.NewCoin("baz", sdk.NewInt(10000000)),
	)
)

func (suite *KeeperTestSuite) TestCreateBalancerPool() {
	params := suite.app.GAMMKeeper.GetParams(suite.ctx)

	poolCreationFeeDecCoins := sdk.DecCoins{}
	for _, coin := range params.PoolCreationFee {
		poolCreationFeeDecCoins = poolCreationFeeDecCoins.Add(sdk.NewDecCoin(coin.Denom, coin.Amount))
	}

	func() {
		keeper := suite.app.GAMMKeeper

		// Try to create pool without balances.
		msg := balancer.NewMsgCreateBalancerPool(acc1, defaultPoolParams, defaultPoolAssets, defaultFutureGovernor)
		_, err := keeper.CreatePool(suite.ctx, msg)
		suite.Require().Error(err)
	}()

	// TODO: Refactor this to be more sensible.
	// The struct should contain a MsgCreateBalancerPool.
	// then the scaffolding should test pool creation, and check if it was a success or not.
	// (And should be moved to balancer package)
	// PoolCreationFee tests should get their own isolated test in this package.
	tests := []struct {
		fn func()
	}{{
		fn: func() {
			keeper := suite.app.GAMMKeeper
			prevFeePool := suite.app.DistrKeeper.GetFeePoolCommunityCoins(suite.ctx)
			prevAcc1Bal := suite.app.BankKeeper.GetAllBalances(suite.ctx, acc1)
			msg := balancer.NewMsgCreateBalancerPool(acc1, defaultPoolParams, defaultPoolAssets, defaultFutureGovernor)
			poolId, err := keeper.CreatePool(suite.ctx, msg)
			suite.Require().NoError(err)

			pool, err := keeper.GetPoolAndPoke(suite.ctx, poolId)
			suite.Require().NoError(err)
			suite.Require().Equal(types.InitPoolSharesSupply.String(), pool.GetTotalShares().String(),
				fmt.Sprintf("share token should be minted as %s initially", types.InitPoolSharesSupply.String()),
			)

			// check fee is correctly sent to community pool
			feePool := suite.app.DistrKeeper.GetFeePoolCommunityCoins(suite.ctx)
			suite.Require().Equal(feePool, prevFeePool.Add(poolCreationFeeDecCoins...))

			// check account's balance is correctly reduced
			acc1Bal := suite.app.BankKeeper.GetAllBalances(suite.ctx, acc1)
			suite.Require().Equal(acc1Bal.String(),
				prevAcc1Bal.Sub(params.PoolCreationFee).
					Sub(sdk.Coins{
						sdk.NewCoin("bar", sdk.NewInt(10000)),
						sdk.NewCoin("foo", sdk.NewInt(10000)),
					}).Add(sdk.NewCoin(types.GetPoolShareDenom(pool.GetId()), types.InitPoolSharesSupply)).String(),
			)

			liquidity := suite.app.GAMMKeeper.GetTotalLiquidity(suite.ctx)
			suite.Require().Equal("10000bar,10000foo", liquidity.String())
		},
	}, {
		fn: func() {
			keeper := suite.app.GAMMKeeper
			msg := balancer.NewMsgCreateBalancerPool(acc1, balancer.PoolParams{
				SwapFee: sdk.NewDecWithPrec(-1, 2),
				ExitFee: sdk.NewDecWithPrec(1, 2),
			}, defaultPoolAssets, defaultFutureGovernor)
			_, err := keeper.CreatePool(suite.ctx, msg)
			suite.Require().Error(err, "can't create a pool with negative swap fee")
		},
	}, {
		fn: func() {
			keeper := suite.app.GAMMKeeper
			msg := balancer.NewMsgCreateBalancerPool(acc1, balancer.PoolParams{
				SwapFee: sdk.NewDecWithPrec(1, 2),
				ExitFee: sdk.NewDecWithPrec(-1, 2),
			}, defaultPoolAssets, defaultFutureGovernor)
			_, err := keeper.CreatePool(suite.ctx, msg)
			suite.Require().Error(err, "can't create a pool with negative exit fee")
		},
	}, {
		fn: func() {
			keeper := suite.app.GAMMKeeper
			msg := balancer.NewMsgCreateBalancerPool(acc1, balancer.PoolParams{
				SwapFee: sdk.NewDecWithPrec(1, 2),
				ExitFee: sdk.NewDecWithPrec(1, 2),
			}, []balancertypes.PoolAsset{}, defaultFutureGovernor)
			_, err := keeper.CreatePool(suite.ctx, msg)
			suite.Require().Error(err, "can't create the pool with empty PoolAssets")
		},
	}, {
		fn: func() {
			keeper := suite.app.GAMMKeeper
			msg := balancer.NewMsgCreateBalancerPool(acc1, balancer.PoolParams{
				SwapFee: sdk.NewDecWithPrec(1, 2),
				ExitFee: sdk.NewDecWithPrec(1, 2),
			}, []balancertypes.PoolAsset{{
				Weight: sdk.NewInt(0),
				Token:  sdk.NewCoin("foo", sdk.NewInt(10000)),
			}, {
				Weight: sdk.NewInt(100),
				Token:  sdk.NewCoin("bar", sdk.NewInt(10000)),
			}}, defaultFutureGovernor)
			_, err := keeper.CreatePool(suite.ctx, msg)
			suite.Require().Error(err, "can't create the pool with 0 weighted PoolAsset")
		},
	}, {
		fn: func() {
			keeper := suite.app.GAMMKeeper
			msg := balancer.NewMsgCreateBalancerPool(acc1, balancer.PoolParams{
				SwapFee: sdk.NewDecWithPrec(1, 2),
				ExitFee: sdk.NewDecWithPrec(1, 2),
			}, []balancertypes.PoolAsset{{
				Weight: sdk.NewInt(-1),
				Token:  sdk.NewCoin("foo", sdk.NewInt(10000)),
			}, {
				Weight: sdk.NewInt(100),
				Token:  sdk.NewCoin("bar", sdk.NewInt(10000)),
			}}, defaultFutureGovernor)
			_, err := keeper.CreatePool(suite.ctx, msg)
			suite.Require().Error(err, "can't create the pool with negative weighted PoolAsset")
		},
	}, {
		fn: func() {
			keeper := suite.app.GAMMKeeper
			msg := balancer.NewMsgCreateBalancerPool(acc1, balancer.PoolParams{
				SwapFee: sdk.NewDecWithPrec(1, 2),
				ExitFee: sdk.NewDecWithPrec(1, 2),
			}, []balancertypes.PoolAsset{{
				Weight: sdk.NewInt(100),
				Token:  sdk.NewCoin("foo", sdk.NewInt(0)),
			}, {
				Weight: sdk.NewInt(100),
				Token:  sdk.NewCoin("bar", sdk.NewInt(10000)),
			}}, defaultFutureGovernor)
			_, err := keeper.CreatePool(suite.ctx, msg)
			suite.Require().Error(err, "can't create the pool with 0 balance PoolAsset")
		},
	}, {
		fn: func() {
			keeper := suite.app.GAMMKeeper
			msg := balancer.NewMsgCreateBalancerPool(acc1, balancer.PoolParams{
				SwapFee: sdk.NewDecWithPrec(1, 2),
				ExitFee: sdk.NewDecWithPrec(1, 2),
			}, []balancertypes.PoolAsset{{
				Weight: sdk.NewInt(100),
				Token: sdk.Coin{
					Denom:  "foo",
					Amount: sdk.NewInt(-1),
				},
			}, {
				Weight: sdk.NewInt(100),
				Token:  sdk.NewCoin("bar", sdk.NewInt(10000)),
			}}, defaultFutureGovernor)
			_, err := keeper.CreatePool(suite.ctx, msg)
			suite.Require().Error(err, "can't create the pool with negative balance PoolAsset")
		},
	}, {
		fn: func() {
			keeper := suite.app.GAMMKeeper
			msg := balancer.NewMsgCreateBalancerPool(acc1, balancer.PoolParams{
				SwapFee: sdk.NewDecWithPrec(1, 2),
				ExitFee: sdk.NewDecWithPrec(1, 2),
			}, []balancertypes.PoolAsset{{
				Weight: sdk.NewInt(100),
				Token:  sdk.NewCoin("foo", sdk.NewInt(10000)),
			}, {
				Weight: sdk.NewInt(100),
				Token:  sdk.NewCoin("foo", sdk.NewInt(10000)),
			}}, defaultFutureGovernor)
			_, err := keeper.CreatePool(suite.ctx, msg)
			suite.Require().Error(err, "can't create the pool with duplicated PoolAssets")
		},
	}, {
		fn: func() {
			keeper := suite.app.GAMMKeeper
			keeper.SetParams(suite.ctx, types.Params{
				PoolCreationFee: sdk.Coins{},
			})
			msg := balancer.NewMsgCreateBalancerPool(acc1, balancer.PoolParams{
				SwapFee: sdk.NewDecWithPrec(1, 2),
				ExitFee: sdk.NewDecWithPrec(1, 2),
			}, defaultPoolAssets, defaultFutureGovernor)
			_, err := keeper.CreatePool(suite.ctx, msg)
			suite.Require().NoError(err)
			pools, err := keeper.GetPoolsAndPoke(suite.ctx)
			suite.Require().Len(pools, 1)
			suite.Require().NoError(err)
		},
	}, {
		fn: func() {
			keeper := suite.app.GAMMKeeper
			keeper.SetParams(suite.ctx, types.Params{
				PoolCreationFee: nil,
			})
			msg := balancer.NewMsgCreateBalancerPool(acc1, balancer.PoolParams{
				SwapFee: sdk.NewDecWithPrec(1, 2),
				ExitFee: sdk.NewDecWithPrec(1, 2),
			}, defaultPoolAssets, defaultFutureGovernor)
			_, err := keeper.CreatePool(suite.ctx, msg)
			suite.Require().NoError(err)
			pools, err := keeper.GetPoolsAndPoke(suite.ctx)
			suite.Require().Len(pools, 1)
			suite.Require().NoError(err)
		},
	}}

	for _, test := range tests {
		suite.SetupTest()

		// Mint some assets to the accounts.
		for _, acc := range []sdk.AccAddress{acc1, acc2, acc3} {
			err := simapp.FundAccount(suite.app.BankKeeper, suite.ctx, acc, defaultAcctFunds)
			if err != nil {
				panic(err)
			}
		}

		test.fn()
	}
}

// TODO: Add more edge cases around TokenInMaxs not containing every token in pool.
func (suite *KeeperTestSuite) TestJoinPoolNoSwap() {
	tests := []struct {
		fn func(poolId uint64)
	}{
		{
			fn: func(poolId uint64) {
				keeper := suite.app.GAMMKeeper
				balancesBefore := suite.app.BankKeeper.GetAllBalances(suite.ctx, acc2)
				err := keeper.JoinPoolNoSwap(suite.ctx, acc2, poolId, types.OneShare.MulRaw(50), sdk.Coins{})
				suite.Require().NoError(err)
				suite.Require().Equal(types.OneShare.MulRaw(50).String(), suite.app.BankKeeper.GetBalance(suite.ctx, acc2, "gamm/pool/1").Amount.String())
				balancesAfter := suite.app.BankKeeper.GetAllBalances(suite.ctx, acc2)

				deltaBalances, _ := balancesBefore.SafeSub(balancesAfter)
				// The pool was created with the 10000foo, 10000bar, and the pool share was minted as 100000000gamm/pool/1.
				// Thus, to get the 50*OneShare gamm/pool/1, (10000foo, 10000bar) * (1 / 2) balances should be provided.
				suite.Require().Equal("5000", deltaBalances.AmountOf("foo").String())
				suite.Require().Equal("5000", deltaBalances.AmountOf("bar").String())

				liquidity := suite.app.GAMMKeeper.GetTotalLiquidity(suite.ctx)
				suite.Require().Equal("15000bar,15000foo", liquidity.String())
			},
		},
		{
			fn: func(poolId uint64) {
				keeper := suite.app.GAMMKeeper
				err := keeper.JoinPoolNoSwap(suite.ctx, acc2, poolId, sdk.NewInt(0), sdk.Coins{})
				suite.Require().Error(err, "can't join the pool with requesting 0 share amount")
			},
		},
		{
			fn: func(poolId uint64) {
				keeper := suite.app.GAMMKeeper
				err := keeper.JoinPoolNoSwap(suite.ctx, acc2, poolId, sdk.NewInt(-1), sdk.Coins{})
				suite.Require().Error(err, "can't join the pool with requesting negative share amount")
			},
		},
		{
			fn: func(poolId uint64) {
				keeper := suite.app.GAMMKeeper
				// Test the "tokenInMaxs"
				// In this case, to get the 50 * OneShare amount of share token, the foo, bar token are expected to be provided as 5000 amounts.
				err := keeper.JoinPoolNoSwap(suite.ctx, acc2, poolId, types.OneShare.MulRaw(50), sdk.Coins{
					sdk.NewCoin("bar", sdk.NewInt(4999)), sdk.NewCoin("foo", sdk.NewInt(4999)),
				})
				suite.Require().Error(err)
			},
		},
		{
			fn: func(poolId uint64) {
				keeper := suite.app.GAMMKeeper
				// Test the "tokenInMaxs"
				// In this case, to get the 50 * OneShare amount of share token, the foo, bar token are expected to be provided as 5000 amounts.
				err := keeper.JoinPoolNoSwap(suite.ctx, acc2, poolId, types.OneShare.MulRaw(50), sdk.Coins{
					sdk.NewCoin("bar", sdk.NewInt(5000)), sdk.NewCoin("foo", sdk.NewInt(5000)),
				})
				suite.Require().NoError(err)

				liquidity := suite.app.GAMMKeeper.GetTotalLiquidity(suite.ctx)
				suite.Require().Equal("15000bar,15000foo", liquidity.String())
			},
		},
	}

	for _, test := range tests {
		suite.SetupTest()

		// Mint some assets to the accounts.
		for _, acc := range []sdk.AccAddress{acc1, acc2, acc3} {
			err := simapp.FundAccount(suite.app.BankKeeper, suite.ctx, acc, defaultAcctFunds)
			if err != nil {
				panic(err)
			}
		}

		// Create the pool at first
		msg := balancer.NewMsgCreateBalancerPool(acc1, balancer.PoolParams{
			SwapFee: sdk.NewDecWithPrec(1, 2),
			ExitFee: sdk.NewDecWithPrec(1, 2),
		}, defaultPoolAssets, defaultFutureGovernor)
		poolId, err := suite.app.GAMMKeeper.CreatePool(suite.ctx, msg)
		suite.Require().NoError(err)

		test.fn(poolId)
	}
}

func (suite *KeeperTestSuite) TestExitPool() {
	tests := []struct {
		fn func(poolId uint64)
	}{
		{
			fn: func(poolId uint64) {
				keeper := suite.app.GAMMKeeper
				// Acc2 has no share token.
				_, err := keeper.ExitPool(suite.ctx, acc2, poolId, types.OneShare.MulRaw(50), sdk.Coins{})
				suite.Require().Error(err)
			},
		},
		{
			fn: func(poolId uint64) {
				keeper := suite.app.GAMMKeeper

				balancesBefore := suite.app.BankKeeper.GetAllBalances(suite.ctx, acc1)
				_, err := keeper.ExitPool(suite.ctx, acc1, poolId, types.InitPoolSharesSupply.QuoRaw(2), sdk.Coins{})
				suite.Require().NoError(err)
				// (100 - 50) * OneShare should remain.
				suite.Require().Equal(types.InitPoolSharesSupply.QuoRaw(2).String(), suite.app.BankKeeper.GetBalance(suite.ctx, acc1, "gamm/pool/1").Amount.String())
				balancesAfter := suite.app.BankKeeper.GetAllBalances(suite.ctx, acc1)

				deltaBalances, _ := balancesBefore.SafeSub(balancesAfter)
				// The pool was created with the 10000foo, 10000bar, and the pool share was minted as 100*OneShare gamm/pool/1.
				// Thus, to refund the 50*OneShare gamm/pool/1, (10000foo, 10000bar) * (1 / 2) balances should be refunded.
				suite.Require().Equal("-5000", deltaBalances.AmountOf("foo").String())
				suite.Require().Equal("-5000", deltaBalances.AmountOf("bar").String())
			},
		},
		{
			fn: func(poolId uint64) {
				keeper := suite.app.GAMMKeeper

				_, err := keeper.ExitPool(suite.ctx, acc1, poolId, sdk.NewInt(0), sdk.Coins{})
				suite.Require().Error(err, "can't join the pool with requesting 0 share amount")
			},
		},
		{
			fn: func(poolId uint64) {
				keeper := suite.app.GAMMKeeper

				_, err := keeper.ExitPool(suite.ctx, acc1, poolId, sdk.NewInt(-1), sdk.Coins{})
				suite.Require().Error(err, "can't join the pool with requesting negative share amount")
			},
		},
		{
			fn: func(poolId uint64) {
				keeper := suite.app.GAMMKeeper

				// Test the "tokenOutMins"
				// In this case, to refund the 50000000 amount of share token, the foo, bar token are expected to be refunded as 5000 amounts.
				_, err := keeper.ExitPool(suite.ctx, acc1, poolId, types.OneShare.MulRaw(50), sdk.Coins{
					sdk.NewCoin("foo", sdk.NewInt(5001)),
				})
				suite.Require().Error(err)
			},
		},
		{
			fn: func(poolId uint64) {
				keeper := suite.app.GAMMKeeper

				// Test the "tokenOutMins"
				// In this case, to refund the 50000000 amount of share token, the foo, bar token are expected to be refunded as 5000 amounts.
				_, err := keeper.ExitPool(suite.ctx, acc1, poolId, types.OneShare.MulRaw(50), sdk.Coins{
					sdk.NewCoin("foo", sdk.NewInt(5000)),
				})
				suite.Require().NoError(err)
			},
		},
	}

	for _, test := range tests {
		suite.SetupTest()

		// Mint some assets to the accounts.
		for _, acc := range []sdk.AccAddress{acc1, acc2, acc3} {
			err := simapp.FundAccount(suite.app.BankKeeper, suite.ctx, acc, defaultAcctFunds)
			if err != nil {
				panic(err)
			}

			// Create the pool at first
			msg := balancer.NewMsgCreateBalancerPool(acc1, balancer.PoolParams{
				SwapFee: sdk.NewDecWithPrec(1, 2),
				ExitFee: sdk.NewDec(0),
			}, defaultPoolAssets, defaultFutureGovernor)
			poolId, err := suite.app.GAMMKeeper.CreatePool(suite.ctx, msg)
			suite.Require().NoError(err)

			test.fn(poolId)
		}
	}
}

func (suite *KeeperTestSuite) TestActiveBalancerPool() {
	type testCase struct {
		blockTime  time.Time
		expectPass bool
	}

	testCases := []testCase{
		{time.Unix(1000, 0), true},
		{time.Unix(2000, 0), true},
	}

	for _, tc := range testCases {
		suite.SetupTest()

		// Mint some assets to the accounts.
		for _, acc := range []sdk.AccAddress{acc1, acc2, acc3} {
			err := simapp.FundAccount(suite.app.BankKeeper, suite.ctx, acc, defaultAcctFunds)
			suite.Require().NoError(err)

			// Create the pool at first
			poolId := suite.prepareBalancerPoolWithPoolParams(balancer.PoolParams{
				SwapFee: sdk.NewDec(0),
				ExitFee: sdk.NewDec(0),
			})
			suite.ctx = suite.ctx.WithBlockTime(tc.blockTime)

			// uneffected by start time
			err = suite.app.GAMMKeeper.JoinPoolNoSwap(suite.ctx, acc1, poolId, types.OneShare.MulRaw(50), sdk.Coins{})
			suite.Require().NoError(err)
			_, err = suite.app.GAMMKeeper.ExitPool(suite.ctx, acc1, poolId, types.InitPoolSharesSupply.QuoRaw(2), sdk.Coins{})
			suite.Require().NoError(err)

			foocoin := sdk.NewCoin("foo", sdk.NewInt(10))
			foocoins := sdk.Coins{foocoin}

			if tc.expectPass {
				_, err = suite.app.GAMMKeeper.JoinSwapExactAmountIn(suite.ctx, acc1, poolId, foocoins, sdk.ZeroInt())
				suite.Require().NoError(err)
				// _, err = suite.app.GAMMKeeper.JoinSwapShareAmountOut(suite.ctx, acc1, poolId, "foo", types.OneShare.MulRaw(10), sdk.NewInt(1000000000000000000))
				// suite.Require().NoError(err)
				_, err = suite.app.GAMMKeeper.ExitSwapShareAmountIn(suite.ctx, acc1, poolId, "foo", types.OneShare.MulRaw(10), sdk.ZeroInt())
				suite.Require().NoError(err)
				_, err = suite.app.GAMMKeeper.ExitSwapExactAmountOut(suite.ctx, acc1, poolId, foocoin, sdk.NewInt(1000000000000000000))
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				_, err = suite.app.GAMMKeeper.JoinSwapShareAmountOut(suite.ctx, acc1, poolId, "foo", types.OneShare.MulRaw(10), sdk.NewInt(1000000000000000000))
				suite.Require().Error(err)
				_, err = suite.app.GAMMKeeper.ExitSwapShareAmountIn(suite.ctx, acc1, poolId, "foo", types.OneShare.MulRaw(10), sdk.ZeroInt())
				suite.Require().Error(err)
				_, err = suite.app.GAMMKeeper.ExitSwapExactAmountOut(suite.ctx, acc1, poolId, foocoin, sdk.NewInt(1000000000000000000))
				suite.Require().Error(err)
			}
		}
	}
}

func (suite *KeeperTestSuite) TestJoinSwapExactAmountInConsistency() {
	testCases := []struct {
		name              string
		poolSwapFee       sdk.Dec
		poolExitFee       sdk.Dec
		tokensIn          sdk.Coins
		shareOutMinAmount sdk.Int
		expectedSharesOut sdk.Int
		tokenOutMinAmount sdk.Int
	}{
		{
			name:              "single coin with zero swap and exit fees",
			poolSwapFee:       sdk.ZeroDec(),
			poolExitFee:       sdk.ZeroDec(),
			tokensIn:          sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000000))),
			shareOutMinAmount: sdk.ZeroInt(),
			expectedSharesOut: sdk.NewInt(6265857020099440400),
			tokenOutMinAmount: sdk.ZeroInt(),
		},
		// TODO: Uncomment or remove this following test case once the referenced
		// issue is resolved.
		//
		// Ref: https://github.com/osmosis-labs/osmosis/issues/1196
		// {
		// 	name:              "single coin with positive swap fee and zero exit fee",
		// 	poolSwapFee:       sdk.NewDecWithPrec(1, 2),
		// 	poolExitFee:       sdk.ZeroDec(),
		// 	tokensIn:          sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000000))),
		// 	shareOutMinAmount: sdk.ZeroInt(),
		// 	expectedSharesOut: sdk.NewInt(6226484702880621000),
		// 	tokenOutMinAmount: sdk.ZeroInt(),
		// },
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := suite.ctx

			poolID := suite.prepareCustomBalancerPool(
				sdk.NewCoins(
					sdk.NewCoin("uosmo", sdk.NewInt(10000000000)),
					sdk.NewCoin("foo", sdk.NewInt(10000000)),
					sdk.NewCoin("bar", sdk.NewInt(10000000)),
					sdk.NewCoin("baz", sdk.NewInt(10000000)),
				),
				[]balancertypes.PoolAsset{
					{
						Weight: sdk.NewInt(100),
						Token:  sdk.NewCoin("foo", sdk.NewInt(5000000)),
					},
					{
						Weight: sdk.NewInt(200),
						Token:  sdk.NewCoin("bar", sdk.NewInt(5000000)),
					},
				},
				balancer.PoolParams{
					SwapFee: tc.poolSwapFee,
					ExitFee: tc.poolExitFee,
				},
			)

			shares, err := suite.app.GAMMKeeper.JoinSwapExactAmountIn(ctx, acc1, poolID, tc.tokensIn, tc.shareOutMinAmount)
			suite.Require().NoError(err)
			suite.Require().Equal(tc.expectedSharesOut, shares)

			tokenOutAmt, err := suite.app.GAMMKeeper.ExitSwapShareAmountIn(
				ctx,
				acc1,
				poolID,
				tc.tokensIn[0].Denom,
				shares,
				tc.tokenOutMinAmount,
			)
			suite.Require().NoError(err)

			// require swapTokenOutAmt <= (tokenInAmt * (1 - tc.poolSwapFee))
			oneMinusSwapFee := sdk.OneDec().Sub(tc.poolSwapFee)
			swapFeeAdjustedAmount := oneMinusSwapFee.MulInt(tc.tokensIn[0].Amount).RoundInt()
			suite.Require().True(tokenOutAmt.LTE(swapFeeAdjustedAmount))

			// require swapTokenOutAmt + 10 > input
			suite.Require().True(
				swapFeeAdjustedAmount.Sub(tokenOutAmt).LTE(sdk.NewInt(10)),
				"expected out amount %s, actual out amount %s",
				swapFeeAdjustedAmount, tokenOutAmt,
			)
		})
	}
}
