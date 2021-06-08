#!/bin/bash
#set -x #echo on
do_run=1 # Execute(evaluate) the commands
#do_run=0 # Don't evaluate any commands
do_echo=1 # Echo the commands
#do_echo=0 # Don't echo any commands

Alice=2E833968E5bB786Ae419c4d13189fB081Cc43bab
Bob=70f587259738cB626A1720Af7038B8DcDb6a42a0
Carol=cd56123D0c5D6C1Ba4D39367b88cba61D93F5405
rid=rid1000001
rpduration=51
rfund=100
rcoll=101
tfuel=50
bobsigs=1
carolsigs=1

startseq=0

#####################################################################
# Default scale used by float functions.

float_scale=18
z18=1000000000000000000

#####################################################################
# Evaluate a floating point number expression.

function un_wei()
{
    local stat=0
    local result=0.0
    if [[ $# -gt 0 ]]; then
        result=$(echo "scale=$float_scale; $*/$z18" | bc -q 2>/dev/null)
        stat=$?
        if [[ $stat -eq 0  &&  -z "$result" ]]; then stat=1; fi
    fi
    echo $result
    return $stat
}

#cbh=$(thetacli query status | tail -n +2 | jq .current_height)
echo ""
echo "Getting Current Block Height."
#cmd='cat outbin/combined.json | jq '"'"'.contracts | ."'$solfile':'$contractName'" | .abi '"'"' > outbin/abi.json'
cmd='cbh=$(thetacli query status | jq .current_height)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo "current_block_height: "$cbh

#aib=$(thetacli query account --address=$Alice | tail -n +2 | jq .coins.tfuelwei | tr -d '"'"'"'"'"')
echo ""
echo "Getting Alice's account balance."
cmd='aib=$(thetacli query account --address='$Alice' | jq .coins.tfuelwei | tr -d '"'"'"'"'"')'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo "Alice initial balance: "$(un_wei $aib)

echo ""
echo "Getting Alice's next sequence."
cmd='aseq=$(thetacli query account --address='$Alice' | jq .sequence | tr -d '"'"'"'"'"')'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
ans=$(( $aseq + 1 ))
echo "Alice next sequence: "$ans

echo ""
echo "Check for existing reserve fund."
cmd='arf=$(thetacli query account --address='$Alice' | jq .reserved_funds)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi

#echo "arf:"${#arf}

if [ ${#arf} == 2 ]; then
    echo "No Reserve Fund"
    echo "Alice begin create reserve."
    cmd='thetacli tx reserve --chain="privatenet" --async --from='$Alice' --fund='$rfund' --collateral='$rcoll' --duration='$rpduration' --resource_ids='$rid' --seq='$ans' --pw=qwertyuiop'
    if [ $do_echo -eq 1 ]; then echo $cmd; fi
    if [ $do_run -eq 1 ]; then eval $cmd; fi
    echo "Alice end create reserve.  Wait 10 seconds and rereun."
    echo "Sleeping..."
    sleep 10
    #exit
else
    echo "Reserve Fund Exists"
    #echo "arf:"$arf
    ebh=$(echo "$arf" | jq .[0].end_block_height | tr -d '"')
    resseq=$(echo "$arf" | jq .[0].reserve_sequence | tr -d '"')
    echo ""
    echo "reserve_sequence:"$resseq
    echo "end_block_height:"$ebh
    echo "current_block_height:"$(thetacli query status | jq .current_height | tr -d '"')
fi

#echo ""
#echo "Send Bob 1 TFuel(manually)."
#cmd='thetacli tx send --chain="privatenet" --async --from='$Alice' --to='$Bob' --theta=0 --tfuel='$tfuel' --seq='$ans
#if [ $do_echo -eq 1 ]; then echo $cmd; fi
##if [ $do_run -eq 1 ]; then eval $cmd; fi
#exit

#echo ""
#echo "Send Carol 1 TFuel(manually)."
#cmd='thetacli tx send --chain="privatenet" --async --from='$Alice' --to='$Carol' --theta=0 --tfuel='$tfuel' --seq='$ans
#if [ $do_echo -eq 1 ]; then echo $cmd; fi
##if [ $do_run -eq 1 ]; then eval $cmd; fi
#exit

aab=$(thetacli query account --address=$Alice | jq .coins.tfuelwei | tr -d '"')
bab=$(thetacli query account --address=$Bob | jq .coins.tfuelwei | tr -d '"')

echo ""
echo "Alice after balance: "$(un_wei $aab)
echo "Bob after balance: "$(un_wei $bab)


aib=$(thetacli query account --address=$Alice | jq .coins.tfuelwei | tr -d '"')
bib=$(thetacli query account --address=$Bob | jq .coins.tfuelwei | tr -d '"')
cib=$(thetacli query account --address=$Carol | jq .coins.tfuelwei | tr -d '"')

aseq=$(thetacli query account --address=$Alice | jq .sequence | tr -d '"')
ans=$(( $aseq + 1 ))

declare -a sigs

echo ""
echo "Begin off-chain signature generation."
echo ""

start=`date +%s`

i=0
while [ $i -lt $bobsigs ]; do
    echo "i:"$i
    payseq=$(( $startseq+$i+1 ))
    #cmd='sigout=$(./sp.sh --from='$Alice' --to='$Bob' --payment_seq='$i' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuel' | tail -n +6 | jq .source.signature | tr -d '"'"'"'"'"')'
    cmd='sigout=$(thetacli tx service_payment --chain="privatenet" --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuel' --pw=qwertyuiop | jq .source.signature | tr -d '"'"'"'"'"')'
    if [ $do_echo -eq 1 ]; then echo $cmd; fi
    if [ $do_run -eq 1 ]; then eval $cmd; fi
    sigs+=($sigout)
    let i++
done

i=0
while [ $i -lt $carolsigs ]; do
    echo "i:"$i
    payseq=$(( $startseq+$bobsigs+$i+1 ))
    #cmd='sigout=$(./sp.sh --from='$Alice' --to='$Carol' --payment_seq='$i' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuel' | tail -n +6 | jq .source.signature | tr -d '"'"'"'"'"')'
    cmd='sigout=$(thetacli tx service_payment --chain="privatenet" --from='$Alice' --to='$Carol' --payment_seq='$payseq' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuel' --pw=qwertyuiop | jq .source.signature | tr -d '"'"'"'"'"')'
    if [ $do_echo -eq 1 ]; then echo $cmd; fi
    if [ $do_run -eq 1 ]; then eval $cmd; fi
    sigs+=($sigout)
    let i++
done

for sig in "${sigs[@]}"
do
    echo "SIG:"$sig
done

echo ""
echo "End off-chain signature generation."
echo ""

end=`date +%s`
let deltatime=end-start
let hours=deltatime/3600
let minutes=(deltatime/60)%60
let seconds=deltatime%60
printf "Off-Chain Time spent: %d:%02d:%02d\n" $hours $minutes $seconds

start=`date +%s`

echo ""
echo "Begin on-chain service-payment transactions."
echo ""

i=0
while [ $i -lt $bobsigs ]; do
    #echo "i:"$i
    sig=${sigs[$i]}
    echo "Bob["$i"]:"$sig
    payseq=$(( $startseq+$i+1 ))
    #cmd='./sp.sh --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$ans' --resource_id='$rid' --tfuel='$tfuel' --on_chain --src_sig='$sig
    cmd='thetacli tx service_payment --chain="privatenet" --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuel' --pw=qwertyuiop --on_chain --src_sig='$sig
    if [ $do_echo -eq 1 ]; then echo $cmd; fi
    if [ $do_run -eq 1 ]; then eval $cmd; fi
    let i++
done

i=0
while [ $i -lt $carolsigs ]; do
    #echo "i:"$i
    sig=${sigs[$i+$bobsigs]}
    echo "Carol["$i+$bobsigs"]:"$sig
    payseq=$(( $startseq+$bobsigs+$i+1 ))
    #cmd='./sp.sh --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$ans' --resource_id='$rid' --tfuel='$tfuel' --on_chain --src_sig='$sig
    cmd='thetacli tx service_payment --chain="privatenet" --from='$Alice' --to='$Carol' --payment_seq='$payseq' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuel' --pw=qwertyuiop --on_chain --src_sig='$sig
    if [ $do_echo -eq 1 ]; then echo $cmd; fi
    if [ $do_run -eq 1 ]; then eval $cmd; fi
    let i++
done

echo ""
echo "End on-chain service-payment transactions."
echo ""

end=`date +%s`
let deltatime=end-start
let hours=deltatime/3600
let minutes=(deltatime/60)%60
let seconds=deltatime%60
printf "On-Chain Time spent: %d:%02d:%02d\n" $hours $minutes $seconds



aab=$(thetacli query account --address=$Alice | jq .coins.tfuelwei | tr -d '"')
bab=$(thetacli query account --address=$Bob | jq .coins.tfuelwei | tr -d '"')
cab=$(thetacli query account --address=$Carol | jq .coins.tfuelwei | tr -d '"')

echo ""
echo "Alice initial balance: "$(un_wei $aib)
echo "Alice final balance  : "$(un_wei $aab)
echo ""
echo "Bob initial balance  : "$(un_wei $bib)
echo "Bob final balance    : "$(un_wei $bab)
echo ""
echo "Carol initial balance: "$(un_wei $cib)
echo "Carol final balance  : "$(un_wei $cab)
echo ""
echo "Finished Batch Test."