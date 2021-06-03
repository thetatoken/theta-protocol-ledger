```
$ thetacli query account --address=2E833968E5bB786Ae419c4d13189fB081Cc43bab
Using config file: /Users/i830671/.thetacli/config.yaml
{
    "code": "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
    "coins": {
        "tfuelwei": "312499798999999000000000000",
        "thetawei": "12500000000000000000000000"
    },
    "last_updated_block_height": "0",
    "reserved_funds": [
        {
            "collateral": {
                "tfuelwei": "101000000000000000000",
                "thetawei": "0"
            },
            "end_block_height": "1428",
            "initial_fund": {
                "tfuelwei": "100000000000000000000",
                "thetawei": "0"
            },
            "reserve_sequence": "1",
            "resource_ids": [
                "rid1000001"
            ],
            "transfer_records": [],
            "used_fund": {
                "tfuelwei": "0",
                "thetawei": "0"
            }
        }
    ],
    "root": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "sequence": "1"
}
```
// From unit tests 
// theta-protocol-ledger > ledger > execution > regular_ty_execution_test.go : Line 723


func TestServicePaymentTxNormalExecutionAndSlash(t *testing.T) {
	assert := assert.New(t)

    // This sets up 3 accounts with alice reserving funds on hers
	et, resourceID, alice, bob, carol, _, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	et.state().Commit()

	txFee := getMinimumTxFee()

// Query Alices' account make sure she has a single reserve fund
	retrievedAliceAcc0 := et.state().Delivered().GetAccount(alice.Address)
	assert.Equal(1, len(retrievedAliceAcc0.ReservedFunds))
	assert.Equal([]string{resourceID}, retrievedAliceAcc0.ReservedFunds[0].ResourceIDs)
	assert.Equal(types.Coins{TFuelWei: big.NewInt(1001 * txFee), ThetaWei: big.NewInt(0)}, retrievedAliceAcc0.ReservedFunds[0].Collateral)
	assert.Equal(uint64(1), retrievedAliceAcc0.ReservedFunds[0].ReserveSequence)

	// Simulate micropayment #1 between Alice and Bob
	payAmount1 := int64(80 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1
	_ = createServicePaymentTx(et.chainID, &alice, &bob, 10*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	_ = createServicePaymentTx(et.chainID, &alice, &bob, 50*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)

	servicePaymentTx1 := createServicePaymentTx(et.chainID, &alice, &bob, payAmount1, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)

	res := et.executor.getTxExecutor(servicePaymentTx1).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx1)

	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(servicePaymentTx1).process(et.chainID, et.state().Delivered(), servicePaymentTx1)

	assert.True(res.IsOK(), res.Message)
	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))

	et.state().Commit()

	retrievedAliceAcc1 := et.state().Delivered().GetAccount(alice.Address)

	assert.Equal(types.Coins{TFuelWei: big.NewInt(payAmount1), ThetaWei: big.NewInt(0)}, retrievedAliceAcc1.ReservedFunds[0].UsedFund)
	retrievedBobAcc1 := et.state().Delivered().GetAccount(bob.Address)
	assert.Equal(bobInitBalance.Plus(types.Coins{TFuelWei: big.NewInt(payAmount1 - txFee), ThetaWei: big.NewInt(0)}), retrievedBobAcc1.Balance) // payAmount1 - txFee: need to account for tx fee

	// Simulate micropayment #2 between Alice and Bob
	payAmount2 := int64(50 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq = 1, 2, 2, 1
	_ = createServicePaymentTx(et.chainID, &alice, &bob, 30*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx2 := createServicePaymentTx(et.chainID, &alice, &bob, payAmount2, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx2).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx2)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(servicePaymentTx2).process(et.chainID, et.state().Delivered(), servicePaymentTx2)
	assert.True(res.IsOK(), res.Message)
	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))

	et.state().Commit()

	retrievedAliceAcc2 := et.state().Delivered().GetAccount(alice.Address)
	assert.Equal(types.Coins{TFuelWei: big.NewInt(payAmount1 + payAmount2), ThetaWei: big.NewInt(0)}, retrievedAliceAcc2.ReservedFunds[0].UsedFund)
	retrievedBobAcc2 := et.state().Delivered().GetAccount(bob.Address)
	assert.Equal(bobInitBalance.Plus(types.Coins{TFuelWei: big.NewInt(payAmount1 + payAmount2 - 2*txFee)}), retrievedBobAcc2.Balance) // payAmount1 + payAmount2 - 2*txFee: need to account for tx fee

	// Simulate micropayment #3 between Alice and Carol
	payAmount3 := int64(120 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq = 1, 1, 3, 1
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 30*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx3 := createServicePaymentTx(et.chainID, &alice, &carol, payAmount3, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx3).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx3)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(servicePaymentTx3).process(et.chainID, et.state().Delivered(), servicePaymentTx3)
	assert.True(res.IsOK(), res.Message)
	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))

	et.state().Commit()

	retrievedAliceAcc3 := et.state().Delivered().GetAccount(alice.Address)
	assert.Equal(types.Coins{TFuelWei: big.NewInt(payAmount1 + payAmount2 + payAmount3), ThetaWei: big.NewInt(0)}, retrievedAliceAcc3.ReservedFunds[0].UsedFund)
	retrievedCarolAcc3 := et.state().Delivered().GetAccount(carol.Address)
	assert.Equal(carolInitBalance.Plus(types.Coins{TFuelWei: big.NewInt(payAmount3 - txFee)}), retrievedCarolAcc3.Balance) // payAmount3 - txFee: need to account for tx fee

	// Simulate micropayment #4 between Alice and Carol. This is an overspend, alice should get slashed.
	payAmount4 := int64(2000 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq = 1, 2, 4, 1
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 70000*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx4 := createServicePaymentTx(et.chainID, &alice, &carol, payAmount4, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx4).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx4)
	assert.True(res.IsOK(), res.Message) // the following process() call will create an SlashIntent

	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
	_, res = et.executor.getTxExecutor(servicePaymentTx4).process(et.chainID, et.state().Delivered(), servicePaymentTx4)
	assert.True(res.IsOK(), res.Message)
	//assert.Equal(1, len(et.state().Delivered().GetSlashIntents()))



thetacli tx reserve --async --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --fund=10 --collateral=11 --duration=1001 --resource_ids=rid1000001 --seq=1

"hash": "0x711e0001d454a556f6f1408f23f263fd2023c4c0e8eb54f5add1aaac137c8370",

thetacli query tx --hash=711e0001d454a556f6f1408f23f263fd2023c4c0e8eb54f5add1aaac137c8370

thetacli tx service_payment --on_chain --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=70f587259738cB626A1720Af7038B8DcDb6a42a0 --payment_seq=1 --reserve_seq=2 --resource_id=rid1000001

thetacli tx service_payment --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=70f587259738cB626A1720Af7038B8DcDb6a42a0 --payment_seq=1 --reserve_seq=2 --resource_id=rid1000001