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

rpduration=101

rfund=100
rcoll=101

# Cost tests performed at Thu Jun 10 13:17:06 EDT 2021
#tfuel=100.0
#Thu Jun 10 13:19:07 EDT 2021
#TFuel(USD): 0.54810887040421
#USD/Transaction: .16
#100.0 TFuel in USD = 54.8109 x 10 items = 548.1090 USD
#visacost USD: 7.57 to send 10 items (1.3800%)margin
#mn30cost USD: 1.60 to send 10 items (.2900%)margin
#
#tfuel=10.0
#Thu Jun 10 13:19:40 EDT 2021
#TFuel(USD): 0.54810887040421
#USD/Transaction: .16
#10.0 TFuel in USD = 5.4811 x 10 items = 54.8110 USD
#visacost USD: 1.21 to send 10 items (2.2000%)margin
#mn30cost USD: 1.60 to send 10 items (2.9100%)margin
#
#tfuel=1.0
#Thu Jun 10 13:20:36 EDT 2021
#TFuel(USD): 0.54810887040421
#USD/Transaction: .16
#1.0 TFuel in USD = .5481 x 10 items = 5.4810 USD
#visacost USD: .57 to send 10 items (10.3900%)margin
#mn30cost USD: 1.60 to send 10 items (29.1900%)margin
#
#tfuel=0.1
#Thu Jun 10 13:21:20 EDT 2021
#TFuel(USD): 0.54810887040421
#USD/Transaction: .16
#0.1 TFuel in USD = .0548 x 10 items = .5480 USD
#visacost USD: .51 to send 10 items (93.0600%)margin
#mn30cost USD: 1.60 to send 10 items (291.9700%)margin
#
#tfuel=0.01
#Thu Jun 10 13:22:17 EDT 2021
#TFuel(USD): 0.55011933917353
#USD/Transaction: .17
#0.01 TFuel in USD = .0055 x 10 items = .0550 USD
#visacost USD: .50 to send 10 items (909.0900%)margin
#mn30cost USD: 1.70 to send 10 items (3090.9000%)margin
#
# Now these are off-chain micropayment scenarios
# performed Fri Jun 11 11:56:06 EDT 2021
#
#Fri Jun 11 11:56:06 EDT 2021
#TFuel(USD): 0.48622932399761
#USD/Transaction: .15
#0.01 TFuel in USD = .0049 x 1000 items = 4.9000 USD
#visacost USD: 50.06 to send 1000 items (1021.6300%)margin
#mn30cost USD: 150.00 to send 1000 items (3061.2200%)margin
#mn30cost USD: .15 to send 1000 items (3.061200%)margin(1 service_payment)
#
#Fri Jun 11 11:58:38 EDT 2021
#TFuel(USD): 0.48603013940903
#USD/Transaction: .15
#0.001 TFuel in USD = .0005 x 10000 items = 5.0000 USD
#visacost USD: 500.06 to send 10000 items (10001.2000%)margin
#mn30cost USD: 1500.00 to send 10000 items (30000.0000%)margin
#mn30cost USD: .15 to send 10000 items (3.000000%)margin(1 service_payment)
#
# WSJ.com example : $38.99 for 4 weeks = $1.3925 per day = $4.1775 per 3 days
# Subscriber reads 10 articles over 3 days on average.  Each article Header + blurb free
# Once article clicked on.  First min charged, additional min/read charged as scrolled to : avg 5 mins/article
# 10 article * 5 min/read/article = 50 transactions over 3 days.
# If subsriber clicks on 0 articles over 3 days, no charges for that time
# Once user clicks on next article, new reserve fund is created for next 3 days
#
#Fri Jun 11 12:22:15 EDT 2021
#TFuel(USD): 0.48705293232693
#USD/Transaction: .15
#0.2 TFuel in USD = .0974 x 50 items = 4.8700 USD
#visacost USD: 2.56 to send 50 items (52.5600%)margin
#mn30cost USD: 7.50 to send 50 items (154.0000%)margin
#mn30cost USD: .15 to send 50 items (3.080000%)margin(1 service_payment)
#
# Daily Reader 3-4 articles/day
# 4.8700+0.15 x 10(3day periods/month) = $48.85/month 
#
# Weekend Reader
# 4.8700+0.15 x 4(3day periods/month) = $19.54/month
#
# Sporatic Reader : 2 articles/day on 10 days spread evenly across the month
# 0.963+0.15 x 10 = $9.78/month
# 
# Occasional Reader : 4 articles/month spread evenly across the month
# 0.4815+0.15 x 4 = $2.53/month
#
# Rare Reader : 1 article/month
# 0.4815+0.15 x 1 = $0.63/month
#
# Rare Reader Aborted Article : 1/5 article/month
# 0.0963+0.15 x 1 = $0.25/month


