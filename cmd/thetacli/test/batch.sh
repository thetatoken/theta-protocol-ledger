#!/bin/bash
#set -x #echo on
do_run=1 # Execute(evaluate) the commands
#do_run=0 # Don't evaluate any commands
#do_echo=1 # Echo the commands
do_echo=0 # Don't echo any commands
do_echo_off_chain=1 # Echo the on-chain commands
#do_echo_off_chain=0 # Don't echo, actually execute on-chain commands
do_echo_on_chain=1 # Echo the on-chain commands
#do_echo_on_chain=0 # Don't echo, actually execute on-chain commands
#calc_costs_only=1 # Only calculate the costs
calc_costs_only=0 # Calculate the costs then run tx

Alice=2E833968E5bB786Ae419c4d13189fB081Cc43bab
Bob=70f587259738cB626A1720Af7038B8DcDb6a42a0
Carol=cd56123D0c5D6C1Ba4D39367b88cba61D93F5405
rid=rid1000001

#rfdurationblocks=101       # Reserve Fund Duration in blocktimes : usualy (101 x 6secs) = 606 / 60 = 10.1 minutes
rfdurationblocks=30        # Reserve Fund Duration in blocktimes : Minimum for testing = 30 = 3 minues
#rfdurationsecs=0          # Reserve Fund Duration in seconds : 0 = Use rfdurationblocks instead
#let rfdurationsecs=120     # Reserve Fund Duration in seconds
let rfdurationsecs=10*60    # 10 minutes
#let rfdurationsecs=3*60*60 # 3 hours

rfund=1000
rcoll=1001

#tfuel=0.1
#tfuel=0.3
tfuel=20

# total 10 items
bobsigs=1
carolsigs=0

tfuelfee=0.3
tfuelperc=0.0
visaperc=0.0129
visaflat=0.05

startseq=0
accumulate=1

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
cmd='cbh=$(thetacli query status | jq .current_height | tr -d '"'"'"'"'"')'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo "current_block_height: "$cbh

echo ""
echo "Getting Current Block Height Timestamp."
#cmd='cat outbin/combined.json | jq '"'"'.contracts | ."'$solfile':'$contractName'" | .abi '"'"' > outbin/abi.json'
cmd='cbt=$(thetacli query block --height='$cbh' | jq .timestamp | tr -d '"'"'"'"'"')'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo "current_timestamp: "$cbt

echo ""
echo "Getting Block Height-10 Timestamp."
cmd='pbh=$(echo "scale=2; '$cbh' - 10" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi

cmd='pbt=$(thetacli query block --height='$pbh' | jq .timestamp | tr -d '"'"'"'"'"')'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo "block-10_timestamp: "$pbt

# Last 10 block running average seconds per block
cmd='aspb=$(echo "scale=2; ('$cbt' - '$pbt')/10" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo "average seconds per block: "$aspb

#// github.com/thetatoken/theta/ledger/types/const.go : Line 85
#	// MaximumFundReserveDuration indicates the maximum duration (in terms of number of blocks) of reserving fund
#    MaximumFundReserveDuration uint64 = 12 * 3600
#
#	// MinimumFundReserveDuration indicates the minimum duration (in terms of number of blocks) of reserving fund
#	MinimumFundReserveDuration uint64 = 300
xfdb=43200
#nfdb=300  My privatenet is overriden to 30 blocks for testing.
nfdb=30

let xffrb=$cbh+xfdb  # maXimum Future Fund Reserverve Block
let nffrb=$cbh+nfdb  # miNimum Future Fund Reserverve Block

cmd='deltatime=$(echo "scale=0; ('$aspb' "'*'" '$xfdb')/1" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo "deltatime: "$deltatime

let hours=deltatime/3600
let minutes=(deltatime/60)%60
let seconds=deltatime%60
printf "Max time until Reserve Deposit Expiration: %d:%02d:%02d\n" $hours $minutes $seconds

cmd='deltatime=$(echo "scale=0; ('$aspb' "'*'" '$nfdb')/1" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo "deltatime: "$deltatime

let hours=deltatime/3600
let minutes=(deltatime/60)%60
let seconds=deltatime%60
printf "Min time until Reserve Deposit Expiration: %d:%02d:%02d\n" $hours $minutes $seconds

