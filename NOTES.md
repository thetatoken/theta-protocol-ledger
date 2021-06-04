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


// Set up for Off-Chain testing

thetacli query account --address=2E833968E5bB786Ae419c4d13189fB081Cc43bab


thetacli tx reserve --async --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --fund=10 --collateral=11 --duration=31 --resource_ids=rid1000001 --seq=1

thetacli tx send --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=70f587259738cB626A1720Af7038B8DcDb6a42a0 --theta=0 --tfuel=1 --seq=2

thetacli query account --address=2E833968E5bB786Ae419c4d13189fB081Cc43bab

thetacli query account --address=70f587259738cB626A1720Af7038B8DcDb6a42a0

// Alice : I want rid1000001 and I'm willing to pay up to 10 TFuel for it.
thetacli query account --address=2E833968E5bB786Ae419c4d13189fB081Cc43bab

thetacli tx reserve --async --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --fund=10 --collateral=11 --duration=101 --resource_ids=rid1000001 --seq=12

thetacli query account --address=2E833968E5bB786Ae419c4d13189fB081Cc43bab

"hash": "0x4bb258b6784ec0a755e5ab7dfb50403a462866ebac3d27c96463b2a1becd65ca"

thetacli query tx --hash=0x3d38c3851ae49072400b9f4c63fea1511b600552ff0eed4f008eb1d5cec5013a

// Alice -> Bob
"end_block_height": "1533"
thetacli query tx --hash=0x29745a458dc5e1f39a511b889f04396d1add0a16ad279816b46dfa496b1fa228
// Alice <- Bob
--resource_id=rid1000001

// Alice -> Bob
thetacli tx service_payment --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=70f587259738cB626A1720Af7038B8DcDb6a42a0 --payment_seq=1 --reserve_seq=12 --resource_id=rid1000001 --tfuel=2

--resource_id=rid1000001

thetacli tx service_payment --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=70f587259738cB626A1720Af7038B8DcDb6a42a0 --payment_seq=2 --reserve_seq=12 --resource_id=rid1000001 --tfuel=2

// Alice -> Carol
thetacli query tx --hash=0x29745a458dc5e1f39a511b889f04396d1add0a16ad279816b46dfa496b1fa228
// Alice <- Carol
--resource_id=rid1000001

// Alice -> Carol
thetacli tx service_payment --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=cd56123D0c5D6C1Ba4D39367b88cba61D93F5405 --payment_seq=3 --reserve_seq=12 --resource_id=rid1000001 --tfuel=2

--resource_id=rid1000001

thetacli tx service_payment --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=cd56123D0c5D6C1Ba4D39367b88cba61D93F5405 --payment_seq=4 --reserve_seq=13 --resource_id=rid1000001 --tfuel=4


// Bob -> Payout
thetacli query tx --hash=0x3d38c3851ae49072400b9f4c63fea1511b600552ff0eed4f008eb1d5cec5013a

thetacli tx service_payment --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=70f587259738cB626A1720Af7038B8DcDb6a42a0 --payment_seq=1 --reserve_seq=12 --resource_id=rid1000001 --tfuel=2 --on_chain --src_sig=0x12bd5090066cb508b50c437faba261afca2ed1c985812a2f7e4d2a6321d9128c33a5456c4105ceafb3da7897cfb97b50e6ff70e1ba7aaf0ba562890246ec728801

thetacli query account --address=70f587259738cB626A1720Af7038B8DcDb6a42a0

// Carol -> Payout
thetacli query tx --hash=0x3d38c3851ae49072400b9f4c63fea1511b600552ff0eed4f008eb1d5cec5013a

thetacli tx service_payment --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=cd56123D0c5D6C1Ba4D39367b88cba61D93F5405 --payment_seq=4 --reserve_seq=12 --resource_id=rid1000001 --tfuel=4 --on_chain --src_sig=0x1114ce5922a7e940542468fc2b6cd22f779408224310d63eb6215171e8618daf53432533b207704dcd2d21235a8e40e84819df369154b01f7a47357e496250e801

thetacli query account --address=0xcd56123D0c5D6C1Ba4D39367b88cba61D93F5405
