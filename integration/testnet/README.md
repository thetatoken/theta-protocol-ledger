### To run a node ###
```
cd $UKULELE_HOME
cp -r ./integration/testnet ../testnet
ukulele start --config=../testnet/node2 |& tee ../node2.log
```