let moviesecs=120*60 # 90mins + 30 for pausing to pee and make popcorn
cmd='deltablocks=$(echo "scale=0; ('$moviesecs' / '$aspb')/1" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo "deltablocks: "$deltablocks" to watch a movie."

echo -n "          Current Date-Time : " ; date -r $cbt
#dis=$(date +%s)
cmd='expireatsecs=$(echo "scale=0; ('$cbt' + '$deltatime')/1" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo -n "Estimated Reserve Expiration: " ; date -r $expireatsecs

echo ""

date

#https://github.com/thetatoken/theta-infrastructure-ledger-explorer/blob/master/docs/api.md
#curl -s https://explorer.thetatoken.org:8443/api/price/all | jq '.body[] | select(._id == "TFUEL") | .price'
tfuelperusd=$(curl -s https://explorer.thetatoken.org:8443/api/price/all | jq '.body[] | select(._id == "TFUEL") | .price')
echo "TFuel(USD): "$tfuelperusd

#curl -s curl https://explorer.thetatoken.org:8443/api/supply/tfuel | jq .


cmd='tfuelflat=$(echo "scale=2; (('$tfuelperusd' "'*'" '$tfuelfee') + 0.005)/1" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo "USD/Transaction: "$tfuelflat

# https://www.fool.com/the-ascent/research/average-credit-card-processing-fees-costs-america/
#Visa	1.29% + $0.05

#(((tfuel * tfuelperusd * visaperc) + visaflat) * (bobsigs + carlolsigs))

#((tfuel * tfuelperusd * tfuelperc) + tfuelflat) * (bobsigs + carlolsigs)

let totalitems=bobsigs+carolsigs

cmd='tfuelitemusd=$(echo "scale=4; (('$tfuel' "'*'" '$tfuelperusd') + 0.00005)/1 " | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi

cmd='totalvalue=$(echo "scale=4; (('$tfuelitemusd' "'*'" '$totalitems') + 0.00005)/1 " | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo $tfuel" TFuel in USD = "$tfuelitemusd" x "$totalitems" items = "$totalvalue" USD"


cmd='visacost=$(echo "scale=2; (((('$tfuel' "'*'" '$tfuelperusd' "'*'" '$visaperc') + '$visaflat') "'*'" '$totalitems') + 0.005)/1 " | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
cmd='visamargin=$(echo "scale=4; (('$visacost' / '$totalvalue') "'*'" 100)" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo "visacost USD: "$visacost" to send "$totalitems" items ("$visamargin"%)margin"

cmd='mn30cost=$(echo "scale=2; (((('$tfuel' "'*'" '$tfuelperusd' "'*'" '$tfuelperc') + '$tfuelflat') "'*'" '$totalitems') + 0.005)/1 " | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
cmd='mn30margin=$(echo "scale=4; (('$mn30cost' / '$totalvalue') "'*'" 100)" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo "mn30cost USD: "$mn30cost" to send "$totalitems" items ("$mn30margin"%)margin"

cmd='mn30cost=$(echo "scale=2; (((('$tfuel' "'*'" '$tfuelperusd' "'*'" '$tfuelperc') "'*'" '$totalitems') + '$tfuelflat') + 0.005)/1 " | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
cmd='mn30margin=$(echo "scale=6; (('$mn30cost' / '$totalvalue') "'*'" 100)" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo "mn30cost USD: "$mn30cost" to send "$totalitems" items ("$mn30margin"%)margin(1 service_payment)"

if [ $calc_costs_only -eq 1 ]; then echo ""; exit; fi

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

#echo ""
#echo "Send Bob 1 TFuel(manually)."
#cmd='thetacli tx send --chain="privatenet" --async --from='$Alice' --to='$Bob' --theta=0 --tfuel=1 --seq='$ans
#if [ $do_echo -eq 1 ]; then echo $cmd; fi
#if [ $do_run -eq 1 ]; then eval $cmd; fi
#exit

#echo ""
#echo "Send Carol 1 TFuel(manually)."
#cmd='thetacli tx send --chain="privatenet" --async --from='$Alice' --to='$Carol' --theta=0 --tfuel=1 --seq='$ans
#if [ $do_echo -eq 1 ]; then echo $cmd; fi
#if [ $do_run -eq 1 ]; then eval $cmd; fi
#exit

echo ""
echo "Check for existing reserve fund."
cmd='arf=$(thetacli query account --address='$Alice' | jq .reserved_funds)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi

#echo "arf:"${#arf}

if [ ${#arf} == 2 ]; then
    echo "No Reserve Fund"
    echo "Alice begin create reserve."
    if [ $rfdurationsecs -eq 0 ]; then
        echo "Using "$rfdurationblocks" blocks"
    else
        cmd='rfdurationblocks=$(echo "scale=0; ('$rfdurationsecs' / '$aspb')/1" | bc -q 2>/dev/null)'
        if [ $do_echo -eq 1 ]; then echo $cmd; fi
        if [ $do_run -eq 1 ]; then eval $cmd; fi
        echo "Using: "$rfdurationblocks" blocks = "$rfdurationsecs"secs"
    fi
    cmd='thetacli tx reserve --chain="privatenet" --async --from='$Alice' --fund='$rfund' --collateral='$rcoll' --duration='$rfdurationblocks' --resource_ids='$rid' --seq='$ans' --password=qwertyuiop'
    if [ $do_echo -eq 1 ]; then echo $cmd; fi
    if [ $do_run -eq 1 ]; then eval $cmd; fi
    echo "Alice end create reserve.  Wait 15 seconds and rereun."
    echo "Sleeping..."
    sleep 15
    exit
else
    echo "Reserve Fund Exists"
    #echo "arf:"$arf
    ebh=$(echo "$arf" | jq .[0].end_block_height | tr -d '"')
    resseq=$(echo "$arf" | jq .[0].reserve_sequence | tr -d '"')
    echo ""
    echo "reserve_sequence:"$resseq
    echo "end_block_height:"$ebh
    echo "current_block_height:"$(thetacli query status | jq .current_height | tr -d '"')
    # Find the largest existing transfer_record sequence number
    # thetacli query account --address=2E833968E5bB786Ae419c4d13189fB081Cc43bab | jq .reserved_funds[0].transfer_records[-1].service_payment.payment_sequence | tr -d '"'
    cmd='txlen=$(thetacli query account --address='$Alice' | jq .reserved_funds[0].transfer_records)'
    if [ $do_echo -eq 1 ]; then echo $cmd; fi
    if [ $do_run -eq 1 ]; then eval $cmd; fi
    if [ ${#txlen} == 2 ]; then
        echo "No Transfer Records exit yet."
    else 
        echo "Transfer Records exit."
        cmd='letxseq=$(thetacli query account --address='$Alice' | jq .reserved_funds[0].transfer_records[-1].service_payment.payment_sequence | tr -d '"'"'"'"'"')'
        if [ $do_echo -eq 1 ]; then echo $cmd; fi
        if [ $do_run -eq 1 ]; then eval $cmd; fi
        startseq=$letxseq
        echo "Setting startseq = "$letxseq
    fi
fi
aib=$(thetacli query account --address=$Alice | jq .coins.tfuelwei | tr -d '"')
bib=$(thetacli query account --address=$Bob | jq .coins.tfuelwei | tr -d '"')
cib=$(thetacli query account --address=$Carol | jq .coins.tfuelwei | tr -d '"')

echo ""
echo "Alice init balance: "$(un_wei $aib)
echo "Bob init balance: "$(un_wei $bib)
echo "Carol init balance: "$(un_wei $cib)


aseq=$(thetacli query account --address=$Alice | jq .sequence | tr -d '"')
ans=$(( $aseq + 1 ))

declare -a sigs

echo ""
echo "Begin off-chain signature generation."
echo ""

offstart=`date +%s`

i=0
tfuelamt=0
while [ $i -lt $bobsigs ]; do
    echo "Alice <=="$tfuel"== Bob"
    payseq=$(( $startseq+$i+1 ))
    if [ $accumulate -eq 1 ]; then
        cmd='tfuelamt=$(echo "scale=3; ('$tfuelamt' + '$tfuel')/1" | bc -q 2>/dev/null)'
        if [ $do_echo -eq 1 ]; then echo $cmd; fi
    	if [ $do_echo_off_chain -eq 1 ]; then echo $cmd; echo ""; fi
        if [ $do_run -eq 1 ]; then eval $cmd; fi
    else 
        tfuelamt=$tfuel 
    fi
    echo "Bob:"$payseq" = "$tfuelamt
    #cmd='sigout=$(./sp.sh --from='$Alice' --to='$Bob' --payment_seq='$i' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuel' | tail -n +6 | jq .source.signature | tr -d '"'"'"'"'"')'
    cmd='sigout=$(thetacli tx service_payment --chain="privatenet" --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuelamt' --password=qwertyuiop | jq .source.signature | tr -d '"'"'"'"'"')'
    if [ $do_echo -eq 1 ]; then echo $cmd; fi
    if [ $do_echo_off_chain -eq 1 ]; then echo $cmd; echo ""; fi
    if [ $do_run -eq 1 ]; then eval $cmd; fi
    sigs+=($sigout)
    echo ""
    let i++
done

i=0
tfuelamt=0
while [ $i -lt $carolsigs ]; do
    echo "Alice <=="$tfuel"== Carol"
    payseq=$(( $startseq+$bobsigs+$i+1 ))
    if [ $accumulate -eq 1 ]; then
        cmd='tfuelamt=$(echo "scale=3; ('$tfuelamt' + '$tfuel')/1" | bc -q 2>/dev/null)'
        if [ $do_echo -eq 1 ]; then echo $cmd; fi
    	if [ $do_echo_off_chain -eq 1 ]; then echo $cmd; echo ""; fi
        if [ $do_run -eq 1 ]; then eval $cmd; fi
    else 
        tfuelamt=$tfuel 
    fi
    echo "Carol:"$payseq" = "$tfuelamt
    #cmd='sigout=$(./sp.sh --from='$Alice' --to='$Bob' --payment_seq='$i' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuel' | tail -n +6 | jq .source.signature | tr -d '"'"'"'"'"')'
    cmd='sigout=$(thetacli tx service_payment --chain="privatenet" --from='$Alice' --to='$Carol' --payment_seq='$payseq' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuelamt' --password=qwertyuiop | jq .source.signature | tr -d '"'"'"'"'"')'
    if [ $do_echo -eq 1 ]; then echo $cmd; fi
    if [ $do_echo_off_chain -eq 1 ]; then echo $cmd; echo ""; fi
    if [ $do_run -eq 1 ]; then eval $cmd; fi
    sigs+=($sigout)
    echo ""
    let i++
done

#for sig in "${sigs[@]}"
#do
#    echo "SIG:"$sig
#done

echo ""
echo "End off-chain signature generation."
echo ""

offend=`date +%s`
let deltatime=offend-offstart
let hours=deltatime/3600
let minutes=(deltatime/60)%60
let seconds=deltatime%60
cmd='secspertx=$(echo "scale=3; ('$deltatime' / ('$bobsigs' + '$carolsigs'))/1" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
printf "Off-Chain Time spent: %d:%02d:%02d (%2.5f sec/tx)\n" $hours $minutes $seconds $secspertx

onstart=`date +%s`

echo ""
echo "Begin on-chain service-payment transactions."
echo ""

i=0
tfuelamt=0
baccum=0
let holdcnt=bobsigs-1
while [ $i -lt $bobsigs ]; do
    #echo "i:"$i
    sig=${sigs[$i]}
    payseq=$(( $startseq+$i+1 ))
    if [ $accumulate -eq 1 ]; then
        cmd='tfuelamt=$(echo "scale=3; ('$tfuelamt' + '$tfuel')/1" | bc -q 2>/dev/null)'
        if [ $do_echo -eq 1 ]; then echo $cmd; fi
        if [ $do_run -eq 1 ]; then eval $cmd; fi
    else 
        tfuelamt=$tfuel 
    fi

    #cmd='./sp.sh --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$ans' --resource_id='$rid' --tfuel='$tfuel' --on_chain --src_sig='$sig
    cmd='sphash=$(thetacli tx service_payment --chain="privatenet" --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuelamt' --password=qwertyuiop --on_chain --src_sig='$sig' | jq .hash)'

    if [ $accumulate -eq 1 ]; then
        if [ $i -lt $holdcnt ]; then
            echo "Hold::: Bob["$payseq"]"
            if [ $do_echo -eq 1 ]; then echo $cmd; fi
            if [ $do_echo_on_chain -eq 1 ]; then echo $cmd; echo ""; fi
        else
            echo "Submit: Bob["$payseq"]"
            let baccum++
            if [ $do_echo -eq 1 ]; then echo $cmd; fi
            if [ $do_echo_on_chain -eq 1 ]; then echo $cmd; echo ""; else if [ $do_run -eq 1 ]; then eval $cmd; echo $sphash; fi fi
       fi
    else
        echo "Bob["$payseq"]"
        if [ $do_echo -eq 1 ]; then echo $cmd; fi
        if [ $do_echo_on_chain -eq 1 ]; then echo $cmd; echo ""; else if [ $do_run -eq 1 ]; then eval $cmd; fi fi
    fi

    let i++
done

i=0
tfuelamt=0
caccum=0
let holdcnt=carolsigs-1
while [ $i -lt $carolsigs ]; do
    #echo "i:"$i
    sig=${sigs[$i+$bobsigs]}
    payseq=$(( $startseq+$bobsigs+$i+1 ))
    if [ $accumulate -eq 1 ]; then
        cmd='tfuelamt=$(echo "scale=3; ('$tfuelamt' + '$tfuel')/1" | bc -q 2>/dev/null)'
        if [ $do_echo -eq 1 ]; then echo $cmd; fi
        if [ $do_run -eq 1 ]; then eval $cmd; fi
    else 
        tfuelamt=$tfuel 
    fi

    #cmd='./sp.sh --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$ans' --resource_id='$rid' --tfuel='$tfuel' --on_chain --src_sig='$sig
    cmd='sphash=$(thetacli tx service_payment --chain="privatenet" --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuelamt' --password=qwertyuiop --on_chain --src_sig='$sig' | jq .hash)'

    if [ $accumulate -eq 1 ]; then
        if [ $i -lt $holdcnt ]; then
            echo "Hold::: Carol["$payseq"]"
            if [ $do_echo -eq 1 ]; then echo $cmd; fi
            if [ $do_echo_on_chain -eq 1 ]; then echo $cmd; echo ""; fi
        else
            echo "Submit: Carol["$payseq"]"
            let caccum++
            if [ $do_echo -eq 1 ]; then echo $cmd; fi
            if [ $do_echo_on_chain -eq 1 ]; then echo $cmd; echo ""; else if [ $do_run -eq 1 ]; then eval $cmd; echo $sphash; fi fi
       fi
    else
        echo "Carol["$payseq"]"
        if [ $do_echo -eq 1 ]; then echo $cmd; fi
        if [ $do_echo_on_chain -eq 1 ]; then echo $cmd; echo ""; else if [ $do_run -eq 1 ]; then eval $cmd; fi fi
    fi

    let i++
done

echo ""
echo "End on-chain service-payment transactions."
echo ""

onend=`date +%s`
let deltatime=onend-onstart
let hours=deltatime/3600
let minutes=(deltatime/60)%60
let seconds=deltatime%60
cmd='secspertx=$(echo "scale=3; ('$deltatime' / ('$baccum' + '$caccum'))/1" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
printf "On-Chain Time spent: %d:%02d:%02d (%2.5f sec/tx)\n" $hours $minutes $seconds $secspertx

let deltatime=onend-offstart
let hours=deltatime/3600
let minutes=(deltatime/60)%60
let seconds=deltatime%60
cmd='secspertx=$(echo "scale=3; ('$deltatime' / ('$baccum' + '$caccum' + '$bobsigs' + '$carolsigs'))/1" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
printf "Total TX Time spent: %d:%02d:%02d (%2.5f sec/tx)\n" $hours $minutes $seconds $secspertx


aab=$(thetacli query account --address=$Alice | jq .coins.tfuelwei | tr -d '"')
bab=$(thetacli query account --address=$Bob | jq .coins.tfuelwei | tr -d '"')
cax=$(thetacli query account --address=$Carol | jq .coins.tfuelwei | tr -d '"')

echo ""
echo "Alice initial balance: "$(un_wei $aib)
echo "Alice final balance  : "$(un_wei $aab)
echo ""
echo "Bob initial balance  : "$(un_wei $bib)
echo "Bob final balance    : "$(un_wei $bab)
echo ""
echo "Carol initial balance: "$(un_wei $cib)
echo "Carol final balance  : "$(un_wei $cax)
echo ""
echo "Finished Batch Test."
