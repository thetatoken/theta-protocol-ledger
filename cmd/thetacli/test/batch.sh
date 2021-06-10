#!/bin/bash
#set -x #echo on
do_run=1 # Execute(evaluate) the commands
#do_run=0 # Don't evaluate any commands
#do_echo=1 # Echo the commands
do_echo=0 # Don't echo any commands
do_echo_on_chain=1 # Echo the on-chain commands
#do_echo_on_chain=0 # Don't echo on-chain commands

Alice=2E833968E5bB786Ae419c4d13189fB081Cc43bab
Bob=70f587259738cB626A1720Af7038B8DcDb6a42a0
Carol=cd56123D0c5D6C1Ba4D39367b88cba61D93F5405
rid=rid1000001
rpduration=51
rfund=200
rcoll=201

tfuel=2

bobsigs=5
carolsigs=5

tfuelfee=0.3
tfuelperc=0.0
visaperc=0.0129
visaflat=0.05

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
echo "Getting Block Height-100 Timestamp."
cmd='pbh=$(echo "scale=2; '$cbh' - 100" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi

cmd='pbt=$(thetacli query block --height='$pbh' | jq .timestamp | tr -d '"'"'"'"'"')'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo "block-100_timestamp: "$pbt

# Last 100 block running average seconds per block
cmd='aspb=$(echo "scale=2; ('$cbt' - '$pbt')/100" | bc -q 2>/dev/null)'
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
echo "deltablocks: "$deltablocks

echo -n "          Current Date-Time : " ; date -r $cbt
#dis=$(date +%s)
cmd='expireatsecs=$(echo "scale=0; ('$cbt' + '$deltatime')/1" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo -n "Estimated Reserve Expiration: " ; date -r $expireatsecs

#curl -s https://explorer.thetatoken.org:8443/api/price/all | jq '.body[] | select(._id == "TFUEL") | .price'
tfuelperusd=$(curl -s https://explorer.thetatoken.org:8443/api/price/all | jq '.body[] | select(._id == "TFUEL") | .price')
echo "TFuel(USD): "$tfuelperusd

cmd='tfuelflat=$(echo "scale=2; (('$tfuelperusd' "'*'" '$tfuelfee') + 0.005)/1" | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi
echo "USD/Transaction: "$tfuelflat

# https://www.fool.com/the-ascent/research/average-credit-card-processing-fees-costs-america/
#Visa	1.29% + $0.05

#(((tfuel * tfuelperusd * visaperc) + visaflat) * (bobsigs + carlolsigs))

#((tfuel * tfuelperusd * tfuelperc) + tfuelflat) * (bobsigs + carlolsigs)

let totalitems=bobsigs+carolsigs

cmd='tfuelitemusd=$(echo "scale=2; (('$tfuel' "'*'" '$tfuelperusd') + 0.005)/1 " | bc -q 2>/dev/null)'
if [ $do_echo -eq 1 ]; then echo $cmd; fi
if [ $do_run -eq 1 ]; then eval $cmd; fi

cmd='totalvalue=$(echo "scale=2; (('$tfuelitemusd' "'*'" '$totalitems') + 0.005)/1 " | bc -q 2>/dev/null)'
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

#exit

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
    cmd='thetacli tx reserve --chain="privatenet" --async --from='$Alice' --fund='$rfund' --collateral='$rcoll' --duration='$rpduration' --resource_ids='$rid' --seq='$ans' --pw=qwertyuiop'
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
fi

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
    payseq=$(( $startseq+$i+1 ))
    echo "Bob["$payseq"]"
    #cmd='./sp.sh --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$ans' --resource_id='$rid' --tfuel='$tfuel' --on_chain --src_sig='$sig
    cmd='thetacli tx service_payment --chain="privatenet" --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuel' --pw=qwertyuiop --on_chain --src_sig='$sig
    if [ $do_echo -eq 1 ]; then echo $cmd; fi
    if [ $do_echo_on_chain -eq 1 ]; then
        echo $cmd
        echo ""
    else
        if [ $do_run -eq 1 ]; then eval $cmd; fi
    fi
    let i++
done

i=0
while [ $i -lt $carolsigs ]; do
    #echo "i:"$i
    sig=${sigs[$i+$bobsigs]}
    payseq=$(( $startseq+$bobsigs+$i+1 ))
    echo "Carol["$payseq"]"
    #cmd='./sp.sh --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$ans' --resource_id='$rid' --tfuel='$tfuel' --on_chain --src_sig='$sig
    cmd='thetacli tx service_payment --chain="privatenet" --from='$Alice' --to='$Carol' --payment_seq='$payseq' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuel' --pw=qwertyuiop --on_chain --src_sig='$sig
    if [ $do_echo -eq 1 ]; then echo $cmd; fi
    if [ $do_echo_on_chain -eq 1 ]; then
        echo $cmd
        echo ""
    else
        if [ $do_run -eq 1 ]; then eval $cmd; fi
    fi
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