tfuel=0.2

# total 10 items
bobsigs=10
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
echo "deltablocks: "$deltablocks

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
tfuelamt=0
while [ $i -lt $bobsigs ]; do
    echo "Alice <=="$tfuel"== Bob"
    payseq=$(( $startseq+$i+1 ))
    if [ $accumulate -eq 1 ]; then
        cmd='tfuelamt=$(echo "scale=3; ('$tfuelamt' + '$tfuel')/1" | bc -q 2>/dev/null)'
        if [ $do_echo -eq 1 ]; then echo $cmd; fi
        if [ $do_run -eq 1 ]; then eval $cmd; fi
    else 
        tfuelamt=$tfuel 
    fi
    echo "Bob:"$payseq" = "$tfuelamt
    #cmd='sigout=$(./sp.sh --from='$Alice' --to='$Bob' --payment_seq='$i' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuel' | tail -n +6 | jq .source.signature | tr -d '"'"'"'"'"')'
    cmd='sigout=$(thetacli tx service_payment --chain="privatenet" --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuelamt' --pw=qwertyuiop | jq .source.signature | tr -d '"'"'"'"'"')'
    if [ $do_echo -eq 1 ]; then echo $cmd; fi
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
        if [ $do_run -eq 1 ]; then eval $cmd; fi
    else 
        tfuelamt=$tfuel 
    fi
    echo "Carol:"$payseq" = "$tfuelamt
    #cmd='sigout=$(./sp.sh --from='$Alice' --to='$Carol' --payment_seq='$i' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuel' | tail -n +6 | jq .source.signature | tr -d '"'"'"'"'"')'
    cmd='sigout=$(thetacli tx service_payment --chain="privatenet" --from='$Alice' --to='$Carol' --payment_seq='$payseq' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuelamt' --pw=qwertyuiop | jq .source.signature | tr -d '"'"'"'"'"')'
    if [ $do_echo -eq 1 ]; then echo $cmd; fi
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
tfuelamt=0
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
    if [ $accumulate -eq 1 ]; then
        if [ $i -lt $holdcnt ]; then
            echo "Hold::: Bob["$payseq"]"
        else
            echo "Submit: Bob["$payseq"]"
        fi
    else
        echo "Bob["$payseq"]"
    fi

    #cmd='./sp.sh --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$ans' --resource_id='$rid' --tfuel='$tfuel' --on_chain --src_sig='$sig
    cmd='thetacli tx service_payment --chain="privatenet" --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuelamt' --pw=qwertyuiop --on_chain --src_sig='$sig
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
tfuelamt=0
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
    if [ $accumulate -eq 1 ]; then
        if [ $i -lt $holdcnt ]; then
            echo "Hold::: Carol["$payseq"]"
        else
            echo "Submit: Carol["$payseq"]"
        fi
    else
        echo "Carol["$payseq"]"
    fi

    #cmd='./sp.sh --from='$Alice' --to='$Bob' --payment_seq='$payseq' --reserve_seq='$ans' --resource_id='$rid' --tfuel='$tfuel' --on_chain --src_sig='$sig
    cmd='thetacli tx service_payment --chain="privatenet" --from='$Alice' --to='$Carol' --payment_seq='$payseq' --reserve_seq='$resseq' --resource_id='$rid' --tfuel='$tfuelamt' --pw=qwertyuiop --on_chain --src_sig='$sig
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